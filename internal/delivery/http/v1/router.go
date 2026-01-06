package v1

import (
	"go-recruitment-backend/config"
	"go-recruitment-backend/internal/delivery/http/middleware"
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/auth"
	"go-recruitment-backend/pkg/security"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type RouterDeps struct {
	AuthUC           domain.AuthUsecase
	JobUC            domain.JobUsecase
	CandidateUC      domain.CandidateUsecase
	ApplicationUC    domain.ApplicationUsecase    // Added for application endpoints
	AdminUC          domain.AdminUsecase          // Added for admin endpoints
	VerificationUC   domain.VerificationUsecase   // Added for verification endpoints
	CompanyProfileUC domain.CompanyProfileUsecase // Added for company profile endpoints
	ContactUC        domain.ContactUsecase        // Added for contact form
	OnboardingUC     domain.OnboardingUsecase     // Added for onboarding wizard
	ATSUC            domain.ATSUsecase            // Added for ATS (Applicant Tracking System)
	LoginTracker     *security.LoginTracker       // Security: Login blocking
	JWKSProvider     *auth.Provider
	Config           *config.Config
}

func NewRouter(deps RouterDeps) *gin.Engine {
	r := gin.New()

	// Global Middlewares
	r.Use(middleware.CORSMiddleware())            // CORS must be first!
	r.Use(middleware.SecurityHeadersMiddleware()) // Security headers (HSTS, XSS, etc.)
	r.Use(middleware.GlobalRateLimitMiddleware()) // Global rate limit: 100 req/min per IP
	r.Use(middleware.CSRFMiddleware())            // CSRF protection (Double-Submit Cookie)
	r.Use(gin.Recovery())
	r.Use(gin.Logger()) // Use standard Gin logger
	r.Use(middleware.RequestID())
	r.Use(middleware.ErrorHandler())

	v1 := r.Group("/v1")

	// Health Check
	v1.GET("/health", func(c *gin.Context) {
		response.Success(c, http.StatusOK, "System operational", nil)
	})

	// Public routes
	NewContactHandler(v1, deps.ContactUC) // Contact form (no auth required)

	// Swagger - ONLY available in development mode
	// In production, this is disabled to prevent API enumeration
	if os.Getenv("GIN_MODE") != "release" {
		v1.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(deps.JWKSProvider, deps.Config, deps.AuthUC))
	{
		NewAuthHandler(v1, protected, deps.AuthUC, deps.OnboardingUC, deps.Config, deps.LoginTracker)
		NewJobHandler(v1, protected, deps.JobUC)
		NewCandidateHandler(protected, deps.CandidateUC)
		NewApplicationHandler(protected, deps.ApplicationUC)                                // Application routes
		NewAdminHandler(protected, deps.AdminUC)                                            // Admin routes
		NewVerificationHandler(protected, deps.VerificationUC)                              // Verification routes
		NewCompanyProfileHandler(v1, protected, deps.CompanyProfileUC, deps.VerificationUC) // Company profile routes
		NewOnboardingHandler(protected, deps.OnboardingUC)                                  // Onboarding wizard routes
		NewATSHandler(protected, deps.ATSUC)                                                // ATS (Applicant Tracking System) routes
	}

	return r
}
