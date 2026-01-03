package v1

import (
	"fmt"
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
		candidates.GET("/me/full", handler.GetFullProfile)    // New Endpoint
		candidates.PUT("/me/full", handler.UpdateFullProfile) // New Endpoint
		candidates.GET("/skills", handler.GetMasterSkills)    // Helper Endpoint
	}
}

// GetProfile (Legacy/Simple)
// @Summary      Get candidate profile (Simple)
// @Description  Get the basic profile
// @Tags         candidates
// @Produce      json
// @Success      200  {object}  response.Response{data=domain.CandidateProfile}
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /candidates/me [get]
// @Security     BearerAuth
func (h *CandidateHandler) GetProfile(c *gin.Context) {
	userID := c.GetString(string(domain.KeyUserID))

	profile, err := h.candidateUC.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Candidate profile", profile)
}

// GetFullProfile
// @Summary      Get full candidate profile
// @Description  Get the full profile including details, work experience, and skills
// @Tags         candidates
// @Produce      json
// @Success      200  {object}  response.Response{data=domain.CandidateWithFullDetails}
// @Router       /candidates/me/full [get]
// @Security     BearerAuth
func (h *CandidateHandler) GetFullProfile(c *gin.Context) {
	userID := c.GetString(string(domain.KeyUserID))

	fullProfile, err := h.candidateUC.GetFullProfile(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Full candidate profile", fullProfile)
}

// UpdateFullProfile
// @Summary      Update full candidate profile
// @Description  Update the full profile transactionally
// @Tags         candidates
// @Accept       json
// @Produce      json
// @Param        payload body domain.CandidateWithFullDetails true "Full Profile Payload"
// @Success      200  {object}  response.Response
// @Router       /candidates/me/full [put]
// @Security     BearerAuth
func (h *CandidateHandler) UpdateFullProfile(c *gin.Context) {
	userID := c.GetString(string(domain.KeyUserID))

	var req domain.CandidateWithFullDetails
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	// Validate certificate scores
	if err := validateCertificateScores(req.Certificates); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	err := h.candidateUC.UpdateFullProfile(c.Request.Context(), userID, &req)
	if err != nil {
		c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Profile updated successfully", nil)
}

// validateCertificateScores validates score ranges for each certificate type
func validateCertificateScores(certs []domain.CandidateCertificate) error {
	for _, c := range certs {
		if c.DocumentFilePath == "" {
			return fmt.Errorf("certificate document is required")
		}
		if c.ScoreTotal == nil {
			continue // Score is optional
		}
		score := *c.ScoreTotal
		switch c.CertificateType {
		case "TOEIC":
			if score < 0 || score > 990 {
				return fmt.Errorf("TOEIC score must be between 0 and 990")
			}
		case "IELTS":
			if score < 0 || score > 9 {
				return fmt.Errorf("IELTS score must be between 0 and 9")
			}
		case "TOEFL":
			if score < 0 || score > 120 {
				return fmt.Errorf("TOEFL iBT score must be between 0 and 120")
			}
		}
	}
	return nil
}

// GetMasterSkills
// @Summary      Get master skills list
// @Description  Get all available skills from master table
// @Tags         candidates
// @Produce      json
// @Success      200  {object}  response.Response{data=[]domain.Skill}
// @Router       /candidates/skills [get]
// @Security     BearerAuth
func (h *CandidateHandler) GetMasterSkills(c *gin.Context) {
	skills, err := h.candidateUC.GetMasterSkills(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	response.Success(c, http.StatusOK, "Master skills", skills)
}
