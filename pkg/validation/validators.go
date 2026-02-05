package validation

import (
	"regexp"
	"time"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// Regex patterns
var (
	// Allow letters, numbers, spaces, and common professional punctuation: . ' - / & ( ) ,
	nameRegex = regexp.MustCompile(`^[\p{L}0-9 .'/&(),-]+$`)

	// E164-like phone: optional +, digits 7-15 length
	phoneRegex = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
)

// RegisterValidators registers custom validators to the validator instance
func RegisterValidators(v *validator.Validate) {
	_ = v.RegisterValidation("valid_name", ValidName)
	_ = v.RegisterValidation("valid_phone", ValidPhone)
	_ = v.RegisterValidation("no_emoji", NoEmoji)
	_ = v.RegisterValidation("max_current_year", MaxCurrentYear)
}

// ValidName validates that a string contains only valid name characters
// Rejects digits and most special symbols
func ValidName(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if val == "" {
		return true // Optional, use required if needed
	}
	return nameRegex.MatchString(val)
}

// ValidPhone validates a phone number structure
func ValidPhone(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if val == "" {
		return true
	}
	return phoneRegex.MatchString(val)
}

// NoEmoji validates that a string does not contain emoji characters
func NoEmoji(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	for _, r := range val {
		// Quick check: most emojis are in higher unicode planes
		// This block rejects many symbols/emojis
		if r > 0x1F000 {
			return false // Supplementary characters (mostly emoji/symbols)
		}
		// Check specific emoji ranges in BMP if needed, but for names/bios,
		// usually IsLetter/IsNumber/IsPunct/IsSpace covers what we want.
		// Detailed emoji detection is complex, but checking for "Symbol" and "Other" categories helps
		if unicode.In(r, unicode.So, unicode.Sk) { // Symbol, other / Symbol, modifier
			return false
		}
	}
	return true
}

// MaxCurrentYear validates that an integer field (year) does not exceed the current year
// This is used for JLPT certificate issue year validation where DB cannot enforce dynamic max
func MaxCurrentYear(fl validator.FieldLevel) bool {
	year := fl.Field().Int()
	if year == 0 {
		return true // Allow zero/nil (optional field)
	}
	currentYear := int64(time.Now().Year())
	return year <= currentYear
}
