package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dika/contextkit/backend/internal/config"
	"github.com/dika/contextkit/backend/internal/mcp"
	"github.com/dika/contextkit/backend/internal/repository"
	"github.com/dika/contextkit/backend/internal/service"
	"github.com/dika/contextkit/backend/internal/worker"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	transport := flag.String("transport", os.Getenv("MCP_TRANSPORT"), "transport type: stdio (default) or http")
	apiKeyFlag := flag.String("api-key", "", "ContextKit project API key (overrides CONTEXTKIT_API_KEY)")
	portFlag := flag.Int("port", 0, "HTTP bind port (overrides MCP_PORT, default 8099)")
	flag.Parse()

	if *transport == "" {
		*transport = "stdio"
	}

	// 1. Load Configurations (secrets/config.json with env overrides)
	cfgPath := "secrets/config.json"
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("failed to load configuration from %s: %v", cfgPath, err)
	}

	// 2. Establish Database Connection (with retry loop for Docker DB startup)
	var db *sqlx.DB
	for i := 1; i <= 10; i++ {
		db, err = sqlx.Connect("postgres", cfg.DatabaseURL)
		if err == nil {
			break
		}
		log.Printf("[Attempt %d/10] waiting for postgres database to become ready...", i)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	// 3. Resolve the project API key and validate it against the api_keys table.
	apiKey := *apiKeyFlag
	if apiKey == "" {
		apiKey = os.Getenv("CONTEXTKIT_API_KEY")
	}
	if apiKey == "" {
		log.Fatalf("an API key is required: set CONTEXTKIT_API_KEY or pass --api-key (use a key from the contextkit build UI or the seeded dev key ck_live_dev_test_key_123)")
	}

	authProvider := mcp.NewAuthProvider(repository.NewMCPAuthRepository(db))
	auth, err := authProvider.Authenticate(context.Background(), apiKey)
	if err != nil {
		log.Fatalf("failed to validate API key: %v", err)
	}
	log.Printf("[Auth]: authenticated project %s", auth.ProjectID)

	// 4. Wire the fresh repository and service layers.
	roomRepo := repository.NewMCPRoomRepository(db)
	stateRepo := repository.NewMCPStateRepository(db)
	messageRepo := repository.NewMCPMessageRepository(db)
	memoryRepo := repository.NewMCPMemoryRepository(db)

	openRouterClient := service.NewOpenRouterServiceClient(cfg)
	embeddingWorker := worker.NewMCPEmbeddingWorker(openRouterClient, memoryRepo, 5000)
	embeddingWorker.Start(5)

	roomService := service.NewMCPRoomService(roomRepo, stateRepo)
	contextService := service.NewMCPContextService(roomRepo, stateRepo, messageRepo, memoryRepo, openRouterClient)
	memoryService := service.NewMCPMemoryService(roomRepo, memoryRepo, openRouterClient, embeddingWorker)

	logger := newServerLogger()
	mcpServer := mcp.NewServer(mcp.Deps{
		AuthProvider:     authProvider,
		DefaultAuth:      auth,
		RoomService:      roomService,
		ContextService:   contextService,
		MemoryService:    memoryService,
		EmbeddingWorker:  embeddingWorker,
		Logger:           logger,
	})

	log.Printf("[MCP Server]: ContextKit MCP server ready (transport=%s)", *transport)

	if *transport == "http" {
		runHTTPServer(mcpServer, authProvider, logger, *portFlag)
		return
	}

	if err := server.ServeStdio(mcpServer); err != nil {
		log.Printf("[MCP Server]: stdio server exited: %v", err)
	}
	embeddingWorker.Close()
}

// newServerLogger builds the structured logger used for request/observability
// logging. Records are JSON lines on stdout so docker logs can be searched and
// shipped to a log aggregator. The log level can be raised with MCP_LOG_LEVEL
// (DEBUG, INFO, WARN, ERROR).
func newServerLogger() *slog.Logger {
	level := slog.LevelInfo
	if lvl := os.Getenv("MCP_LOG_LEVEL"); lvl != "" {
		var parsed slog.Level
		if err := parsed.UnmarshalText([]byte(lvl)); err == nil {
			level = parsed
		}
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func runHTTPServer(mcpServer *server.MCPServer, authProvider *mcp.AuthProvider, logger *slog.Logger, port int) {
	if port == 0 {
		if envPort := os.Getenv("MCP_PORT"); envPort != "" {
			fmt.Sscanf(envPort, "%d", &port)
		}
	}
	if port == 0 {
		port = 8099
	}

	httpServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			ctx = mcp.WithRequestLogger(ctx, logger)
			rawKey := r.Header.Get("X-API-Key")
			if rawKey == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
					rawKey = strings.TrimSpace(authHeader[len("Bearer "):])
				}
			}
			if rawKey == "" {
				// Query-param credentials support MCP clients that cannot set
				// headers, such as Claude Desktop / claude.ai remote servers
				// (e.g. https://mcp.contextkit.app/mcp?api_key=ck_...).
				queryKey := r.URL.Query().Get("api_key")
				if queryKey == "" {
					queryKey = r.URL.Query().Get("token")
				}
				rawKey = queryKey
			}
			auth, err := authProvider.Authenticate(ctx, rawKey)
			if err != nil {
				return mcp.WithAuthError(ctx, err)
			}
			return mcp.WithAuth(ctx, auth)
		}),
	)

	addr := fmt.Sprintf(":%d", port)
	errCh := make(chan error, 1)
	go func() {
		log.Printf("[MCP Server]: HTTP listening on %s/mcp", addr)
		errCh <- httpServer.Start(addr)
	}()

	// 5. Graceful shutdown on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("[MCP Server]: HTTP server failure: %v", err)
		}
	case <-sigCh:
		log.Println("[MCP Server]: shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("[MCP Server]: graceful shutdown error: %v", err)
		}
	}
}