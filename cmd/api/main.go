package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-recruitment-backend/config"
	_ "go-recruitment-backend/docs" // Important for Swagger
	v1 "go-recruitment-backend/internal/delivery/http/v1"
	"go-recruitment-backend/internal/repository/postgres"
	"go-recruitment-backend/internal/usecase"
	"go-recruitment-backend/pkg/auth"
	"go-recruitment-backend/pkg/database"
	"go-recruitment-backend/pkg/email"
	"go-recruitment-backend/pkg/logger"
	"go-recruitment-backend/pkg/redis"
	"go-recruitment-backend/pkg/security"
	"go-recruitment-backend/pkg/validation"

	"github.com/go-playground/validator/v10"
)

// @title           Recruitment Backend API
// @version         1.0
// @description     Backend for recruitment system using Clean Architecture.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

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
	logger.Log.Info("Initializing recruitment backend...")

	// 3. Setup Database
	dbPool, err := database.NewPostgresConnection(cfg.DBUrl)
	if err != nil {
		logger.Log.Error("Failed to connect to database", "error", err)
		// Paksa berhenti jika DB mati, karena app tidak berguna tanpa DB
		os.Exit(1)
	}
	defer dbPool.Close()

	// 2b. Initialize Redis
	redisCfg := redis.Config{
		URL:      cfg.UpstashRedisURL,
		Password: cfg.UpstashRedisPassword,
	}
	if err := redis.Initialize(redisCfg); err != nil {
		logger.Log.Warn("Redis initialization failed - rate limiting will fall back to in-memory", "error", err)
	} else {
		logger.Log.Info("Redis initialized successfully")
		defer redis.Close()
	}

	// 2c. Initialize Security Logger
	// Use GIN_MODE to determine environment
	env := "development"
	if os.Getenv("GIN_MODE") == "release" {
		env = "production"
	}
	secLogger := security.InitSecurityLogger("j-expert-backend", env)

	// 2e. Setup Security Event Persistence (if enabled)
	if cfg.SecurityLogToDB {
		secEventRepo := security.NewSecurityEventRepository(dbPool)
		secLogger.SetPersistFunc(secEventRepo.CreatePersistFunc())
		logger.Log.Info("Security event database persistence enabled")
	}

	// 2d. Initialize Login Tracker
	loginTracker := security.NewLoginTracker(security.LoginTrackerConfig{
		MaxAttempts:   cfg.FailedLoginMaxAttempts,
		AttemptWindow: time.Duration(cfg.RateLimitWindowSeconds) * time.Second, // Track attempts within rate limit window
		BlockDuration: time.Duration(cfg.FailedLoginBlockMinutes) * time.Minute,
		UseIPTracking: true,
	})

	// 4. Setup Repositories
	userRepo := postgres.NewUserRepository(dbPool)
	jobRepo := postgres.NewJobRepository(dbPool)
	candidateRepo := postgres.NewCandidateRepository(dbPool)
	adminRepo := postgres.NewAdminRepository(dbPool)
	verificationRepo := postgres.NewVerificationRepository(dbPool)
	applicationRepo := postgres.NewApplicationRepository(dbPool)
	companyProfileRepo := postgres.NewCompanyProfileRepository(dbPool)
	onboardingRepo := postgres.NewOnboardingRepository(dbPool)
	atsRepo := postgres.NewATSRepository(dbPool)

	// 5. Setup Email Service
	emailService := email.NewEmailService(cfg)
	if !emailService.IsConfigured() {
		logger.Log.Warn("Email service missing configuration - contact/verification features may fail")
	}

	// 6. Setup UseCases
	validate := validator.New()
	validation.RegisterValidators(validate) // Register custom validators
	authUC := usecase.NewAuthUsecase(userRepo)
	jobUC := usecase.NewJobUsecase(jobRepo, companyProfileRepo)
	candidateUC := usecase.NewCandidateUsecase(candidateRepo, validate)
	adminUC := usecase.NewAdminUsecase(adminRepo)
	verificationUC := usecase.NewVerificationUsecase(verificationRepo, userRepo)
	applicationUC := usecase.NewApplicationUsecase(applicationRepo, jobRepo, verificationRepo)
	companyProfileUC := usecase.NewCompanyProfileUsecase(companyProfileRepo, verificationRepo)
	contactUC := usecase.NewContactUsecase(emailService)
	onboardingUC := usecase.NewOnboardingUsecase(onboardingRepo, validate)
	atsUC := usecase.NewATSUsecase(atsRepo)

	// 7. Setup Auth Provider (JWKS)
	// URL construction is now safer due to config sanitization
	jwksURL := fmt.Sprintf("%s/auth/v1/.well-known/jwks.json", cfg.SupabaseUrl)
	jwksProvider := auth.NewProvider(jwksURL)

	// 8. Setup Router
	router := v1.NewRouter(v1.RouterDeps{
		AuthUC:           authUC,
		JobUC:            jobUC,
		CandidateUC:      candidateUC,
		ApplicationUC:    applicationUC,
		AdminUC:          adminUC,
		VerificationUC:   verificationUC,
		CompanyProfileUC: companyProfileUC,
		ContactUC:        contactUC,
		OnboardingUC:     onboardingUC,
		ATSUC:            atsUC,
		LoginTracker:     loginTracker,
		JWKSProvider:     jwksProvider,
		Config:           cfg,
	})

	// 9. Start Server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
		// Good practice: Set timeouts to prevent slowloris attacks
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Log.Info("Server is running", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Error("Listen failed", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful Shutdown Logic
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Log.Info("Shutting down server...")

	// REVISI: Naikkan timeout ke 10-15 detik untuk Cloud Environment
	// 5 detik seringkali terlalu cepat untuk memutus koneksi DB yang sibuk
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("Server forced to shutdown", "error", err)
	}

	logger.Log.Info("Server exited properly")
}
