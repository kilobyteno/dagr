// Package auth holds authentication helpers shared by the API and workers.
package auth

import (
	"fmt"
	"unicode"
)

// PasswordPolicy is the administrator-configured password complexity rules.
type PasswordPolicy struct {
	MinLength        int  `json:"minLength"`
	RequireUppercase bool `json:"requireUppercase"`
	RequireLowercase bool `json:"requireLowercase"`
	RequireNumber    bool `json:"requireNumber"`
	RequireSymbol    bool `json:"requireSymbol"`
}

// Validate checks password against the policy. Returns a user-facing error message.
func (p PasswordPolicy) Validate(password string) error {
	if len(password) < p.MinLength {
		return fmt.Errorf("password must be at least %d characters", p.MinLength)
	}

	var hasUpper, hasLower, hasNumber, hasSymbol bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsNumber(r):
			hasNumber = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}

	if p.RequireUppercase && !hasUpper {
		return fmt.Errorf("password must include an uppercase letter")
	}
	if p.RequireLowercase && !hasLower {
		return fmt.Errorf("password must include a lowercase letter")
	}
	if p.RequireNumber && !hasNumber {
		return fmt.Errorf("password must include a number")
	}
	if p.RequireSymbol && !hasSymbol {
		return fmt.Errorf("password must include a symbol")
	}
	return nil
}
