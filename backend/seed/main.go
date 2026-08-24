// Command seed creates the initial ADMIN user and default templates.
// Safe to run multiple times: existing rows are left untouched.
package main

import (
	"context"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"linkmeqr/backend/internal/config"
	"linkmeqr/backend/internal/database"
	"linkmeqr/backend/internal/models"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/utils"
)

func main() {
	_ = godotenv.Load()          // backend/.env, if running from backend/
	_ = godotenv.Load("../.env") // repo-root .env, if running from backend/

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	db, err := database.Connect(cfg.MySQLDSN())
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	users := repository.NewUserRepository(db)

	adminEmail := getEnv("SEED_ADMIN_EMAIL", "admin@linkmeqr.com")
	adminPassword := getEnv("SEED_ADMIN_PASSWORD", "")
	if adminPassword == "" {
		log.Fatal("SEED_ADMIN_PASSWORD must be set to seed the initial admin user")
	}

	if _, err := users.GetByEmail(ctx, adminEmail); err == nil {
		log.Printf("admin user %s already exists, skipping", adminEmail)
	} else if err == repository.ErrNotFound {
		hash, err := utils.HashPassword(adminPassword)
		if err != nil {
			log.Fatalf("hash password: %v", err)
		}

		admin := &models.User{
			ID:           uuid.NewString(),
			Email:        adminEmail,
			PasswordHash: hash,
			Role:         models.RoleAdmin,
			FullName:     "LinkMeQR Admin",
			IsActive:     true,
		}
		if err := users.Create(ctx, admin); err != nil {
			log.Fatalf("create admin: %v", err)
		}
		log.Printf("created admin user: %s", adminEmail)
	} else {
		log.Fatalf("check admin user: %v", err)
	}

	if err := seedTemplates(ctx, db); err != nil {
		log.Fatalf("seed templates: %v", err)
	}

	log.Println("seed complete")
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
