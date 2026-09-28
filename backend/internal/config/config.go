package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	AppEnv   string
	HTTPPort string
	HTTPHost string
	// DBPath is the SQLite database file; created on first start.
	DBPath          string
	JWTSecret       string
	JWTRefreshTTL   time.Duration
	JWTAccessTTL    time.Duration
	FrontendOrigins []string
	PublicBaseURL   string
	// FrontendShellURL is where the SPA's index.html is fetched from when
	// FrontendDistPath is unset (local dev: the Vite server), so /p/:slug
	// can be returned with the business's own Open Graph tags injected into
	// it (see ShellHandler.ProfileShell).
	FrontendShellURL string
	MediaStoragePath string
	// FrontendDistPath, when set, is the built SPA (frontend/dist) this same
	// process serves for every non-API route, and where the /p/:slug shell is
	// read from instead of FrontendShellURL. Production runs this way — one
	// process, no separate web server for the frontend.
	FrontendDistPath string

	GoogleWalletIssuerID            string
	GoogleWalletServiceAccountEmail string
	GoogleWalletPrivateKey          string
	GoogleWalletReviewStatus        string
}

func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		HTTPHost:         getEnv("HTTP_HOST", ""),
		DBPath:           getEnv("DB_PATH", "./data/linkmeqr.db"),
		JWTSecret:        getEnv("JWT_SECRET", ""),
		JWTAccessTTL:     15 * time.Minute,
		JWTRefreshTTL:    30 * 24 * time.Hour,
		FrontendOrigins:  splitCSV(getEnv("FRONTEND_ORIGIN", "http://localhost:5173")),
		PublicBaseURL:    getEnv("PUBLIC_BASE_URL", "http://localhost:5173"),
		FrontendShellURL: getEnv("FRONTEND_SHELL_URL", "http://localhost:5173/"),
		MediaStoragePath: getEnv("MEDIA_STORAGE_PATH", "./media"),
		FrontendDistPath: getEnv("FRONTEND_DIST_PATH", ""),

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
