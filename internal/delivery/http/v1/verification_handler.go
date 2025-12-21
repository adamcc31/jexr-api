package v1

import (
	"bytes"
	"fmt"
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/internal/domain"
	"image"
	"image/jpeg"
	_ "image/png" // Register PNG decoder
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/image/draw"
)

type VerificationHandler struct {
	verificationUC domain.VerificationUsecase
}

func NewVerificationHandler(r *gin.RouterGroup, uc domain.VerificationUsecase) {
	handler := &VerificationHandler{
		verificationUC: uc,
	}

	// Admin routes
	verifications := r.Group("/verifications")
	{
		verifications.GET("", handler.List)
		verifications.GET("/:id", handler.GetDetail)      // Get single verification with experiences
		verifications.POST("/:id/verify", handler.Verify) // Action: approve/reject in body
	}

	// User routes
	r.GET("/verifications/me", handler.MyStatus)

	// Candidate Verification Routes
	candidates := r.Group("/candidates")
	{
		candidates.GET("/me/verification", handler.MyStatus)
		candidates.PUT("/me/verification", handler.UpdateProfile)
	}

	r.POST("/upload", handler.UploadFile)
}

// UpdateProfileRequest struct
type UpdateProfileRequest struct {
	Verification *domain.AccountVerification  `json:"verification"`
	Experiences  []domain.JapanWorkExperience `json:"experiences"`
}

