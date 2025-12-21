package v1

import (
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ContactHandler struct {
	contactUC domain.ContactUsecase
}

// NewContactHandler registers the contact routes (public, no auth required)
func NewContactHandler(public *gin.RouterGroup, contactUC domain.ContactUsecase) {
	handler := &ContactHandler{
		contactUC: contactUC,
	}

	// Public Routes - NO authentication required
	public.POST("/contact", handler.SubmitContact)
}

// SubmitContact godoc
// @Summary      Submit Contact Form
// @Description  Send a message through the contact form. This is a public endpoint.
// @Tags         contact
// @Accept       json
// @Produce      json
// @Param        contact  body      domain.ContactRequest  true  "Contact Form Data"
// @Success      200      {object}  response.Response
// @Failure      400      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Router       /contact [post]
func (h *ContactHandler) SubmitContact(c *gin.Context) {
	var req domain.ContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	if err := h.contactUC.SendContactMessage(c.Request.Context(), &req); err != nil {
		// Check if it's a configuration error vs a send error
		if err.Error() == "email service is not configured" {
			c.Error(apperror.New(http.StatusServiceUnavailable, "Contact service temporarily unavailable", err))
			return
		}
		c.Error(apperror.New(http.StatusInternalServerError, "Failed to send message. Please try again later.", err))
		return
	}

	response.Success(c, http.StatusOK, "Your message has been sent successfully!", nil)
}
