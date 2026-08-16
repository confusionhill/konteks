package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dika/contextkit/backend/internal/config"
	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/dika/contextkit/backend/internal/handler"
	"github.com/dika/contextkit/backend/internal/repository"
	"github.com/dika/contextkit/backend/internal/service"
	"github.com/dika/contextkit/backend/internal/usecase"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

func runMigrations(db *sqlx.DB, filepath string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	_, err = db.Exec(string(content))
	if err != nil {
		return fmt.Errorf("failed to execute migration script: %w", err)
	}

	return nil
}

func main() {
	// 1. Load Configurations
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

	// 3. Programmatically Apply Consolidated Database Schema & Seed Data
	migrationFiles := []string{
		"db/migrations/01_init.sql",
		"db/migrations/02_add_encryption_fields.sql",
		"db/migrations/03_add_evaluation_encryption_fields.sql",
		"db/migrations/04_add_room_description_and_parent.sql",
		"db/migrations/05_add_artifacts.sql",
		"db/migrations/07_add_context_health_settings.sql",
		"db/migrations/08_optional_embedding.sql",
	}
	for _, migrationFile := range migrationFiles {
		log.Printf("applying database migrations from %s...", migrationFile)
		if err := runMigrations(db, migrationFile); err != nil {
			log.Printf("migrations warning (%s): %v", migrationFile, err)
		} else {
			log.Printf("database migration %s applied successfully.", migrationFile)
		}
	}

	// 4. Initialize Services, Clients & Vector Store
	var vectorStore domain.VectorStore
	if cfg.VectorProvider == "cloudflare" {
		vectorStore = repository.NewCloudflareVectorizeStore(cfg)
		log.Println("[VectorStore]: Initialized Cloudflare Vectorize provider.")
	} else {
		vectorStore = repository.NewPgvectorStore(db)
		log.Println("[VectorStore]: Initialized pgvector provider.")
	}

	// 5. Initialize Core Repositories
	authRepo := repository.NewAuthRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	memoryRepo := repository.NewMemoryRepository(db, vectorStore)
	eventRepo := repository.NewEventRepository(db)
	usageRepo := repository.NewUsageRepository(db)
	evalRepo := repository.NewEvaluationRepository(db)
	artifactRepo := repository.NewArtifactRepository(db, vectorStore)
	healthRepo := repository.NewContextHealthRepository(db)

	openRouterClient := service.NewOpenRouterServiceClient(cfg)
	cryptoService := service.NewCryptoService()
	storageProvider, err := service.NewS3StorageProvider(cfg)
	if err != nil {
		log.Fatalf("failed to initialize S3 storage provider: %v", err)
	}
	if err := storageProvider.EnsureBucketExists(context.Background()); err != nil {
		log.Printf("[Warning]: failed to ensure S3 bucket on startup: %v", err)
	}

	// 6. Initialize Business Logic Usecases
	artifactUsecase := usecase.NewArtifactUsecase(artifactRepo, openRouterClient, storageProvider)
	roomUsecase := usecase.NewRoomUsecase(roomRepo)
	messageUsecaseImpl := usecase.NewMessageUsecase(roomRepo, messageRepo, memoryRepo, openRouterClient, storageProvider, cryptoService, cfg)
	contextUsecase := usecase.NewContextUsecase(roomRepo, messageRepo, memoryRepo, openRouterClient, storageProvider, evalRepo, cryptoService, cfg, artifactUsecase)
	eventUsecase := usecase.NewEventUsecase(roomRepo, eventRepo)
	usageUsecase := usecase.NewUsageUsecase(usageRepo)
	healthUsecase := usecase.NewContextHealthUsecase(healthRepo, roomRepo, openRouterClient)

	messageUsecase := messageUsecaseImpl

	// 7. Initialize Echo REST API Handlers
	eventBroker := handler.NewEventBroker()
	authMiddleware := handler.AuthMiddleware(authRepo, cfg.JWTSecret)

	authHandler := handler.NewAuthHandler(authRepo, cfg.JWTSecret)
	projectHandler := handler.NewProjectHandler(authRepo, cfg.JWTSecret)
	roomHandler := handler.NewRoomHandler(roomUsecase)
	messageHandler := handler.NewMessageHandler(messageUsecase)
	artifactHandler := handler.NewArtifactHandler(artifactUsecase)
	contextHandler := handler.NewContextHandler(contextUsecase)
	eventHandler := handler.NewEventHandler(eventUsecase, eventBroker)
	dashboardHandler := handler.NewDashboardHandler(usageUsecase, roomUsecase, messageUsecase, roomRepo, storageProvider, evalRepo)
	healthHandler := handler.NewContextHealthHandler(healthUsecase)

	// 8. Initialize Echo HTTP Router
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-Project-ID", "X-API-Key", "X-Encryption-Key"},
	}))

	// Liveness Probe Check
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Public Auth Routes
	v1Auth := e.Group("/v1/auth")
	v1Auth.POST("/register", authHandler.Register)
	v1Auth.POST("/login", authHandler.Login)

	// Protected Routes V1 (User JWT or Project API Key Required)
	v1 := e.Group("/v1")
	v1.Use(authMiddleware)
	v1.Use(handler.UsageTrackingMiddleware(usageUsecase))

	// Auth Me
	v1.GET("/auth/me", authHandler.Me)

	// Project & API Key Management
	v1.GET("/projects", projectHandler.ListProjects)
	v1.POST("/projects", projectHandler.CreateProject)
	v1.DELETE("/projects/:project_id", projectHandler.DeleteProject)
	v1.GET("/projects/:project_id/api-keys", projectHandler.ListAPIKeys)
	v1.POST("/projects/:project_id/api-keys", projectHandler.CreateAPIKey)
	v1.DELETE("/projects/:project_id/api-keys/:key_id", projectHandler.RevokeAPIKey)
	v1.GET("/projects/:project_id/context-health-policy", healthHandler.GetPolicy)
	v1.PUT("/projects/:project_id/context-health-policy", healthHandler.UpdatePolicy)

	// Artifacts & Core Memory API
	v1.POST("/projects/:project_id/artifacts", artifactHandler.CreateArtifact)
	v1.GET("/projects/:project_id/artifacts", artifactHandler.ListArtifacts)
	v1.GET("/projects/:project_id/artifacts/graph", artifactHandler.GetKnowledgeGraph)
	v1.GET("/projects/:project_id/artifacts/:artifact_id", artifactHandler.GetArtifact)
	v1.PUT("/projects/:project_id/artifacts/:artifact_id", artifactHandler.UpdateArtifact)
	v1.DELETE("/projects/:project_id/artifacts/:artifact_id", artifactHandler.DeleteArtifact)

	// Rooms API & Blacklists
	v1.POST("/rooms", roomHandler.CreateRoom)
	v1.GET("/rooms/:room_id", roomHandler.GetRoomState)
	v1.GET("/rooms/:room_id/children", roomHandler.GetChildRooms)
	v1.PATCH("/rooms/:room_id", roomHandler.UpdateRoom)
	v1.DELETE("/rooms/:room_id", roomHandler.DeleteRoom)
	v1.POST("/rooms/:room_id/warmup", contextHandler.WarmupRoom)
	v1.POST("/rooms/:room_id/blacklist", artifactHandler.AddRoomBlacklist)
	v1.GET("/rooms/:room_id/blacklist", artifactHandler.GetRoomBlacklists)
	v1.DELETE("/rooms/:room_id/blacklist/:blacklist_id", artifactHandler.RemoveRoomBlacklist)
	v1.GET("/rooms/:room_id/memories", healthHandler.ListRoomMemories)
	v1.POST("/rooms/:room_id/context-health/clean", healthHandler.CleanRoom)

	// Memories Restoration API
	v1.POST("/memories/:memory_id/restore", healthHandler.RestoreMemory)

	// Messages API
	v1.POST("/rooms/:room_id/messages", messageHandler.CreateMessage)
	v1.GET("/rooms/:room_id/messages", messageHandler.ListMessages)

	// Context Assembly API
	v1.POST("/rooms/:room_id/context", contextHandler.GetContextPackage)

	// Events & SSE Logging API
	v1.POST("/rooms/:room_id/events", eventHandler.LogEvent)
	v1.GET("/rooms/:room_id/events", eventHandler.StreamEvents)

	// Dashboard API
	v1.GET("/dashboard/summary", dashboardHandler.GetSummary)
	v1.GET("/dashboard/rooms", dashboardHandler.ListRooms)
	v1.GET("/dashboard/rooms/:room_id", dashboardHandler.GetRoomDetail)
	v1.GET("/dashboard/rooms/:room_id/archive-url", dashboardHandler.GetArchiveURL)
	v1.GET("/dashboard/logs", dashboardHandler.GetProjectLogs)
	v1.GET("/evaluations", dashboardHandler.GetProjectEvaluations)
	v1.GET("/evaluations/summary", dashboardHandler.GetEvaluationsSummary)
	v1.GET("/evaluations/:eval_id", dashboardHandler.GetEvaluationDetail)
	v1.GET("/rooms/:room_id/evaluations", dashboardHandler.GetRoomEvaluations)

	// 9. Bind Listeners & Listen
	serverPort := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("konteks backend listening on port %s...", serverPort)
	if err := e.Start(serverPort); err != nil {
		log.Fatalf("server listener failure: %v", err)
	}
}
