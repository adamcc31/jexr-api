package validation

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FieldLabels maps struct field names to user-friendly Indonesian labels
var FieldLabels = map[string]string{
	// CandidateProfile fields
	"Title":                    "Judul Profil",
	"Bio":                      "Bio",
	"HighestEducation":         "Pendidikan Terakhir",
	"MajorField":               "Jurusan",
	"DesiredJobPosition":       "Posisi Pekerjaan yang Diinginkan",
	"DesiredJobPositionOther":  "Posisi Pekerjaan Lainnya",
	"PreferredWorkEnvironment": "Lingkungan Kerja Pilihan",
	"CareerGoals3Y":            "Target Karir 3 Tahun",
	"MainConcernsReturning":    "Kekhawatiran Utama",
	"SpecialMessage":           "Pesan Khusus",
	"SkillsOther":              "Keahlian Lainnya",
	"ResumeURL":                "URL Resume",

	// CandidateDetail fields
	"SoftSkillsDescription": "Deskripsi Soft Skills",
	"AppliedWorkValues":     "Nilai Kerja yang Diterapkan",
	"MajorAchievements":     "Pencapaian Utama",

	// Work Experience fields
	"CountryCode":    "Kode Negara",
	"ExperienceType": "Tipe Pengalaman",
	"CompanyName":    "Nama Perusahaan",
	"JobTitle":       "Jabatan",
	"StartDate":      "Tanggal Mulai",
	"EndDate":        "Tanggal Selesai",
	"Description":    "Deskripsi",

	// Certificate fields
	"CertificateType":  "Jenis Sertifikat",
	"CertificateName":  "Nama Sertifikat",
	"ScoreTotal":       "Skor Total",
	"IssuedDate":       "Tanggal Terbit",
	"ExpiresDate":      "Tanggal Kadaluarsa",
	"DocumentFilePath": "File Dokumen",

	// Account Verification / Physical Attributes
	"FirstName":                "Nama Depan",
	"LastName":                 "Nama Belakang",
	"Phone":                    "Nomor Telepon",
	"Occupation":               "Pekerjaan",
	"Intro":                    "Perkenalan",
	"JapanExperienceDuration":  "Durasi Pengalaman di Jepang",
	"JapaneseCertificateURL":   "URL Sertifikat Bahasa Jepang",
	"CvURL":                    "URL CV",
	"JapaneseLevel":            "Level Bahasa Jepang",
	"BirthDate":                "Tanggal Lahir",
	"DomicileCity":             "Kota Domisili",
	"MaritalStatus":            "Status Pernikahan",
	"ChildrenCount":            "Jumlah Anak",
	"HeightCm":                 "Tinggi Badan",
	"WeightKg":                 "Berat Badan",
	"Religion":                 "Agama",
	"JLPTCertificateIssueYear": "Tahun Sertifikat JLPT",
	"WillingToInterviewOnsite": "Kesediaan Interview Onsite",
	"MainJobFields":            "Bidang Pekerjaan Utama",
	"GoldenSkill":              "Keahlian Utama",
	"JapaneseSpeakingLevel":    "Kemampuan Berbicara Jepang",
	"ExpectedSalary":           "Gaji yang Diharapkan",
	"JapanReturnDate":          "Tanggal Kembali dari Jepang",
	"AvailableStartDate":       "Tanggal Mulai Tersedia",
	"PreferredLocations":       "Lokasi Pilihan",
	"PreferredIndustries":      "Industri Pilihan",
	"Gender":                   "Jenis Kelamin",

	// Onboarding fields
	"Interests":          "Minat",
	"LPKSelection":       "Pilihan LPK",
	"CompanyPreferences": "Preferensi Perusahaan",

	// Auth fields
	"Email":           "Email",
	"Password":        "Password",
	"PasswordConfirm": "Konfirmasi Password",
	"Name":            "Nama",

	// Job fields
	"MinSalary":    "Gaji Minimum",
	"MaxSalary":    "Gaji Maksimum",
	"Location":     "Lokasi",
	"Requirements": "Persyaratan",
}

// ValidationRules contains max/min values for validation messages
var ValidationRules = map[string]map[string]interface{}{
	"Bio":                      {"max": 500},
	"Title":                    {"min": 3, "max": 100},
	"HeightCm":                 {"min": 50, "max": 300, "unit": "cm"},
	"WeightKg":                 {"min": 10, "max": 500, "unit": "kg"},
	"JLPTCertificateIssueYear": {"min": 1984, "max": "tahun ini"},
	"Phone":                    {"min": 7, "max": 15},
	"CountryCode":              {"len": 2},
}

// FormatValidationErrors converts validator.ValidationErrors to user-friendly messages
func FormatValidationErrors(err error) []string {
	var messages []string

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		// Not a validation error, return generic message
		return []string{err.Error()}
	}

	for _, e := range validationErrors {
		msg := formatSingleError(e)
		messages = append(messages, msg)
	}

	return messages
}

