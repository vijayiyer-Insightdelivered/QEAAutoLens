package writer

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/insightdelivered/bank-statement-converter/internal/models"
)

// CSVWriter writes transactions to CSV format.
type CSVWriter struct {
	IncludeHeader bool
}

// WriteToFile writes transactions to a CSV file at the given path.
func (w *CSVWriter) WriteToFile(path string, info *models.StatementInfo) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create output file %q: %w", path, err)
	}
	defer f.Close()

	return w.Write(f, info)
}

// Write writes transactions in CSV format to the given writer.
func (w *CSVWriter) Write(out io.Writer, info *models.StatementInfo) error {
	writer := csv.NewWriter(out)
	defer writer.Flush()

	// Write metadata as comments (CSV header rows)
	if w.IncludeHeader {
		if info.Bank != "" {
			writer.Write([]string{"# Bank", string(info.Bank)})
		}
		if info.AccountHolder != "" {
			writer.Write([]string{"# Account Holder", info.AccountHolder})
		}
		if info.AccountNumber != "" {
			writer.Write([]string{"# Account Number", info.AccountNumber})
		}
		if info.SortCode != "" {
			writer.Write([]string{"# Sort Code", info.SortCode})
		}
		if info.StatementPeriod != "" {
			writer.Write([]string{"# Statement Period", info.StatementPeriod})
		}
	}

	// Write column headers
	header := []string{"Date", "Description", "Type", "Amount", "Balance"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write transaction rows
	for _, txn := range info.Transactions {
		row := []string{
			txn.Date,
			txn.Description,
			txn.Type,
			formatAmount(txn.Amount),
			formatAmount(txn.Balance),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

func formatAmount(amount float64) string {
	if amount == 0 {
		return ""
	}
	return strconv.FormatFloat(amount, 'f', 2, 64)
}
