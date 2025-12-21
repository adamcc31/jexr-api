package main

import (
	"context"
	"go-recruitment-backend/config"
	_ "go-recruitment-backend/docs" // Important for Swagger
	v1 "go-recruitment-backend/internal/delivery/http/v1"
	"go-recruitment-backend/internal/repository/postgres"
	"go-recruitment-backend/internal/usecase"
	"go-recruitment-backend/pkg/auth"
	"go-recruitment-backend/pkg/database"
	"go-recruitment-backend/pkg/email"
	"go-recruitment-backend/pkg/logger"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
)

// @title           Recruitment Backend API
// @version         1.0
// @description     Backend for recruitment system using Clean Architecture.
// @host            localhost:8080
// @BasePath        /v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// 1. Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Setup Logger
	logger.Init()
	logger.Log.Info("Starting recruitment backend", "port", cfg.Port)

	// 3. Setup Database
	dbPool, err := database.NewPostgresConnection(cfg.DBUrl)
	if err != nil {
		logger.Log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// 4. Setup Repositories
	userRepo := postgres.NewUserRepository(dbPool)
	jobRepo := postgres.NewJobRepository(dbPool)
	candidateRepo := postgres.NewCandidateRepository(dbPool)
	adminRepo := postgres.NewAdminRepository(dbPool)
	verificationRepo := postgres.NewVerificationRepository(dbPool)     // Verification Repo
	applicationRepo := postgres.NewApplicationRepository(dbPool)       // Application Repo
	companyProfileRepo := postgres.NewCompanyProfileRepository(dbPool) // Company Profile Repo

	// 5. Setup Email Service
	emailService := email.NewEmailService(cfg)
	if !emailService.IsConfigured() {
		logger.Log.Warn("Email service not fully configured - contact form will be unavailable")
	}

	// 6. Setup UseCases
	validate := validator.New()
	authUC := usecase.NewAuthUsecase(userRepo)
	jobUC := usecase.NewJobUsecase(jobRepo, companyProfileRepo)
	candidateUC := usecase.NewCandidateUsecase(candidateRepo, validate)
	adminUC := usecase.NewAdminUsecase(adminRepo)
	verificationUC := usecase.NewVerificationUsecase(verificationRepo, userRepo)               // Verification Usecase
	applicationUC := usecase.NewApplicationUsecase(applicationRepo, jobRepo, verificationRepo) // Application Usecase
	companyProfileUC := usecase.NewCompanyProfileUsecase(companyProfileRepo, verificationRepo) // Company Profile Usecase
	contactUC := usecase.NewContactUsecase(emailService)                                       // Contact Usecase

	// 7. Setup Auth Provider (JWKS)
	// Assuming Supabase URL is like https://xyz.supabase.co
	jwksURL := cfg.SupabaseUrl + "/auth/v1/.well-known/jwks.json"
	jwksProvider := auth.NewProvider(jwksURL)

	// 8. Setup Router
	router := v1.NewRouter(v1.RouterDeps{
		AuthUC:           authUC,
		JobUC:            jobUC,
		CandidateUC:      candidateUC,
		ApplicationUC:    applicationUC, // Inject ApplicationUC
		AdminUC:          adminUC,
		VerificationUC:   verificationUC,   // Inject VerificationUC
		CompanyProfileUC: companyProfileUC, // Inject CompanyProfileUC
		ContactUC:        contactUC,        // Inject ContactUC
		JWKSProvider:     jwksProvider,
		Config:           cfg,
	})

	// 9. Start Server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Error("Listen failed", "error", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("Server forced to shutdown", "error", err)
	}

	logger.Log.Info("Server exiting")
}