// formatSingleError formats a single validation error to a user-friendly message
func formatSingleError(e validator.FieldError) string {
	fieldName := e.Field()
	label := getFieldLabel(fieldName)
	tag := e.Tag()
	param := e.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("%s: Wajib diisi", label)

	case "min":
		if rules, ok := ValidationRules[fieldName]; ok {
			if unit, hasUnit := rules["unit"]; hasUnit {
				return fmt.Sprintf("%s: Minimal %s %s", label, param, unit)
			}
		}
		if e.Kind().String() == "string" {
			return fmt.Sprintf("%s: Minimal %s karakter", label, param)
		}
		return fmt.Sprintf("%s: Minimal %s", label, param)

	case "max":
		if rules, ok := ValidationRules[fieldName]; ok {
			if unit, hasUnit := rules["unit"]; hasUnit {
				return fmt.Sprintf("%s: Maksimal %s %s", label, param, unit)
			}
		}
		if e.Kind().String() == "string" {
			return fmt.Sprintf("%s: Maksimal %s karakter", label, param)
		}
		return fmt.Sprintf("%s: Maksimal %s", label, param)

	case "len":
		return fmt.Sprintf("%s: Harus tepat %s karakter", label, param)

	case "oneof":
		options := formatOneOfOptions(param)
		return fmt.Sprintf("%s: Harus salah satu dari: %s", label, options)

	case "email":
		return fmt.Sprintf("%s: Format email tidak valid", label)

	case "url":
		return fmt.Sprintf("%s: Format URL tidak valid", label)

	case "valid_name":
		return fmt.Sprintf("%s: Hanya boleh huruf, spasi, dan tanda baca umum (. ' - /)", label)

	case "valid_phone":
		return fmt.Sprintf("%s: Format nomor telepon tidak valid (7-15 digit, dengan/tanpa +)", label)

	case "no_emoji":
		return fmt.Sprintf("%s: Tidak boleh mengandung emoji atau simbol khusus", label)

	case "max_current_year":
		return fmt.Sprintf("%s: Tidak boleh melebihi tahun ini", label)

	case "eqfield":
		paramLabel := getFieldLabel(param)
		return fmt.Sprintf("%s: Harus sama dengan %s", label, paramLabel)

	case "gtfield":
		paramLabel := getFieldLabel(param)
		return fmt.Sprintf("%s: Harus lebih besar dari %s", label, paramLabel)

	case "ltfield":
		paramLabel := getFieldLabel(param)
		return fmt.Sprintf("%s: Harus lebih kecil dari %s", label, paramLabel)

	default:
		// Fallback for unknown tags
		return fmt.Sprintf("%s: Validasi gagal (%s)", label, tag)
	}
}

// getFieldLabel returns the user-friendly label for a field
func getFieldLabel(fieldName string) string {
	if label, ok := FieldLabels[fieldName]; ok {
		return label
	}
	// Return field name with spaces between camelCase words
	return formatCamelCase(fieldName)
}

// formatCamelCase converts CamelCase to spaced words
func formatCamelCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}
	return result.String()
}

// formatOneOfOptions formats oneof options for display
func formatOneOfOptions(param string) string {
	options := strings.Split(param, " ")
	formatted := make([]string, len(options))
	for i, opt := range options {
		formatted[i] = formatEnumValue(opt)
	}
	return strings.Join(formatted, ", ")
}

// formatEnumValue formats enum values for display
func formatEnumValue(value string) string {
	// Map common enum values to Indonesian
	enumLabels := map[string]string{
		"MALE":      "Laki-laki",
		"FEMALE":    "Perempuan",
		"SINGLE":    "Belum Menikah",
		"MARRIED":   "Menikah",
		"DIVORCED":  "Cerai",
		"LOCAL":     "Lokal",
		"OVERSEAS":  "Luar Negeri",
		"TOEFL":     "TOEFL",
		"IELTS":     "IELTS",
		"TOEIC":     "TOEIC",
		"OTHER":     "Lainnya",
		"ISLAM":     "Islam",
		"KRISTEN":   "Kristen",
		"KATOLIK":   "Katolik",
		"HINDU":     "Hindu",
		"BUDDHA":    "Buddha",
		"KONGHUCU":  "Konghucu",
		"N1":        "N1",
		"N2":        "N2",
		"N3":        "N3",
		"N4":        "N4",
		"N5":        "N5",
		"NATIVE":    "Native",
		"FLUENT":    "Lancar",
		"BASIC":     "Dasar",
		"PASSIVE":   "Pasif",
		"ADMIN":     "Admin",
		"EMPLOYER":  "Employer",
		"CANDIDATE": "Candidate",
	}

	if label, ok := enumLabels[value]; ok {
		return label
	}
	return value
}
