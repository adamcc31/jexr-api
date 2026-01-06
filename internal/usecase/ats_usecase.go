package usecase

import (
	"bytes"
	"context"
	"fmt"
	"go-recruitment-backend/internal/domain"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type atsUsecase struct {
	repo domain.ATSRepository
}

// NewATSUsecase creates a new ATS usecase instance
func NewATSUsecase(repo domain.ATSRepository) domain.ATSUsecase {
	return &atsUsecase{repo: repo}
}

// SearchCandidates searches candidates with validation and returns paginated results
func (u *atsUsecase) SearchCandidates(ctx context.Context, filter domain.ATSFilter) (*domain.PaginatedResult[domain.ATSCandidate], error) {
	// Validate and set defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	// Validate age range
	if filter.AgeMin != nil && filter.AgeMax != nil {
		if *filter.AgeMin > *filter.AgeMax {
			return nil, fmt.Errorf("minimum age cannot be greater than maximum age")
		}
	}
	if filter.AgeMin != nil && (*filter.AgeMin < 18 || *filter.AgeMin > 100) {
		return nil, fmt.Errorf("age must be between 18 and 100")
	}
	if filter.AgeMax != nil && (*filter.AgeMax < 18 || *filter.AgeMax > 100) {
		return nil, fmt.Errorf("age must be between 18 and 100")
	}

	// Validate salary range
	if filter.ExpectedSalaryMin != nil && filter.ExpectedSalaryMax != nil {
		if *filter.ExpectedSalaryMin > *filter.ExpectedSalaryMax {
			return nil, fmt.Errorf("minimum salary cannot be greater than maximum salary")
		}
	}

	// Validate experience range
	if filter.TotalExperienceMin != nil && filter.TotalExperienceMax != nil {
		if *filter.TotalExperienceMin > *filter.TotalExperienceMax {
			return nil, fmt.Errorf("minimum experience cannot be greater than maximum experience")
		}
	}

	candidates, total, err := u.repo.SearchCandidates(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search candidates: %w", err)
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &domain.PaginatedResult[domain.ATSCandidate]{
		Data:       candidates,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetFilterOptions returns all available filter options for the UI
func (u *atsUsecase) GetFilterOptions(ctx context.Context) (*domain.ATSFilterOptions, error) {
	return u.repo.GetFilterOptions(ctx)
}

// ExportCandidates exports candidates to Excel or CSV format
func (u *atsUsecase) ExportCandidates(ctx context.Context, req domain.ATSExportRequest) ([]byte, string, error) {
	// Limit export to 10,000 rows
	req.Filter.Page = 1
	req.Filter.PageSize = 10000

	candidates, _, err := u.repo.SearchCandidates(ctx, req.Filter)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch candidates for export: %w", err)
	}

	if len(req.Columns) == 0 {
		req.Columns = domain.ExportableColumns
	}

	// Migrate old column names for backwards compatibility
	for i, col := range req.Columns {
		if col == "has_lpk_training" {
			req.Columns[i] = "lpk_training_name"
		}
	}

	// Remove duplicates (in case both old and new names were sent)
	seen := make(map[string]bool)
	uniqueColumns := make([]string, 0, len(req.Columns))
	for _, col := range req.Columns {
		if !seen[col] {
			seen[col] = true
			uniqueColumns = append(uniqueColumns, col)
		}
	}
	req.Columns = uniqueColumns

	// Validate selected columns
	validColumns := make(map[string]bool)
	for _, col := range domain.ExportableColumns {
		validColumns[col] = true
	}
	for _, col := range req.Columns {
		if !validColumns[col] {
			return nil, "", fmt.Errorf("invalid export column: %s", col)
		}
	}

	switch req.Format {
	case "csv":
		return u.exportCSV(candidates, req.Columns)
	case "xlsx", "":
		return u.exportExcel(candidates, req.Columns)
	default:
		return nil, "", fmt.Errorf("unsupported export format: %s", req.Format)
	}
}

// exportExcel generates an Excel file from candidate data
func (u *atsUsecase) exportExcel(candidates []domain.ATSCandidate, columns []string) ([]byte, string, error) {
	f := excelize.NewFile()
	sheetName := "Candidates"
	f.SetSheetName("Sheet1", sheetName)

	// Column headers with friendly names
	headerNames := map[string]string{
		"full_name":               "FULL NAME",
		"age":                     "AGE",
		"gender":                  "GENDER",
		"domicile_city":           "DOMICILE CITY",
		"marital_status":          "MARITAL STATUS",
		"japanese_level":          "JLPT LEVEL",
		"japan_experience_months": "JAPAN EXPERIENCE (MONTHS)",
		"lpk_training_name":       "LPK TRAINING",
		"english_cert_type":       "ENGLISH CERTIFICATION",
		"english_score":           "ENGLISH SCORE",
		"skills":                  "SKILLS",
		"highest_education":       "EDUCATION",
		"major_field":             "MAJOR FIELD",
		"total_experience_months": "TOTAL EXPERIENCE (MONTHS)",
		"last_position":           "LAST POSITION",
		"expected_salary":         "EXPECTED SALARY (IDR)",
		"available_start_date":    "AVAILABLE START DATE",
		"verification_status":     "VERIFICATION STATUS",
		"verified_at":             "VERIFIED AT",
	}

	// Write headers
	for i, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		headerName := headerNames[col]
		if headerName == "" {
			headerName = col
		}
		f.SetCellValue(sheetName, cell, headerName)
	}

	// Style headers - Dark Blue background with White text
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#1E3A5F"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	endCell, _ := excelize.CoordinatesToCellName(len(columns), 1)
	f.SetCellStyle(sheetName, "A1", endCell, headerStyle)

	// Write data rows
	for rowIdx, candidate := range candidates {
		for colIdx, col := range columns {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			value := u.getCandidateFieldValue(candidate, col)
			// Convert string values to uppercase
			if strVal, ok := value.(string); ok {
				value = strings.ToUpper(strVal)
			}
			f.SetCellValue(sheetName, cell, value)
		}
	}

	// Auto-fit column widths (approximate)
	for i := range columns {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetName, colName, colName, 20)
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, "", fmt.Errorf("failed to write Excel file: %w", err)
	}

	filename := fmt.Sprintf("ats_candidates_%s.xlsx", time.Now().Format("20060102_150405"))
	return buf.Bytes(), filename, nil
}

// exportCSV generates a CSV file from candidate data
func (u *atsUsecase) exportCSV(candidates []domain.ATSCandidate, columns []string) ([]byte, string, error) {
	var buf bytes.Buffer

	// Header row
	buf.WriteString(strings.Join(columns, ",") + "\n")

	// Data rows
	for _, candidate := range candidates {
		var values []string
		for _, col := range columns {
			value := u.getCandidateFieldValue(candidate, col)
			// Escape CSV values and convert to uppercase
			valueStr := strings.ToUpper(fmt.Sprintf("%v", value))
			if strings.Contains(valueStr, ",") || strings.Contains(valueStr, "\"") || strings.Contains(valueStr, "\n") {
				valueStr = "\"" + strings.ReplaceAll(valueStr, "\"", "\"\"") + "\""
			}
			values = append(values, valueStr)
		}
		buf.WriteString(strings.Join(values, ",") + "\n")
	}

	filename := fmt.Sprintf("ats_candidates_%s.csv", time.Now().Format("20060102_150405"))
	return buf.Bytes(), filename, nil
}

// getCandidateFieldValue extracts a field value from candidate struct
func (u *atsUsecase) getCandidateFieldValue(c domain.ATSCandidate, field string) interface{} {
	switch field {
	case "full_name":
		return c.FullName
	case "age":
		if c.Age != nil {
			return *c.Age
		}
		return ""
	case "gender":
		if c.Gender != nil {
			return *c.Gender
		}
		return ""
	case "domicile_city":
		if c.DomicileCity != nil {
			return *c.DomicileCity
		}
		return ""
	case "marital_status":
		if c.MaritalStatus != nil {
			return *c.MaritalStatus
		}
		return ""
	case "japanese_level":
		if c.JapaneseLevel != nil {
			return *c.JapaneseLevel
		}
		return ""
	case "japan_experience_months":
		if c.JapanExperienceMonths != nil {
			return *c.JapanExperienceMonths
		}
		return ""
	case "lpk_training_name":
		if c.LPKTrainingName != nil {
			return *c.LPKTrainingName
		}
		return ""
	case "english_cert_type":
		if c.EnglishCertType != nil {
			return *c.EnglishCertType
		}
		return ""
	case "english_score":
		if c.EnglishScore != nil {
			return strconv.FormatFloat(*c.EnglishScore, 'f', 1, 64)
		}
		return ""
	case "skills":
		if len(c.Skills) > 0 {
			return strings.Join(c.Skills, ", ")
		}
		return ""
	case "highest_education":
		if c.HighestEducation != nil {
			return *c.HighestEducation
		}
		return ""
	case "major_field":
		if c.MajorField != nil {
			return *c.MajorField
		}
		return ""
	case "total_experience_months":
		if c.TotalExperienceMonths != nil {
			return *c.TotalExperienceMonths
		}
		return 0
	case "last_position":
		if c.LastPosition != nil {
			return *c.LastPosition
		}
		return ""
	case "expected_salary":
		if c.ExpectedSalary != nil {
			return *c.ExpectedSalary
		}
		return ""
	case "available_start_date":
		if c.AvailableStartDate != nil {
			return c.AvailableStartDate.Format("2006-01-02")
		}
		return ""
	case "verification_status":
		return c.VerificationStatus
	case "verified_at":
		if c.VerifiedAt != nil {
			return c.VerifiedAt.Format("2006-01-02")
		}
		return ""
	default:
		return ""
	}
}
