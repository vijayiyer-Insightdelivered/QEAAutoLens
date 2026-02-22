package parser

import (
	"fmt"

	"github.com/insightdelivered/bank-statement-converter/internal/models"
)

// Parser defines the interface for bank statement parsers.
type Parser interface {
	// Parse takes raw text from PDF pages and returns structured statement data.
	Parse(pages []string) (*models.StatementInfo, error)
	// BankName returns the human-readable bank name.
	BankName() string
}

// New returns the appropriate parser for the given bank type.
func New(bankType models.BankType) (Parser, error) {
	switch bankType {
	case models.BankMetro:
		return &MetroBankParser{}, nil
	case models.BankHSBC:
		return &HSBCParser{}, nil
	case models.BankBarclays:
		return &BarclaysParser{}, nil
	default:
		return nil, fmt.Errorf("unsupported bank type: %q", bankType)
	}
}

// AutoDetect tries to identify the bank from the PDF text content.
func AutoDetect(pages []string) (models.BankType, error) {
	combined := ""
	for _, p := range pages {
		combined += p + "\n"
	}

	// Check for bank-specific identifiers
	if containsAny(combined, []string{"Metro Bank", "METRO BANK", "metrobankonline"}) {
		return models.BankMetro, nil
	}
	if containsAny(combined, []string{"HSBC", "hsbc.co.uk", "HSBC UK Bank"}) {
		return models.BankHSBC, nil
	}
	if containsAny(combined, []string{"Barclays", "BARCLAYS", "barclays.co.uk"}) {
		return models.BankBarclays, nil
	}

	return "", fmt.Errorf("could not auto-detect bank from statement content; please specify --bank flag")
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if containsIgnoreCase(text, needle) {
			return true
		}
	}
	return false
}

func containsIgnoreCase(text, substr string) bool {
	// Simple case-insensitive contains
	textLower := toLower(text)
	substrLower := toLower(substr)
	return len(substrLower) > 0 && indexOf(textLower, substrLower) >= 0
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func indexOf(s, substr string) int {
	if len(substr) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
