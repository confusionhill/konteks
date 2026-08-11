package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/dika/contextkit/backend/internal/config"
	"github.com/dika/contextkit/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func generateRandomToken() (string, error) {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "kt_" + hex.EncodeToString(bytes), nil
}

func main() {
	cfg, err := config.LoadConfig("secrets/config.json")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	authRepo := repository.NewAuthRepository(db)
	ctx := context.Background()

	// 1. Create Tenant
	tenantName := "Default Tenant"
	tenant, err := authRepo.CreateTenant(ctx, tenantName)
	if err != nil {
		log.Fatalf("failed to create tenant: %v", err)
	}

	// 2. Generate and Hash API Key
	token, err := generateRandomToken()
	if err != nil {
		log.Fatalf("failed to generate secure key: %v", err)
	}

	hashBytes := sha256.Sum256([]byte(token))
	keyHash := fmt.Sprintf("%x", hashBytes)

	// 3. Persist Key Hash
	_, err = authRepo.CreateAPIKey(ctx, tenant.ID, "Primary API Key", keyHash)
	if err != nil {
		log.Fatalf("failed to store API key: %v", err)
	}

	fmt.Println("==================================================")
	fmt.Println(" konteks Seed Completed Successfully!")
	fmt.Println("==================================================")
	fmt.Printf("Tenant Name : %s\n", tenant.Name)
	fmt.Printf("Tenant ID   : %s\n", tenant.ID)
	fmt.Printf("API Key     : %s\n", token)
	fmt.Println("==================================================")
	fmt.Println("IMPORTANT: Store this API key securely. It will not be shown again.")
	fmt.Println("Use this key in your request header: Authorization: Bearer <API Key>")
	fmt.Println("==================================================")
}
