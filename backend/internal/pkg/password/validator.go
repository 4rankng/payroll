package password

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// ValidationRule represents a password validation rule
type ValidationRule struct {
	Check   func(string) bool
	Message string
}

// PasswordValidator holds password validation rules
type PasswordValidator struct {
	rules []ValidationRule
}

// NewPasswordValidator creates a new password validator with default Vietnamese requirements
func NewPasswordValidator() *PasswordValidator {
	return &PasswordValidator{
		rules: []ValidationRule{
			{
				Check:   func(p string) bool { return len(p) >= 8 },
				Message: "Mật khẩu phải có ít nhất 8 ký tự",
			},
			{
				Check:   func(p string) bool { return len(p) <= 72 }, // bcrypt limit
				Message: "Mật khẩu không được vượt quá 72 ký tự",
			},
			{
				Check: func(p string) bool {
					for _, r := range p {
						if unicode.IsUpper(r) {
							return true
						}
					}
					return false
				},
				Message: "Mật khẩu phải có ít nhất một chữ cái viết hoa",
			},
			{
				Check: func(p string) bool {
					for _, r := range p {
						if unicode.IsLower(r) {
							return true
						}
					}
					return false
				},
				Message: "Mật khẩu phải có ít nhất một chữ cái viết thường",
			},
			{
				Check: func(p string) bool {
					for _, r := range p {
						if unicode.IsDigit(r) {
							return true
						}
					}
					return false
				},
				Message: "Mật khẩu phải có ít nhất một chữ số",
			},
			{
				Check: func(p string) bool {
					specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?"
					for _, r := range p {
						if strings.ContainsRune(specialChars, r) {
							return true
						}
					}
					return false
				},
				Message: "Mật khẩu phải có ít nhất một ký tự đặc biệt (!@#$%^&*()_+-=[]{}|;:,.<>?)",
			},
		},
	}
}

// Validate validates a password against all rules
func (v *PasswordValidator) Validate(password string) error {
	var errors []string

	for _, rule := range v.rules {
		if !rule.Check(password) {
			errors = append(errors, rule.Message)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("mật khẩu không hợp lệ: %s", strings.Join(errors, "; "))
	}

	return nil
}

// ValidatePasswordStrength returns a password strength score (0-100)
func (v *PasswordValidator) ValidatePasswordStrength(password string) (int, []string) {
	score := 0
	feedback := []string{}

	// Length scoring
	length := len(password)
	if length >= 8 {
		score += 20
	} else {
		feedback = append(feedback, "Tăng độ dài mật khẩu")
	}

	if length >= 12 {
		score += 10
	}

	// Character variety scoring
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`\d`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{}|;:,.<>?]`).MatchString(password)

	varietyCount := 0
	if hasUpper {
		varietyCount++
		score += 15
	} else {
		feedback = append(feedback, "Thêm chữ cái viết hoa")
	}

	if hasLower {
		varietyCount++
		score += 15
	} else {
		feedback = append(feedback, "Thêm chữ cái viết thường")
	}

	if hasDigit {
		varietyCount++
		score += 15
	} else {
		feedback = append(feedback, "Thêm chữ số")
	}

	if hasSpecial {
		varietyCount++
		score += 15
	} else {
		feedback = append(feedback, "Thêm ký tự đặc biệt")
	}

	// Bonus for using all character types
	if varietyCount >= 4 {
		score += 10
	}

	// Skip common pattern penalty

	// Ensure score is within bounds
	if score > 100 {
		score = 100
	} else if score < 0 {
		score = 0
	}

	return score, feedback
}

// AddCustomRule adds a custom validation rule
func (v *PasswordValidator) AddCustomRule(rule ValidationRule) {
	v.rules = append(v.rules, rule)
}

// RemoveRule removes a rule by index
func (v *PasswordValidator) RemoveRule(index int) error {
	if index < 0 || index >= len(v.rules) {
		return fmt.Errorf("invalid rule index")
	}

	v.rules = append(v.rules[:index], v.rules[index+1:]...)
	return nil
}

// GetStrengthLabel returns a Vietnamese label for password strength
func GetStrengthLabel(score int) string {
	if score >= 90 {
		return "Rất mạnh"
	} else if score >= 75 {
		return "Mạnh"
	} else if score >= 50 {
		return "Trung bình"
	} else if score >= 25 {
		return "Yếu"
	} else {
		return "Rất yếu"
	}
}
