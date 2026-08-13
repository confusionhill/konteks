package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/dika/contextkit/backend/internal/config"
	"github.com/dika/contextkit/backend/internal/repository"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

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

	// 1. Create or fetch User
	user, err := authRepo.GetUserByEmail(ctx, "admin@admin.com")
	if err != nil {
		log.Fatalf("failed to check user: %v", err)
	}
	if user == nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		user, err = authRepo.CreateUser(ctx, "admin@admin.com", string(hash), "Admin User")
		if err != nil {
			log.Fatalf("failed to create user: %v", err)
		}
	}

	// 2. Create Project
	projects, err := authRepo.ListProjectsByOwnerID(ctx, user.ID)
	var projID string
	if len(projects) == 0 {
		proj, err := authRepo.CreateProject(ctx, "Demo AI Assistant Project", "Default project for testing", user.ID)
		if err != nil {
			log.Fatalf("failed to create project: %v", err)
		}
		projID = proj.ID
	} else {
		projID = projects[0].ID
	}

	// 3. Generate API Key
	rawKey := "ck_live_dev_test_key_123"
	hashBytes := sha256.Sum256([]byte(rawKey))
	keyHash := fmt.Sprintf("%x", hashBytes)

	_, err = authRepo.CreateAPIKey(ctx, projID, "Development Test Key", keyHash, "ck_live_dev...", nil)
	if err != nil {
		log.Printf("API key seed info: %v", err)
	}

	fmt.Println("==================================================")
	fmt.Println(" konteks Seed Completed Successfully!")
	fmt.Println("==================================================")
	fmt.Printf("User Email : %s\n", user.Email)
	fmt.Printf("Password   : admin123\n")
	fmt.Printf("Project ID : %s\n", projID)
	fmt.Printf("API Key    : %s\n", rawKey)
	fmt.Println("==================================================")
}
