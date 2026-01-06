package v1

import (
	"go-recruitment-backend/internal/delivery/http/response"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/apperror"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ATSHandler struct {
	atsUC domain.ATSUsecase
}

// NewATSHandler registers ATS routes
func NewATSHandler(protected *gin.RouterGroup, atsUC domain.ATSUsecase) {
	handler := &ATSHandler{atsUC: atsUC}

	ats := protected.Group("/admin/ats")
	{
		ats.GET("/candidates", handler.SearchCandidates)
		ats.GET("/export", handler.ExportCandidates)
		ats.GET("/filter-options", handler.GetFilterOptions)
	}
}

// SearchCandidates godoc
// @Summary      Search candidates with filters
// @Description  Returns paginated list of candidates matching the filter criteria
// @Tags         admin-ats
// @Produce      json
// @Security     BearerAuth
// @Param        japanese_levels       query     string   false  "Comma-separated JLPT levels (N1,N2,N3,N4,N5,NON_CERTIFIED)"
// @Param        japan_experience_min  query     int      false  "Minimum Japan experience in months"
// @Param        japan_experience_max  query     int      false  "Maximum Japan experience in months"
// @Param        has_lpk_training      query     bool     false  "Filter by LPK training status"
// @Param        english_cert_types    query     string   false  "Comma-separated cert types (TOEFL,IELTS,TOEIC)"
// @Param        english_min_score     query     number   false  "Minimum English score"
// @Param        technical_skill_ids   query     string   false  "Comma-separated skill IDs"
// @Param        computer_skill_ids    query     string   false  "Comma-separated skill IDs"
// @Param        age_min               query     int      false  "Minimum age"
// @Param        age_max               query     int      false  "Maximum age"
// @Param        genders               query     string   false  "Comma-separated genders (MALE,FEMALE)"
// @Param        domicile_cities       query     string   false  "Comma-separated city names"
// @Param        expected_salary_min   query     int      false  "Minimum expected salary"
// @Param        expected_salary_max   query     int      false  "Maximum expected salary"
// @Param        available_start_before query    string   false  "Available start date (YYYY-MM-DD)"
// @Param        education_levels      query     string   false  "Education levels (HIGH_SCHOOL,DIPLOMA,BACHELOR,MASTER)"
// @Param        major_fields          query     string   false  "Comma-separated major fields"
// @Param        total_experience_min  query     int      false  "Minimum total experience in months"
// @Param        total_experience_max  query     int      false  "Maximum total experience in months"
// @Param        page                  query     int      false  "Page number (default: 1)"
// @Param        page_size             query     int      false  "Items per page (default: 20, max: 100)"
// @Param        sort_by               query     string   false  "Sort column (verified_at,japanese_level,age,expected_salary)"
// @Param        sort_order            query     string   false  "Sort order (asc,desc)"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /admin/ats/candidates [get]
func (h *ATSHandler) SearchCandidates(c *gin.Context) {
	filter := domain.ATSFilter{}

	// Parse Japanese Proficiency Group
	if levels := c.Query("japanese_levels"); levels != "" {
		filter.JapaneseLevels = strings.Split(levels, ",")
	}
	if min := c.Query("japan_experience_min"); min != "" {
		if v, err := strconv.Atoi(min); err == nil {
			filter.JapanExperienceMin = &v
		}
	}
	if max := c.Query("japan_experience_max"); max != "" {
		if v, err := strconv.Atoi(max); err == nil {
			filter.JapanExperienceMax = &v
		}
	}
	if lpk := c.Query("has_lpk_training"); lpk != "" {
		v := lpk == "true"
		filter.HasLPKTraining = &v
	}

	// Parse Competency & Language Group
	if certs := c.Query("english_cert_types"); certs != "" {
		filter.EnglishCertTypes = strings.Split(certs, ",")
	}
	if score := c.Query("english_min_score"); score != "" {
		if v, err := strconv.ParseFloat(score, 64); err == nil {
			filter.EnglishMinScore = &v
		}
	}
	if skills := c.Query("technical_skill_ids"); skills != "" {
		filter.TechnicalSkillIDs = parseIntArray(skills)
	}
	if skills := c.Query("computer_skill_ids"); skills != "" {
		filter.ComputerSkillIDs = parseIntArray(skills)
	}

	// Parse Logistics & Availability Group
	if min := c.Query("age_min"); min != "" {
		if v, err := strconv.Atoi(min); err == nil {
			filter.AgeMin = &v
		}
	}
	if max := c.Query("age_max"); max != "" {
		if v, err := strconv.Atoi(max); err == nil {
			filter.AgeMax = &v
		}
	}
	if genders := c.Query("genders"); genders != "" {
		filter.Genders = strings.Split(genders, ",")
	}
	if cities := c.Query("domicile_cities"); cities != "" {
		filter.DomicileCities = strings.Split(cities, ",")
	}
	if min := c.Query("expected_salary_min"); min != "" {
		if v, err := strconv.ParseInt(min, 10, 64); err == nil {
			filter.ExpectedSalaryMin = &v
		}
	}
	if max := c.Query("expected_salary_max"); max != "" {
		if v, err := strconv.ParseInt(max, 10, 64); err == nil {
			filter.ExpectedSalaryMax = &v
		}
	}
	if dateStr := c.Query("available_start_before"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			filter.AvailableStartBefore = &t
		}
	}

	// Parse Education & Experience Group
	if levels := c.Query("education_levels"); levels != "" {
		filter.EducationLevels = strings.Split(levels, ",")
	}
	if majors := c.Query("major_fields"); majors != "" {
		filter.MajorFields = strings.Split(majors, ",")
	}
	if min := c.Query("total_experience_min"); min != "" {
		if v, err := strconv.Atoi(min); err == nil {
			filter.TotalExperienceMin = &v
		}
	}
	if max := c.Query("total_experience_max"); max != "" {
		if v, err := strconv.Atoi(max); err == nil {
			filter.TotalExperienceMax = &v
		}
	}

	// Parse Pagination & Sorting
	filter.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	filter.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	filter.SortBy = c.DefaultQuery("sort_by", "verified_at")
	filter.SortOrder = c.DefaultQuery("sort_order", "desc")

	result, err := h.atsUC.SearchCandidates(c, filter)
	if err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	response.Success(c, http.StatusOK, "Candidates retrieved", result)
}

