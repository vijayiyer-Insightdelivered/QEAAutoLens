package parser

import (
	"regexp"
	"strings"

	"github.com/insightdelivered/bank-statement-converter/internal/models"
)

// HSBCParser handles HSBC bank statement PDFs.
//
// HSBC statements typically have this layout:
//   Date | Payment type and details | Paid out | Paid in | Balance
//
// Date format: DD Mon YY (e.g., 15 Jan 24) or DD Mon YYYY
// Example: "15 Jan 24  CARD PAYMENT TO TESCO  £25.99  £1,234.56"
type HSBCParser struct{}

func (p *HSBCParser) BankName() string {
	return "HSBC"
}

// HSBC transaction line patterns
// Note: PDF extraction can produce £ as "£", Unicode \u00A3, or omit it entirely.
// We use [£\u00A3]? to handle all cases and \s+ for variable spacing.
var hsbcTxnPattern = regexp.MustCompile(
	`^(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{2,4})\s+` +
		`(.+?)\s{2,}` +
		`[£\x{00A3}]?([\d,]+\.\d{2})?\s+[£\x{00A3}]?([\d,]+\.\d{2})?\s+[£\x{00A3}]?([\d,]+\.\d{2})\s*$`,
)

// Relaxed variant: date + description + any 1-3 amounts separated by whitespace
var hsbcTxnFlexible = regexp.MustCompile(
	`^(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{2,4})\s+` +
		`(.+?)\s+` +
		`[£\x{00A3}]?([\d,]+\.\d{2})?\s*[£\x{00A3}]?([\d,]+\.\d{2})?\s*[£\x{00A3}]?([\d,]+\.\d{2})\s*$`,
)

var hsbcTxnSimple = regexp.MustCompile(
	`^(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{2,4})\s+` +
		`(.+?)\s+[£\x{00A3}]?([\d,]+\.\d{2})\s*$`,
)

// Pattern for DD-Mon-YY format (some HSBC variants)
var hsbcDashDatePattern = regexp.MustCompile(
	`^(\d{1,2}-(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*-\d{2,4})\s+` +
		`(.+?)\s+[£\x{00A3}]?([\d,]+\.\d{2})?\s*[£\x{00A3}]?([\d,]+\.\d{2})?\s*[£\x{00A3}]?([\d,]+\.\d{2})\s*$`,
)

// Pattern for DD/MM/YYYY format (some HSBC statements use this)
var hsbcSlashDatePattern = regexp.MustCompile(
	`^(\d{1,2}/\d{1,2}/\d{2,4})\s+(.+?)\s+` +
		`[£\x{00A3}]?([\d,]+\.\d{2})?\s*[£\x{00A3}]?([\d,]+\.\d{2})?\s*[£\x{00A3}]?([\d,]+\.\d{2})\s*$`,
)

func (p *HSBCParser) Parse(pages []string) (*models.StatementInfo, error) {
	info := &models.StatementInfo{
		Bank: models.BankHSBC,
	}

	allText := strings.Join(pages, "\n")

	info.AccountNumber = findAccountNumber(allText)
	info.SortCode = findSortCode(allText)
	info.AccountHolder = extractNameNearLabel(allText, []string{"Account holder", "Account name", "Mr ", "Mrs ", "Ms ", "Name"})
	info.StatementPeriod = extractPeriod(allText)

	for _, page := range pages {
		lines := strings.Split(page, "\n")
		txns := p.parseLines(lines)
		info.Transactions = append(info.Transactions, txns...)
	}

	return info, nil
}

// normalizeLine cleans up common PDF extraction artifacts.
func normalizeLine(line string) string {
	// Replace Unicode pound sign with ASCII £
	line = strings.ReplaceAll(line, "\u00A3", "£")
	// Collapse multiple spaces to single (but preserve double-space as column separator)
	// Remove zero-width characters
	line = strings.ReplaceAll(line, "\u200B", "")
	line = strings.ReplaceAll(line, "\u00A0", " ") // non-breaking space
	return strings.TrimSpace(line)
}

func (p *HSBCParser) parseLines(lines []string) []models.Transaction {
	var transactions []models.Transaction
	inTransactionSection := false

	for i := 0; i < len(lines); i++ {
		line := normalizeLine(lines[i])
		if line == "" {
			continue
		}

		if containsTransactionHeader(line) {
			inTransactionSection = true
			continue
		}

		if !inTransactionSection && !startsWithDate(line) {
			continue
		}

		if startsWithDate(line) {
			inTransactionSection = true
		}

		// Try strict text-date pattern (DD Mon YY) with double-space column separator
		if txn, ok := p.tryPattern(hsbcTxnPattern, line); ok {
			transactions = append(transactions, txn)
			continue
		}

		// Try flexible text-date pattern (single-space separator)
		if txn, ok := p.tryPattern(hsbcTxnFlexible, line); ok {
			transactions = append(transactions, txn)
			continue
		}

		// Try dash-date pattern (DD-Mon-YY)
		if txn, ok := p.tryPattern(hsbcDashDatePattern, line); ok {
			transactions = append(transactions, txn)
			continue
		}

		// Try slash-date pattern (DD/MM/YYYY)
		if txn, ok := p.tryPattern(hsbcSlashDatePattern, line); ok {
			transactions = append(transactions, txn)
			continue
		}

		// Try simple (date + description + one amount)
		if m := hsbcTxnSimple.FindStringSubmatch(line); m != nil {
			txn := models.Transaction{
				Date:        m[1],
				Description: strings.TrimSpace(m[2]),
			}
			amt, _ := parseAmount(m[3])
			txn.Amount = amt
			if isDebitDescription(txn.Description) {
				txn.Type = "DEBIT"
			} else {
				txn.Type = "CREDIT"
			}
			transactions = append(transactions, txn)
			continue
		}

		// Multi-line description continuation
		if len(transactions) > 0 && !startsWithDate(line) && line != "" && inTransactionSection {
			if !isSummaryLine(line) {
				last := &transactions[len(transactions)-1]
				last.Description += " " + line
			}
		}
	}

	return transactions
}

func (p *HSBCParser) tryPattern(pat *regexp.Regexp, line string) (models.Transaction, bool) {
	m := pat.FindStringSubmatch(line)
	if m == nil {
		return models.Transaction{}, false
	}

	txn := models.Transaction{
		Date:        m[1],
		Description: strings.TrimSpace(m[2]),
	}

	paidOut := strings.TrimSpace(m[3])
	paidIn := strings.TrimSpace(m[4])
	balance := strings.TrimSpace(m[5])

	if paidOut != "" {
		txn.Amount, _ = parseAmount(paidOut)
		txn.Type = "DEBIT"
	} else if paidIn != "" {
		txn.Amount, _ = parseAmount(paidIn)
		txn.Type = "CREDIT"
	}

	if balance != "" {
		txn.Balance, _ = parseAmount(balance)
	}

	return txn, true
}
