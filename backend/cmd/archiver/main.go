package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/dika/contextkit/backend/internal/config"
	"github.com/dika/contextkit/backend/internal/repository"
	"github.com/dika/contextkit/backend/internal/service"
	"github.com/dika/contextkit/backend/internal/worker"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Parse CLI flags
	inactiveDays := flag.Int("days", 14, "number of days of inactivity before archiving a room")
	roomIDFlag := flag.String("room", "", "archiving a specific room ID immediately (ignores days)")
	flag.Parse()

	log.Println("[Archiver CLI]: Initializing standalone archiver CLI app...")

	// 1. Load Configurations
	cfgPath := "secrets/config.json"
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("failed to load configuration from %s: %v", cfgPath, err)
	}

	// 2. Establish Database Connection
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	// 3. Initialize Vector Store & Storage Service
	// Cloudflare Vectorize is the only vector store; Postgres holds no embeddings.
	vectorStore := repository.NewCloudflareVectorizeStore(cfg)

	storageProvider, err := service.NewS3StorageProvider(cfg)
	if err != nil {
		log.Fatalf("failed to initialize S3 storage provider: %v", err)
	}

	// 4. Initialize Core Repositories
	roomRepo := repository.NewRoomRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	memoryRepo := repository.NewMemoryRepository(db, vectorStore)

	// 5. Run Archival Sweep
	inactiveDuration := time.Duration(*inactiveDays) * 24 * time.Hour
	ctx := context.Background()
	err = worker.RunArchivalSweep(ctx, roomRepo, messageRepo, memoryRepo, vectorStore, storageProvider, inactiveDuration, *roomIDFlag)
	if err != nil {
		log.Fatalf("archival sweep failed: %v", err)
	}

	log.Println("[Archiver CLI]: Standalone archival sweep finished successfully. Exiting.")
}
