package parser

import (
	"testing"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		wantErr  bool
	}{
		{"25.99", 25.99, false},
		{"1,234.56", 1234.56, false},
		{"£25.99", 25.99, false},
		{"-25.99", -25.99, false},
		{"£1,234,567.89", 1234567.89, false},
		{"0.00", 0.00, false},
		{"", 0, false},
		{" 25.99 ", 25.99, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseAmount(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("got %f, want %f", got, tt.expected)
			}
		})
	}
}

func TestStartsWithDate(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"15/01/2024 CARD PAYMENT", true},
		{"1/1/24 PAYMENT", true},
		{"15 Jan 2024 CARD PAYMENT", true},
		{"15-Jan-2024 PAYMENT", true},
		{"CARD PAYMENT 15/01/2024", false},
		{"not a date line", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := startsWithDate(tt.input)
			if got != tt.expected {
				t.Errorf("startsWithDate(%q): got %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"15/01/2024 CARD PAYMENT", "15/01/2024"},
		{"15 Jan 2024 CARD PAYMENT", "15 Jan 2024"},
		{"15-Jan-2024 PAYMENT", "15-Jan-2024"},
		{"not a date", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractDate(tt.input)
			if got != tt.expected {
				t.Errorf("extractDate(%q): got %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFindAccountNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Account number: 12345678", "12345678"},
		{"Account: 87654321 Sort code: 20-00-00", "87654321"},
		{"no account here", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := findAccountNumber(tt.input)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFindSortCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Sort code: 20-00-00", "20-00-00"},
		{"Sort code 40-12-34 Account", "40-12-34"},
		{"no sort code", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := findSortCode(tt.input)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
