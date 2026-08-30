package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	AppEnv           string
	HTTPPort         string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	JWTSecret        string
	JWTRefreshTTL    time.Duration
	JWTAccessTTL     time.Duration
	FrontendOrigins  []string
	PublicBaseURL    string
	// FrontendShellURL is where the built SPA's index.html is served from,
	// so /p/:slug can be returned with the business's own Open Graph tags
	// injected into it (see PublicHandler.ProfileShell). In production
	// that's the frontend container on the internal network.
	FrontendShellURL string
	MediaStoragePath string

	GoogleWalletIssuerID            string
	GoogleWalletServiceAccountEmail string
	GoogleWalletPrivateKey          string
	GoogleWalletReviewStatus        string
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		DBHost:           getEnv("DB_HOST", "127.0.0.1"),
		DBPort:           getEnv("DB_PORT", "3306"),
		DBUser:           getEnv("DB_USER", "root"),
		DBPassword:       getEnv("DB_PASSWORD", ""),
		DBName:           getEnv("DB_NAME", "linkmeqr"),
		JWTSecret:        getEnv("JWT_SECRET", ""),
		JWTAccessTTL:     15 * time.Minute,
		JWTRefreshTTL:    30 * 24 * time.Hour,
		FrontendOrigins:  splitCSV(getEnv("FRONTEND_ORIGIN", "http://localhost:5173")),
		PublicBaseURL:    getEnv("PUBLIC_BASE_URL", "http://localhost:5173"),
		FrontendShellURL: getEnv("FRONTEND_SHELL_URL", "http://frontend/"),
		MediaStoragePath: getEnv("MEDIA_STORAGE_PATH", "./media"),

		// Google Wallet: all optional — the loyalty "Add to Google Wallet"
		// button simply stays hidden until an Issuer account + service
		// account key are configured. See .env.example for setup notes.
		GoogleWalletIssuerID:            getEnv("GOOGLE_WALLET_ISSUER_ID", ""),
		GoogleWalletServiceAccountEmail: getEnv("GOOGLE_WALLET_SERVICE_ACCOUNT_EMAIL", ""),
		GoogleWalletPrivateKey:          getEnv("GOOGLE_WALLET_PRIVATE_KEY", ""),
		GoogleWalletReviewStatus:        getEnv("GOOGLE_WALLET_REVIEW_STATUS", "UNDER_REVIEW"),
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func (c *Config) MySQLDSN() string {
	// multiStatements is required because the migrator (golang-migrate) runs
	// each .sql file as a single Exec call over this same connection.
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=UTC&multiStatements=true",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// splitCSV lets FRONTEND_ORIGIN carry multiple allowed origins (comma-separated),
// useful in local dev to allow both http://localhost:5173 and a LAN IP for phone testing.
func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
