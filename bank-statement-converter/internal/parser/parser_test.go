package parser

import (
	"testing"

	"github.com/insightdelivered/bank-statement-converter/internal/models"
)

func TestAutoDetect(t *testing.T) {
	tests := []struct {
		name     string
		pages    []string
		expected models.BankType
		wantErr  bool
	}{
		{
			name:     "detects Metro Bank",
			pages:    []string{"Metro Bank\nAccount Statement\n15/01/2024"},
			expected: models.BankMetro,
		},
		{
			name:     "detects HSBC",
			pages:    []string{"HSBC UK Bank plc\nYour Statement\n15 Jan 2024"},
			expected: models.BankHSBC,
		},
		{
			name:     "detects Barclays",
			pages:    []string{"Barclays Bank UK PLC\nStatement\n15/01/2024"},
			expected: models.BankBarclays,
		},
		{
			name:    "unknown bank returns error",
			pages:   []string{"Some Unknown Bank\nStatement"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AutoDetect(tt.pages)
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
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		bankType models.BankType
		wantName string
		wantErr  bool
	}{
		{models.BankMetro, "Metro Bank", false},
		{models.BankHSBC, "HSBC", false},
		{models.BankBarclays, "Barclays", false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.bankType), func(t *testing.T) {
			p, err := New(tt.bankType)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.BankName() != tt.wantName {
				t.Errorf("got %q, want %q", p.BankName(), tt.wantName)
			}
		})
	}
}
