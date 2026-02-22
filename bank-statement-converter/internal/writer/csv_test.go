package writer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/insightdelivered/bank-statement-converter/internal/models"
)

func TestCSVWriter_Write(t *testing.T) {
	info := &models.StatementInfo{
		Bank:            models.BankMetro,
		AccountHolder:   "John Smith",
		AccountNumber:   "12345678",
		SortCode:        "23-05-80",
		StatementPeriod: "01/01/2024 to 31/01/2024",
		Transactions: []models.Transaction{
			{Date: "15/01/2024", Description: "CARD PAYMENT TESCO", Type: "DEBIT", Amount: 25.99, Balance: 1234.56},
			{Date: "16/01/2024", Description: "SALARY", Type: "CREDIT", Amount: 2500.00, Balance: 3734.56},
		},
	}

	var buf bytes.Buffer
	w := &CSVWriter{IncludeHeader: true}
	err := w.Write(&buf, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Check metadata headers
	if !strings.Contains(output, "# Bank") {
		t.Error("expected bank metadata header")
	}
	if !strings.Contains(output, "# Account Holder") {
		t.Error("expected account holder metadata")
	}

	// Check column headers
	if !strings.Contains(output, "Date,Description,Type,Amount,Balance") {
		t.Error("expected column headers")
	}

	// Check transaction data
	if !strings.Contains(output, "15/01/2024") {
		t.Error("expected first transaction date")
	}
	if !strings.Contains(output, "CARD PAYMENT TESCO") {
		t.Error("expected first transaction description")
	}
	if !strings.Contains(output, "25.99") {
		t.Error("expected first transaction amount")
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	// 5 metadata lines + 1 header + 2 transactions = 8
	if len(lines) != 8 {
		t.Errorf("expected 8 lines, got %d", len(lines))
	}
}

func TestCSVWriter_WriteNoHeader(t *testing.T) {
	info := &models.StatementInfo{
		Bank: models.BankHSBC,
		Transactions: []models.Transaction{
			{Date: "15/01/2024", Description: "PAYMENT", Type: "DEBIT", Amount: 10.00},
		},
	}

	var buf bytes.Buffer
	w := &CSVWriter{IncludeHeader: false}
	err := w.Write(&buf, info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Should NOT have metadata
	if strings.Contains(output, "# Bank") {
		t.Error("should not have bank metadata when header=false")
	}

	// Should still have column headers
	if !strings.Contains(output, "Date,Description,Type,Amount,Balance") {
		t.Error("expected column headers even without metadata")
	}
}

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{25.99, "25.99"},
		{1234.56, "1234.56"},
		{0, ""},
		{2500.00, "2500.00"},
	}

	for _, tt := range tests {
		got := formatAmount(tt.input)
		if got != tt.expected {
			t.Errorf("formatAmount(%f): got %q, want %q", tt.input, got, tt.expected)
		}
	}
}
