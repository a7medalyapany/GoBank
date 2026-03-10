package val

import (
	"fmt"
	"net/mail"
	"regexp"

	"github.com/a7medalyapany/GoBank.git/util"
)

var (
	isValidUsername = regexp.MustCompile(`^[a-z0-9_]+$`).MatchString
	isValidFullname = regexp.MustCompile(`^[a-zA-Z\s]+$`).MatchString
)

func ValidateString(value string, minLength int, maxLength int) error {
	n := len(value)
	if n < minLength || n > maxLength {
		return fmt.Errorf("must be between %d and %d characters", minLength, maxLength)
	}
	return nil
}

func ValidateUsername(username string) error {
	if err := ValidateString(username, 3, 50); err != nil {
		return fmt.Errorf("invalid username: %w", err)
	}
	if !isValidUsername(username) {
		return fmt.Errorf("must contain only lowercase letters, digits, and underscores")
	}
	return nil
}

func ValidatePassword(password string) error {
	if err := ValidateString(password, 8, 100); err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}
	return nil
}

func ValidateEmail(email string) error {
	if err := ValidateString(email, 5, 100); err != nil {
		return fmt.Errorf("invalid email: %w", err)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func ValidateFullname(fullname string) error {
	if err := ValidateString(fullname, 2, 100); err != nil {
		return fmt.Errorf("invalid full name: %w", err)
	}
	if !isValidFullname(fullname) {
		return fmt.Errorf("must contain only letters and spaces")
	}
	return nil
}

func ValidateID(id int64) error {
	if id <= 0 {
		return fmt.Errorf("must be a positive integer")
	}
	return nil
}

func ValidateCurrency(currency string) error {
	if err := ValidateString(currency, 3, 3); err != nil {
		return fmt.Errorf("invalid currency: %w", err)
	}
	if !util.IsSupportedCurrency(currency) {
		return fmt.Errorf("unsupported currency: must be one of USD, EUR, EGP")
	}
	return nil
}

func ValidatePageID(pageID int32) error {
	if pageID < 1 {
		return fmt.Errorf("must be at least 1")
	}
	return nil
}

func ValidatePageSize(pageSize int32) error {
	if pageSize < 1 || pageSize > 100 {
		return fmt.Errorf("must be between 1 and 100")
	}
	return nil
}

func ValidateAmount(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("must be greater than zero")
	}
	return nil
}