// UpdateProfile godoc
// @Summary Update candidate verification profile
// @Description Update profile data and work experiences
// @Tags Verification
// @Accept json
// @Produce json
// @Success 200 {object} domain.AccountVerification
// @Router /candidates/me/verification [put]
func (h *VerificationHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString(string(domain.KeyUserID))
	// Validate role? Middleware likely handles authentication, but role check good practice

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if req.Verification == nil {
		req.Verification = &domain.AccountVerification{}
	}

	err := h.verificationUC.UpdateCandidateProfile(c.Request.Context(), userID, req.Verification, req.Experiences)
	if err != nil {
		log.Printf("ERROR UpdateProfile: userID=%s, error=%v", userID, err)
		response.Error(c, http.StatusInternalServerError, "Failed to update profile", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Profile updated successfully", nil)
}

// UploadFile godoc
// @Summary Upload a file
// @Description Upload a file (image/pdf) and get a URL. Images are compressed automatically.
// @Tags Upload
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Param bucket query string false "Target bucket"
// @Param old_url query string false "Previous file URL to delete"
// @Success 200 {object} map[string]string
// @Router /upload [post]
func (h *VerificationHandler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "No file uploaded", err.Error())
		return
	}

	// Determine bucket based on query param
	bucket := c.DefaultQuery("bucket", "CV") // Default to CV bucket

	// Validate bucket name - include all supported buckets
	validBuckets := map[string]bool{
		"Profile_Picture": true,
		"JLPT":            true,
		"CV":              true,
		"Company_Logo":    true,
		"Company_Gallery": true,
		"company_gallery": true, // Supabase bucket names as shown
		"COMPANY_GALLERY": true, // Uppercase version
		"profile_company": true, // Supabase bucket names as shown
	}
	if !validBuckets[bucket] {
		log.Printf("Invalid bucket requested: %s, falling back to CV", bucket)
		bucket = "CV" // Fallback to CV
	}

	// Get Supabase config
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_SERVICE_KEY")
	if supabaseKey == "" {
		supabaseKey = os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
		if supabaseKey == "" {
			supabaseKey = os.Getenv("SUPABASE_KEY")
		}
	}

	// Delete old file if old_url is provided
	oldURL := c.Query("old_url")
	if oldURL != "" && supabaseURL != "" && supabaseKey != "" {
		// Extract bucket and filename from the old URL
		// URL format: https://xxx.supabase.co/storage/v1/object/public/BUCKET/FILENAME
		if strings.Contains(oldURL, "/storage/v1/object/public/") {
			parts := strings.Split(oldURL, "/storage/v1/object/public/")
			if len(parts) == 2 {
				pathParts := strings.SplitN(parts[1], "/", 2)
				if len(pathParts) == 2 {
					oldBucket := pathParts[0]
					oldFilename := pathParts[1]
					deleteURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", supabaseURL, oldBucket, oldFilename)

					deleteReq, _ := http.NewRequest("DELETE", deleteURL, nil)
					deleteReq.Header.Set("Authorization", "Bearer "+supabaseKey)

					client := &http.Client{Timeout: 10 * time.Second}
					deleteResp, deleteErr := client.Do(deleteReq)
					if deleteErr == nil {
						deleteResp.Body.Close()
						log.Printf("Deleted old file: %s", oldFilename)
					}
				}
			}
		}
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to open file", err.Error())
		return
	}
	defer src.Close()

	// Read file content
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to read file", err.Error())
		return
	}

	// Detect content type from file bytes (more reliable than header)
	contentType := http.DetectContentType(fileBytes)
	log.Printf("Detected content type: %s (original filename: %s)", contentType, file.Filename)

	// Determine if it's an image for compression
	isImage := strings.HasPrefix(contentType, "image/")
	var finalBytes []byte
	var finalFilename string

	if isImage {
		// Compress image
		compressedBytes, compressErr := compressImage(fileBytes, contentType, 1200, 80)
		if compressErr != nil {
			log.Printf("Image compression failed, using original: %v", compressErr)
			finalBytes = fileBytes
		} else {
			finalBytes = compressedBytes
			log.Printf("Image compressed: %d bytes -> %d bytes", len(fileBytes), len(compressedBytes))
		}

		// Generate filename with proper extension (ASCII only for Supabase)
		finalFilename = fmt.Sprintf("%d_%s.jpg", time.Now().UnixNano(), sanitizeFilename(file.Filename))
	} else {
		// Non-image file (PDF, etc) - use as-is
		finalBytes = fileBytes
		finalFilename = fmt.Sprintf("%d_%s", time.Now().UnixNano(), sanitizeFilename(file.Filename))
	}

	log.Printf("Upload using key from env (first 20 chars: %s...)", supabaseKey[:min(20, len(supabaseKey))])

	if supabaseURL == "" || supabaseKey == "" {
		response.Error(c, http.StatusInternalServerError, "Storage not configured", "Missing Supabase credentials")
		return
	}

	// Upload to Supabase Storage
	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", supabaseURL, bucket, finalFilename)

	req, err := http.NewRequest("POST", uploadURL, bytes.NewReader(finalBytes))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create request", err.Error())
		return
	}

	// Set headers with correct content type
	// Supabase requires the correct MIME type for uploads
	req.Header.Set("Authorization", "Bearer "+supabaseKey)
	if isImage {
		req.Header.Set("Content-Type", "image/jpeg") // Compressed images are always JPEG
	} else {
		req.Header.Set("Content-Type", contentType) // Use detected content type for non-images
	}
	req.Header.Set("x-upsert", "true") // Overwrite if exists

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to upload file", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Upload failed: status=%d, body=%s", resp.StatusCode, string(respBody))
		response.Error(c, http.StatusInternalServerError, "Upload failed", string(respBody))
		return
	}

	// Construct public URL
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", supabaseURL, bucket, finalFilename)

	response.Success(c, http.StatusOK, "File uploaded", gin.H{"url": publicURL})
}

// compressImage compresses an image to the specified max dimension and quality
func compressImage(data []byte, contentType string, maxDimension int, quality int) ([]byte, error) {
	// Decode image using generic decoder (works with any registered format)
	reader := bytes.NewReader(data)
	img, format, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image (format: %s): %w", format, err)
	}

	log.Printf("Decoding image format: %s", format)

	// Get original dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate new dimensions maintaining aspect ratio
	var newWidth, newHeight int
	if width > height {
		if width > maxDimension {
			newWidth = maxDimension
			newHeight = int(float64(height) * float64(maxDimension) / float64(width))
		} else {
			newWidth = width
			newHeight = height
		}
	} else {
		if height > maxDimension {
			newHeight = maxDimension
			newWidth = int(float64(width) * float64(maxDimension) / float64(height))
		} else {
			newWidth = width
			newHeight = height
		}
	}

	// Create resized image
	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)

	// Encode as JPEG with specified quality
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}

// getExtension returns the file extension from a filename
func getExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

