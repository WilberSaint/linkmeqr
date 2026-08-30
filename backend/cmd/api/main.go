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
	loyaltyRepo := repository.NewLoyaltyRepository(db)
	printCardRepo := repository.NewPrintCardRepository(db)

	// --- services ---
	authSvc := services.NewAuthService(userRepo, refreshRepo, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	auditSvc := services.NewAuditService(auditRepo)
	licenseSvc := services.NewLicenseService(db, licenseRepo, codeRepo, activationRepo, auditSvc)
	clientSvc := services.NewClientService(userRepo)
	profileSvc := services.NewProfileService(profileRepo, themeRepo)
	blockSvc := services.NewBlockService(blockRepo)
	mediaSvc := services.NewMediaService(mediaRepo, cfg.MediaStoragePath)
	analyticsSvc := services.NewAnalyticsService(analyticsRepo)
	qrSvc := services.NewQRManagementService(qrRepo, mediaRepo, mediaSvc, cfg.PublicBaseURL)
	templateSvc := services.NewTemplateService(templateRepo)
	loyaltySvc := services.NewLoyaltyService(loyaltyRepo)
	printCardSvc := services.NewPrintCardService(printCardRepo, profileSvc, blockSvc, qrSvc, loyaltySvc, mediaRepo, analyticsRepo)
	googleWalletSvc, err := services.NewGoogleWalletService(services.GoogleWalletConfig{
		IssuerID:            cfg.GoogleWalletIssuerID,
		ServiceAccountEmail: cfg.GoogleWalletServiceAccountEmail,
		PrivateKeyPEM:       cfg.GoogleWalletPrivateKey,
		ReviewStatus:        cfg.GoogleWalletReviewStatus,
	}, cfg.PublicBaseURL)
	if err != nil {
		log.Fatalf("google wallet config error: %v", err)
	}

	// --- handlers ---
	authHandler := handlers.NewAuthHandler(authSvc)
	meHandler := handlers.NewMeHandler(userRepo, licenseRepo)
	licenseHandler := handlers.NewLicenseHandler(licenseSvc, licenseRepo, userRepo, auditSvc)
	clientHandler := handlers.NewClientHandler(clientSvc, licenseRepo, auditSvc, cfg.JWTSecret, cfg.JWTAccessTTL)
	auditHandler := handlers.NewAuditHandler(auditRepo, userRepo)
	profileHandler := handlers.NewProfileHandler(profileSvc, mediaRepo)
	blockHandler := handlers.NewBlockHandler(blockSvc, profileSvc, mediaRepo)
	mediaHandler := handlers.NewMediaHandler(mediaSvc)
	publicHandler := handlers.NewPublicHandler(profileSvc, blockSvc, licenseRepo, analyticsSvc, mediaRepo)
	shellHandler := handlers.NewShellHandler(profileSvc, licenseRepo, mediaRepo, cfg.FrontendShellURL, cfg.PublicBaseURL)
	statsHandler := handlers.NewStatsHandler(profileSvc, analyticsSvc)
	qrHandler := handlers.NewQRHandler(qrSvc, profileSvc, mediaRepo)
	adminProfileHandler := handlers.NewAdminProfileHandler(profileSvc, auditSvc)
	templateHandler := handlers.NewTemplateHandler(templateSvc, auditSvc)
	loyaltyHandler := handlers.NewLoyaltyHandler(loyaltySvc, profileSvc, qrSvc, mediaRepo, auditSvc, googleWalletSvc)
	printCardHandler := handlers.NewPrintCardHandler(printCardSvc, profileSvc, qrSvc, mediaRepo, mediaSvc, auditSvc, analyticsSvc)

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
	//
	// Every stored file is named by a UUID and never rewritten — replacing an
	// image uploads a new one under a new name — so the bytes at a given path
	// are immutable and can be cached hard. Without this every visit re-fetched
	// the logo, cover and background in full, which on mobile data is most of
	// what a QR scan waits for.
	mediaFiles := http.StripPrefix("/media/", http.FileServer(http.Dir(cfg.MediaStoragePath)))
	r.Handle("/media/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		mediaFiles.ServeHTTP(w, r)
	}))

	// Short, trackable links every exported print card's QR encodes — see
	// PrintCardHandler.Scan. Deliberately outside /api and unauthenticated:
	// this is hit directly by a phone camera scanning a physical card.
	r.Get("/q/{code}", printCardHandler.Scan)
	r.Get("/q/{code}/{slot}", printCardHandler.Scan)

	// The public profile page. Served here rather than straight off the
	// static frontend so each business's own Open Graph tags land in the
	// HTML — see ShellHandler. Visitors still get the identical SPA shell.
	r.Get("/p/{slug}", shellHandler.ProfileShell)

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.With(loginLimiter.Middleware).Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		r.Route("/public", func(r chi.Router) {
			r.Get("/profiles/{slug}", publicHandler.GetBySlug)
			r.Post("/profiles/{slug}/events", publicHandler.TrackEvent)

			r.Get("/loyalty/{token}", loyaltyHandler.PublicStatus)
			r.Post("/loyalty/{token}/register", loyaltyHandler.PublicRegister)
			r.Get("/loyalty/{token}/wallet", loyaltyHandler.WalletSaveURL)
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

				r.Get("/me/loyalty", loyaltyHandler.GetMine)
				r.Patch("/me/loyalty", loyaltyHandler.UpdateMine)
				r.Get("/me/loyalty/qr", loyaltyHandler.ExportQR)
				r.Get("/me/loyalty/customers", loyaltyHandler.ListCustomers)
				r.Post("/me/loyalty/customers/{id}/stamp", loyaltyHandler.StampCustomer)
				r.Post("/me/loyalty/customers/{id}/redeem", loyaltyHandler.RedeemCustomer)

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
					r.Patch("/{id}/profile/logo", adminProfileHandler.UpdateLogoForClient)
					r.Post("/{id}/license/activate", licenseHandler.AdminActivate)
					r.Post("/{id}/impersonate", clientHandler.Impersonate)

					r.Get("/{id}/qr", qrHandler.GetForClient)
					r.Patch("/{id}/qr", qrHandler.UpdateForClient)
					r.Get("/{id}/qr/validate", qrHandler.ValidateForClient)
					r.Get("/{id}/qr/export", qrHandler.ExportForClient)
					r.Post("/{id}/media/upload", mediaHandler.UploadForClient)

					r.Route("/{id}/print-cards", func(r chi.Router) {
						r.Get("/", printCardHandler.List)
						r.Post("/", printCardHandler.Create)
						r.Post("/preview", printCardHandler.Preview)
						r.Post("/seed-layout", printCardHandler.SeedLayout)
						r.Post("/qr-preview", printCardHandler.QRPreview)
						r.Get("/qr-targets", printCardHandler.QRTargets)
						r.Get("/{cardId}", printCardHandler.Get)
						r.Patch("/{cardId}", printCardHandler.Update)
						r.Patch("/{cardId}/status", printCardHandler.UpdateStatus)
						r.Delete("/{cardId}", printCardHandler.Delete)
						r.Get("/{cardId}/export", printCardHandler.Export)
						r.Get("/{cardId}/layout", printCardHandler.GetLayout)
						r.Put("/{cardId}/layout", printCardHandler.SaveLayout)
						r.Get("/{cardId}/layout/versions", printCardHandler.ListLayoutVersions)
						r.Post("/{cardId}/layout/versions/{version}/restore", printCardHandler.RestoreLayoutVersion)
					})
				})

				r.Get("/print-cards/icons/{name}", printCardHandler.IconPreview)

				r.Get("/profiles", adminProfileHandler.List)

				r.Route("/templates", func(r chi.Router) {
					r.Get("/", templateHandler.ListAllAdmin)
					r.Post("/", templateHandler.Create)
					r.Get("/{id}", templateHandler.Get)
					r.Patch("/{id}", templateHandler.Update)
					r.Delete("/{id}", templateHandler.Delete)
					r.Post("/{id}/activate", templateHandler.SetActive(true))
					r.Post("/{id}/deactivate", templateHandler.SetActive(false))
				})

				r.Route("/licenses", func(r chi.Router) {
					r.Post("/codes", licenseHandler.GenerateCode)
					r.Post("/codes/batch", licenseHandler.GenerateBatch)
					r.Get("/codes", licenseHandler.ListCodes)
					r.Post("/codes/{id}/revoke", licenseHandler.RevokeCode)
					r.Post("/codes/{id}/assign", licenseHandler.AssignCode)
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
