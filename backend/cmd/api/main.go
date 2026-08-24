package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"

	"linkmeqr/backend/internal/config"
	"linkmeqr/backend/internal/database"
	"linkmeqr/backend/internal/handlers"
	appmiddleware "linkmeqr/backend/internal/middleware"
	"linkmeqr/backend/internal/repository"
	"linkmeqr/backend/internal/services"
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

	if err := database.Migrate(db, "migrations"); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	// --- repositories ---
	userRepo := repository.NewUserRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	licenseRepo := repository.NewLicenseRepository(db)
	codeRepo := repository.NewActivationCodeRepository(db)
	activationRepo := repository.NewLicenseActivationRepository(db)
	auditRepo := repository.NewAuditLogRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	themeRepo := repository.NewProfileThemeRepository(db)
	blockRepo := repository.NewProfileBlockRepository(db)
	mediaRepo := repository.NewMediaRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	qrRepo := repository.NewQRRepository(db)
	templateRepo := repository.NewTemplateRepository(db)

	// --- services ---
	authSvc := services.NewAuthService(userRepo, refreshRepo, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	auditSvc := services.NewAuditService(auditRepo)
	licenseSvc := services.NewLicenseService(db, licenseRepo, codeRepo, activationRepo, auditSvc)
	clientSvc := services.NewClientService(userRepo)
	profileSvc := services.NewProfileService(profileRepo, themeRepo)
	blockSvc := services.NewBlockService(blockRepo)
	mediaSvc := services.NewMediaService(mediaRepo, cfg.MediaStoragePath)
	analyticsSvc := services.NewAnalyticsService(analyticsRepo)
	qrSvc := services.NewQRManagementService(qrRepo, cfg.PublicBaseURL)

	// --- handlers ---
	authHandler := handlers.NewAuthHandler(authSvc)
	meHandler := handlers.NewMeHandler(userRepo, licenseRepo)
	licenseHandler := handlers.NewLicenseHandler(licenseSvc, licenseRepo, auditSvc)
	clientHandler := handlers.NewClientHandler(clientSvc, licenseRepo, auditSvc)
	auditHandler := handlers.NewAuditHandler(auditRepo)
	profileHandler := handlers.NewProfileHandler(profileSvc, mediaRepo)
	blockHandler := handlers.NewBlockHandler(blockSvc, profileSvc, mediaRepo)
	mediaHandler := handlers.NewMediaHandler(mediaSvc)
	publicHandler := handlers.NewPublicHandler(profileSvc, blockSvc, licenseRepo, analyticsSvc, mediaRepo)
	statsHandler := handlers.NewStatsHandler(profileSvc, analyticsSvc)
	qrHandler := handlers.NewQRHandler(qrSvc, profileSvc)
	adminProfileHandler := handlers.NewAdminProfileHandler(profileSvc, auditSvc)
	templateHandler := handlers.NewTemplateHandler(templateRepo)

	loginLimiter := appmiddleware.NewIPRateLimiter(rate.Every(2*time.Second), 5)
	generalLimiter := appmiddleware.NewIPRateLimiter(rate.Every(100*time.Millisecond), 30)

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.FrontendOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(generalLimiter.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Serve uploaded media (logos, backgrounds, block images) directly.
	r.Handle("/media/*", http.StripPrefix("/media/", http.FileServer(http.Dir(cfg.MediaStoragePath))))

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.With(loginLimiter.Middleware).Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		r.Route("/public", func(r chi.Router) {
			r.Get("/profiles/{slug}", publicHandler.GetBySlug)
			r.Post("/profiles/{slug}/events", publicHandler.TrackEvent)
		})

		r.Get("/templates", templateHandler.List)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(cfg.JWTSecret))

			r.Get("/me", meHandler.Get)

			r.Group(func(r chi.Router) {
				r.Use(appmiddleware.RequireRole("CLIENT"))

				r.Post("/me/license/activate", licenseHandler.Activate)
				r.Get("/me/license/history", licenseHandler.History)

				r.Get("/me/profile", profileHandler.GetMine)
				r.Patch("/me/profile", profileHandler.UpdateMine)
				r.Get("/me/theme", profileHandler.GetMyTheme)
				r.Patch("/me/theme", profileHandler.UpdateMyTheme)

				r.Get("/me/blocks", blockHandler.List)
				r.Post("/me/blocks", blockHandler.Create)
				r.Patch("/me/blocks/{id}", blockHandler.Update)
				r.Delete("/me/blocks/{id}", blockHandler.Delete)
				r.Post("/me/blocks/{id}/duplicate", blockHandler.Duplicate)
				r.Patch("/me/blocks/reorder", blockHandler.Reorder)

				r.Get("/me/qr", qrHandler.Get)
				r.Patch("/me/qr", qrHandler.Update)
				r.Get("/me/qr/validate", qrHandler.Validate)
				r.Get("/me/qr/export", qrHandler.Export)

				r.Get("/me/stats/summary", statsHandler.MySummary)

				r.Post("/media/upload", mediaHandler.Upload)
			})

			r.Route("/admin", func(r chi.Router) {
				r.Use(appmiddleware.RequireRole("ADMIN"))

				r.Route("/clients", func(r chi.Router) {
					r.Get("/", clientHandler.List)
					r.Post("/", clientHandler.Create)
					r.Get("/{id}", clientHandler.Get)
					r.Patch("/{id}", clientHandler.Update)
					r.Post("/{id}/activate", clientHandler.SetActive(true))
					r.Post("/{id}/deactivate", clientHandler.SetActive(false))
					r.Get("/{id}/profile", adminProfileHandler.GetForClient)
					r.Post("/{id}/profile", adminProfileHandler.CreateForClient)
					r.Post("/{id}/license/activate", licenseHandler.AdminActivate)
				})

				r.Get("/profiles", adminProfileHandler.List)

				r.Route("/licenses", func(r chi.Router) {
					r.Post("/codes", licenseHandler.GenerateCode)
					r.Post("/codes/batch", licenseHandler.GenerateBatch)
					r.Get("/codes", licenseHandler.ListCodes)
					r.Post("/codes/{id}/revoke", licenseHandler.RevokeCode)
					r.Get("/{userId}/history", licenseHandler.AdminHistory)
				})

				r.Get("/audit-logs", auditHandler.List)
			})
		})
	})

	addr := ":" + cfg.HTTPPort
	log.Printf("LinkMeQR API listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