// sanitizeFilename removes non-ASCII characters and replaces spaces with underscores
// Supabase requires ASCII-only filenames
func sanitizeFilename(filename string) string {
	// Extract extension
	ext := getExtension(filename)
	baseName := strings.TrimSuffix(filename, "."+ext)

	// Replace spaces with underscores
	baseName = strings.ReplaceAll(baseName, " ", "_")

	// Keep only ASCII alphanumeric and underscores
	var result strings.Builder
	for _, r := range baseName {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		}
	}

	// If result is empty, use a generic name
	if result.Len() == 0 {
		return "file"
	}

	return result.String()
}

// List godoc
// @Summary List account verifications
// @Description Get paginated list of account verifications (Admin only)
// @Tags Verification
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param role query string false "Filter by role (CANDIDATE, EMPLOYER)"
// @Param status query string false "Filter by status (PENDING, VERIFIED, REJECTED)"
// @Success 200 {object} domain.PaginatedResult[domain.AccountVerification]
// @Router /verifications [get]
func (h *VerificationHandler) List(c *gin.Context) {
	// TODO: Check if user is ADMIN (Middleware should handle this, or check here)
	// Assuming RBAC middleware handles general auth, but we might want specific role check here if not global
	role, exists := c.Get(string(domain.KeyUserRole))
	if !exists || role != "admin" {
		response.Error(c, http.StatusForbidden, "Access denied: Admins only", nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	filterRole := c.Query("role")
	filterStatus := c.Query("status")

	filter := domain.VerificationFilter{
		Page:   page,
		Limit:  limit,
		Role:   filterRole,
		Status: filterStatus,
	}

	data, total, err := h.verificationUC.ListVerifications(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch verifications", err.Error())
		return
	}

	res := domain.PaginatedResult[domain.AccountVerification]{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	}

	response.Success(c, http.StatusOK, "Verifications fetched successfully", res)
}

// GetDetail godoc
// @Summary Get verification detail
// @Description Get a single verification with work experiences (Admin only)
// @Tags Verification
// @Accept json
// @Produce json
// @Param id path int true "Verification ID"
// @Success 200 {object} domain.VerificationResponse
// @Router /verifications/{id} [get]
func (h *VerificationHandler) GetDetail(c *gin.Context) {
	// Check Admin
	role, exists := c.Get(string(domain.KeyUserRole))
	if !exists || role != "admin" {
		response.Error(c, http.StatusForbidden, "Access denied: Admins only", nil)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid ID", nil)
		return
	}

	detail, err := h.verificationUC.GetVerificationByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch verification", err.Error())
		return
	}

	if detail == nil {
		response.Error(c, http.StatusNotFound, "Verification not found", nil)
		return
	}

	response.Success(c, http.StatusOK, "Verification fetched successfully", detail)
}

type VerifyRequest struct {
	Action string `json:"action" binding:"required,oneof=APPROVE REJECT approve reject"`
	Notes  string `json:"notes"`
}

// Verify godoc
// @Summary Verify an account
// @Description Approve or Reject an account verification request
// @Tags Verification
// @Accept json
// @Produce json
// @Param id path int true "Verification ID"
// @Param request body VerifyRequest true "Action and Notes"
// @Success 200 {object} domain.AccountVerification
// @Router /verifications/{id}/verify [post]
func (h *VerificationHandler) Verify(c *gin.Context) {
	// Check Admin
	role, exists := c.Get(string(domain.KeyUserRole))
	if !exists || role != "admin" {
		response.Error(c, http.StatusForbidden, "Access denied: Admins only", nil)
		return
	}
	adminID, _ := c.Get(string(domain.KeyUserID))

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid ID", nil)
		return
	}

	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	err = h.verificationUC.VerifyUser(c.Request.Context(), adminID.(string), id, req.Action, req.Notes)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to verify user", err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Verification updated", nil)
}

// MyStatus godoc
// @Summary Get my verification status
// @Description Get the verification status of the current user
// @Tags Verification
// @Accept json
// @Produce json
// @Success 200 {object} domain.AccountVerification
// @Router /verifications/me [get]
func (h *VerificationHandler) MyStatus(c *gin.Context) {
	userID, exists := c.Get(string(domain.KeyUserID))
	if !exists {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	status, err := h.verificationUC.GetVerificationStatus(c.Request.Context(), userID.(string))
	if err != nil {
		// It's possible they don't have a record yet
		response.Error(c, http.StatusNotFound, "No verification record found", nil)
		return
	}

	response.Success(c, http.StatusOK, "Status fetched", status)
}
