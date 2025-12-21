package v1

import (
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CandidateHandler struct {
	candidateUC domain.CandidateUsecase
}

func NewCandidateHandler(r *gin.RouterGroup, candidateUC domain.CandidateUsecase) {
	handler := &CandidateHandler{candidateUC: candidateUC}

	candidates := r.Group("/candidates")
	{
		candidates.GET("/me", handler.GetProfile)
	}
}

// GetProfile godoc
// @Summary      Get candidate profile
// @Description  Get the profile of the currently logged-in candidate
// @Tags         candidates
// @Produce      json
// @Success      200  {object}  response.Response{data=domain.CandidateProfile}
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /candidates/me [get]
// @Security     BearerAuth
func (h *CandidateHandler) GetProfile(c *gin.Context) {
	userID := c.GetString(string(domain.KeyUserID))
	// Role check could be done here or in middleware
	role := c.GetString(string(domain.KeyUserRole))
	if role != "candidate" {
		// Strict check if middleware is generic
		// c.Error(apperror.Forbidden("Only candidates can access this"))
		// return
		// For now assume middleware handles it or we allow it but return 404 if no profile
	}

	// Fix: Pass 'c' directly because Gin Context implements context.Context and contains the Keys
	profile, err := h.candidateUC.GetProfile(c, userID)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Candidate profile", profile)
}
