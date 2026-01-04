package security

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
)

// FileValidationResult contains the result of file validation
type FileValidationResult struct {
	Valid        bool   // Whether the file passed all validation checks
	Extension    string // Detected file extension
	DetectedMIME string // Detected MIME type
	Error        string // Error message if validation failed
}

// Magic byte signatures for allowed file types
// Maps lowercase extension to possible magic byte prefixes
var magicBytes = map[string][][]byte{
	".jpg":  {{0xFF, 0xD8, 0xFF}},
	".jpeg": {{0xFF, 0xD8, 0xFF}},
	".png":  {{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}},
	".gif":  {{0x47, 0x49, 0x46, 0x38, 0x37, 0x61}, {0x47, 0x49, 0x46, 0x38, 0x39, 0x61}}, // GIF87a & GIF89a
	".webp": {{0x52, 0x49, 0x46, 0x46}},                                                   // RIFF header
	".pdf":  {{0x25, 0x50, 0x44, 0x46}},                                                   // %PDF
	".doc":  {{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}},                           // OLE Compound Document
	".docx": {{0x50, 0x4B, 0x03, 0x04}},                                                   // ZIP (PK..)
	".txt":  {},                                                                           // Text files have no magic bytes - rely on MIME detection
}

// Allowed file extensions (strict whitelist)
var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".txt":  true,
}

// Strict MIME types - DO NOT include application/octet-stream
var strictMIMETypes = map[string]bool{
	// Images
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
	// Documents
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	// Text
	"text/plain": true,
	// ZIP-based documents (DOCX detection fallback)
	"application/zip": true,
}

// ValidateFile performs 3-layer file validation:
// 1. Extension whitelist check
// 2. Magic byte verification (content matches extension)
// 3. MIME type whitelist (application/octet-stream REJECTED)
func ValidateFile(filename string, data []byte, detectedMIME string) FileValidationResult {
	result := FileValidationResult{
		DetectedMIME: detectedMIME,
	}

	// Sanitize and extract extension
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		result.Error = "file has no extension"
		return result
	}
	result.Extension = ext

	// Layer 1: Extension whitelist
	if !allowedExtensions[ext] {
		result.Error = "file extension not allowed: " + ext
		return result
	}

	// Layer 2: Magic byte validation (skip for text files)
	if ext != ".txt" {
		if !validateMagicBytes(ext, data) {
			result.Error = "file content does not match extension (potential file spoofing detected)"
			return result
		}
	}

	// Layer 3: MIME type whitelist
	// CRITICAL: Reject application/octet-stream - it allows arbitrary binary uploads
	if detectedMIME == "application/octet-stream" {
		// Special case: .docx files are sometimes detected as octet-stream
		// Validate via magic bytes already done above, allow if extension is valid
		if ext == ".docx" || ext == ".doc" {
			// Already validated by magic bytes, allow it
		} else {
			result.Error = "binary files not allowed; file type could not be determined"
			return result
		}
	} else if !strictMIMETypes[detectedMIME] {
		result.Error = "MIME type not allowed: " + detectedMIME
		return result
	}

	result.Valid = true
	return result
}

// validateMagicBytes checks if file content starts with expected magic bytes
func validateMagicBytes(ext string, data []byte) bool {
	if len(data) < 4 {
		return false // File too small to validate
	}

	signatures, ok := magicBytes[ext]
	if !ok {
		return false // Unknown extension
	}

	// Empty signatures array = no magic bytes to check (e.g., txt)
	if len(signatures) == 0 {
		return true
	}

	for _, sig := range signatures {
		if len(data) >= len(sig) && bytes.HasPrefix(data, sig) {
			return true
		}
	}

	return false
}

// ValidateFileExtension checks only the extension (for quick pre-validation)
func ValidateFileExtension(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return errors.New("file has no extension")
	}
	if !allowedExtensions[ext] {
		return errors.New("file extension not allowed: " + ext)
	}
	return nil
}

// GetAllowedExtensions returns a list of allowed extensions for error messages
func GetAllowedExtensions() []string {
	extensions := make([]string, 0, len(allowedExtensions))
	for ext := range allowedExtensions {
		extensions = append(extensions, ext)
	}
	return extensions
}

// IsImageExtension checks if the extension is an image type
func IsImageExtension(ext string) bool {
	ext = strings.ToLower(ext)
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp"
}