// ExportCandidates godoc
// @Summary      Export candidates to Excel/CSV
// @Description  Downloads candidates matching the filter criteria as Excel or CSV file
// @Tags         admin-ats
// @Produce      application/octet-stream
// @Security     BearerAuth
// @Param        format               query     string   false  "Export format (xlsx, csv). Default: xlsx"
// @Param        columns              query     string   false  "Comma-separated column names to include"
// @Param        japanese_levels      query     string   false  "Comma-separated JLPT levels"
// @Param        ... (same filters as SearchCandidates)
// @Success      200  {file}    binary
// @Failure      400  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /admin/ats/export [get]
func (h *ATSHandler) ExportCandidates(c *gin.Context) {
	// Parse the same filters as SearchCandidates
	filter := domain.ATSFilter{}

	if levels := c.Query("japanese_levels"); levels != "" {
		filter.JapaneseLevels = strings.Split(levels, ",")
	}
	if min := c.Query("japan_experience_min"); min != "" {
		if v, err := strconv.Atoi(min); err == nil {
			filter.JapanExperienceMin = &v
		}
	}
	if max := c.Query("japan_experience_max"); max != "" {
		if v, err := strconv.Atoi(max); err == nil {
			filter.JapanExperienceMax = &v
		}
	}
	if lpk := c.Query("has_lpk_training"); lpk != "" {
		v := lpk == "true"
		filter.HasLPKTraining = &v
	}
	if certs := c.Query("english_cert_types"); certs != "" {
		filter.EnglishCertTypes = strings.Split(certs, ",")
	}
	if score := c.Query("english_min_score"); score != "" {
		if v, err := strconv.ParseFloat(score, 64); err == nil {
			filter.EnglishMinScore = &v
		}
	}
	if skills := c.Query("technical_skill_ids"); skills != "" {
		filter.TechnicalSkillIDs = parseIntArray(skills)
	}
	if skills := c.Query("computer_skill_ids"); skills != "" {
		filter.ComputerSkillIDs = parseIntArray(skills)
	}
	if min := c.Query("age_min"); min != "" {
		if v, err := strconv.Atoi(min); err == nil {
			filter.AgeMin = &v
		}
	}
	if max := c.Query("age_max"); max != "" {
		if v, err := strconv.Atoi(max); err == nil {
			filter.AgeMax = &v
		}
	}
	if genders := c.Query("genders"); genders != "" {
		filter.Genders = strings.Split(genders, ",")
	}
	if cities := c.Query("domicile_cities"); cities != "" {
		filter.DomicileCities = strings.Split(cities, ",")
	}
	if min := c.Query("expected_salary_min"); min != "" {
		if v, err := strconv.ParseInt(min, 10, 64); err == nil {
			filter.ExpectedSalaryMin = &v
		}
	}
	if max := c.Query("expected_salary_max"); max != "" {
		if v, err := strconv.ParseInt(max, 10, 64); err == nil {
			filter.ExpectedSalaryMax = &v
		}
	}
	if dateStr := c.Query("available_start_before"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			filter.AvailableStartBefore = &t
		}
	}
	if levels := c.Query("education_levels"); levels != "" {
		filter.EducationLevels = strings.Split(levels, ",")
	}
	if majors := c.Query("major_fields"); majors != "" {
		filter.MajorFields = strings.Split(majors, ",")
	}
	if min := c.Query("total_experience_min"); min != "" {
		if v, err := strconv.Atoi(min); err == nil {
			filter.TotalExperienceMin = &v
		}
	}
	if max := c.Query("total_experience_max"); max != "" {
		if v, err := strconv.Atoi(max); err == nil {
			filter.TotalExperienceMax = &v
		}
	}

	// Parse export-specific params
	format := c.DefaultQuery("format", "xlsx")
	var columns []string
	if cols := c.Query("columns"); cols != "" {
		columns = strings.Split(cols, ",")
	}

	req := domain.ATSExportRequest{
		Filter:  filter,
		Columns: columns,
		Format:  format,
	}

	data, filename, err := h.atsUC.ExportCandidates(c, req)
	if err != nil {
		c.Error(apperror.BadRequest(err.Error()))
		return
	}

	// Set content type based on format
	contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if format == "csv" {
		contentType = "text/csv"
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, contentType, data)
}

// GetFilterOptions godoc
// @Summary      Get available filter options
// @Description  Returns all available filter options for the ATS UI
// @Tags         admin-ats
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Router       /admin/ats/filter-options [get]
func (h *ATSHandler) GetFilterOptions(c *gin.Context) {
	options, err := h.atsUC.GetFilterOptions(c)
	if err != nil {
		c.Error(apperror.Internal(err))
		return
	}

	response.Success(c, http.StatusOK, "Filter options retrieved", options)
}

// parseIntArray parses a comma-separated string into an int array
func parseIntArray(s string) []int {
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		if v, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			result = append(result, v)
		}
	}
	return result
}
