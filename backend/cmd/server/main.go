package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dika/contextkit/backend/internal/config"
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

	// 3. Programmatically Apply Database Schema Migrations
	migrationFile := "db/migrations/000001_init.up.sql"
	log.Printf("applying database migrations from %s...", migrationFile)
	if err := runMigrations(db, migrationFile); err != nil {
		log.Printf("migrations warning (continuing execution): %v", err)
	} else {
		log.Println("database migrations applied successfully.")
	}

	migrationFile2 := "db/migrations/000002_usage_metering.up.sql"
	log.Printf("applying database migrations from %s...", migrationFile2)
	if err := runMigrations(db, migrationFile2); err != nil {
		log.Printf("migrations warning (continuing execution): %v", err)
	} else {
		log.Println("usage metering migration applied successfully.")
	}

	migrationFile3 := "db/migrations/000003_token_cost_tracking.up.sql"
	log.Printf("applying database migrations from %s...", migrationFile3)
	if err := runMigrations(db, migrationFile3); err != nil {
		log.Printf("migrations warning (continuing execution): %v", err)
	} else {
		log.Println("token & cost tracking migration applied successfully.")
	}

	migrationFile4 := "db/migrations/000004_room_state_references.up.sql"
	log.Printf("applying database migrations from %s...", migrationFile4)
	if err := runMigrations(db, migrationFile4); err != nil {
		log.Printf("migrations warning (continuing execution): %v", err)
	} else {
		log.Println("room state references migration applied successfully.")
	}

	migrationFile5 := "db/migrations/000005_room_policy_and_schema.up.sql"
	log.Printf("applying database migrations from %s...", migrationFile5)
	if err := runMigrations(db, migrationFile5); err != nil {
		log.Printf("migrations warning (continuing execution): %v", err)
	} else {
		log.Println("room policy and schema migration applied successfully.")
	}

	// 4. Initialize Core Repositories
	authRepo := repository.NewAuthRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	memoryRepo := repository.NewMemoryRepository(db)
	eventRepo := repository.NewEventRepository(db)
	usageRepo := repository.NewUsageRepository(db)

	// 5. Initialize Services & Clients
	openRouterClient := service.NewOpenRouterServiceClient(cfg)

	// 6. Initialize Business Logic Usecases
	roomUsecase := usecase.NewRoomUsecase(roomRepo)
	messageUsecase := usecase.NewMessageUsecase(roomRepo, messageRepo, memoryRepo, openRouterClient)
	contextUsecase := usecase.NewContextUsecase(roomRepo, messageRepo, memoryRepo, openRouterClient)
	eventUsecase := usecase.NewEventUsecase(roomRepo, eventRepo)
	usageUsecase := usecase.NewUsageUsecase(usageRepo)

	// 7. Initialize Echo REST API Handlers
	eventBroker := handler.NewEventBroker()
	authMiddleware := handler.AuthMiddleware(authRepo)

	roomHandler := handler.NewRoomHandler(roomUsecase)
	messageHandler := handler.NewMessageHandler(messageUsecase)
	contextHandler := handler.NewContextHandler(contextUsecase)
	eventHandler := handler.NewEventHandler(eventUsecase, eventBroker)
	dashboardHandler := handler.NewDashboardHandler(usageUsecase, roomUsecase, messageUsecase, roomRepo)

	// 8. Initialize Echo HTTP Router
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Liveness Probe Check
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Authorized Routes V1
	v1 := e.Group("/v1")
	v1.Use(authMiddleware)
	v1.Use(handler.UsageTrackingMiddleware(usageUsecase))

	// Rooms API
	v1.POST("/rooms", roomHandler.CreateRoom)
	v1.GET("/rooms/:room_id", roomHandler.GetRoomState)
	v1.DELETE("/rooms/:room_id", roomHandler.DeleteRoom)

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

	// 9. Bind Listeners & Listen
	serverPort := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("konteks MVP backend listening on port %s...", serverPort)
	if err := e.Start(serverPort); err != nil {
		log.Fatalf("server listener failure: %v", err)
	}
}
