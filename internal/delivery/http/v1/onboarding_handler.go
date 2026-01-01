package v1

import (
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type OnboardingHandler struct {
	onboardingUC domain.OnboardingUsecase
}

func NewOnboardingHandler(r *gin.RouterGroup, onboardingUC domain.OnboardingUsecase) {
	handler := &OnboardingHandler{onboardingUC: onboardingUC}

	onboarding := r.Group("/onboarding")
	{
		onboarding.GET("/status", handler.GetStatus)
		onboarding.GET("/lpk/search", handler.SearchLPK)
		onboarding.POST("/complete", handler.Complete)
	}
}

// GetStatus godoc
// @Summary      Get onboarding status
// @Description  Check if the current user has completed the onboarding wizard
// @Tags         onboarding
// @Produce      json
// @Success      200  {object}  response.Response{data=domain.OnboardingStatus}
// @Failure      401  {object}  response.Response
// @Router       /onboarding/status [get]
// @Security     BearerAuth
func (h *OnboardingHandler) GetStatus(c *gin.Context) {
	userID := c.GetString(string(domain.KeyUserID))

	status, err := h.onboardingUC.GetOnboardingStatus(c, userID)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Onboarding status retrieved", status)
}

// SearchLPK godoc
// @Summary      Search LPK training centers
// @Description  Search for LPK (Lembaga Pelatihan Kerja) by name for autocomplete
// @Tags         onboarding
// @Produce      json
// @Param        q    query     string  false  "Search query"
// @Success      200  {object}  response.Response{data=[]domain.LPK}
// @Failure      401  {object}  response.Response
// @Router       /onboarding/lpk/search [get]
// @Security     BearerAuth
func (h *OnboardingHandler) SearchLPK(c *gin.Context) {
	query := c.Query("q")

	results, err := h.onboardingUC.SearchLPK(c, query)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "LPK search results", results)
}

// Complete godoc
// @Summary      Complete onboarding wizard
// @Description  Submit all onboarding wizard data and mark onboarding as complete
// @Tags         onboarding
// @Accept       json
// @Produce      json
// @Param        request  body      domain.OnboardingSubmitRequest  true  "Onboarding data"
// @Success      200      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Router       /onboarding/complete [post]
// @Security     BearerAuth
func (h *OnboardingHandler) Complete(c *gin.Context) {
	userID := c.GetString(string(domain.KeyUserID))

	var req domain.OnboardingSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body: "+err.Error(), nil)
		return
	}

	if err := h.onboardingUC.CompleteOnboarding(c, userID, &req); err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Onboarding completed successfully", nil)
}
