package parser

import (
	"math"
	"regexp"
	"strings"

	"github.com/insightdelivered/bank-statement-converter/internal/models"
)

// MetroBankParser handles Metro Bank statement PDFs.
//
// Metro Bank statements typically have this layout:
//   Date | Transaction type | Description | Paid out | Paid in | Balance
//
// Date format: DD/MM/YYYY
// Example line: "15/01/2024 CARD PAYMENT TESCO STORES 25.99 1,234.56"
type MetroBankParser struct{}

func (p *MetroBankParser) BankName() string {
	return "Metro Bank"
}

// Metro Bank transaction line pattern:
// DATE  DESCRIPTION  [PAID_OUT]  [PAID_IN]  BALANCE
var metroTxnPattern = regexp.MustCompile(
	`^(\d{1,2}/\d{1,2}/\d{2,4})\s+(.+?)` +
		`\s+([\d,]+\.\d{2})?\s*([\d,]+\.\d{2})?\s+([\d,]+\.\d{2})\s*$`,
)

// Simpler pattern for lines with fewer columns
var metroTxnSimple = regexp.MustCompile(
	`^(\d{1,2}/\d{1,2}/\d{2,4})\s+(.+?)\s+([\d,]+\.\d{2})\s*$`,
)

func (p *MetroBankParser) Parse(pages []string) (*models.StatementInfo, error) {
	info := &models.StatementInfo{
		Bank: models.BankMetro,
	}

	allText := strings.Join(pages, "\n")

	// Extract account metadata
	info.AccountNumber = findAccountNumber(allText)
	info.SortCode = findSortCode(allText)
	info.AccountHolder = extractNameNearLabel(allText, []string{"Account holder", "Account name", "Mr ", "Mrs ", "Ms "})
	info.StatementPeriod = extractPeriod(allText)

	for _, page := range pages {
		lines := strings.Split(page, "\n")
		txns := p.parseLines(lines)
		info.Transactions = append(info.Transactions, txns...)
	}

	return info, nil
}

func (p *MetroBankParser) parseLines(lines []string) []models.Transaction {
	var transactions []models.Transaction
	inTransactionSection := false
	var lastBalance float64

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Try to extract opening balance before skipping summary lines
		if bal, ok := extractOpeningBalance(line); ok {
			lastBalance = bal
			continue
		}

		// Detect start of transaction table
		if containsTransactionHeader(line) {
			inTransactionSection = true
			continue
		}

		// Skip non-transaction lines before the table
		if !inTransactionSection && !startsWithDate(line) {
			continue
		}

		if startsWithDate(line) {
			inTransactionSection = true
		}

		// Try full pattern first
		if m := metroTxnPattern.FindStringSubmatch(line); m != nil {
			txn := models.Transaction{
				Date:        m[1],
				Description: strings.TrimSpace(m[2]),
			}

			paidOut := strings.TrimSpace(m[3])
			paidIn := strings.TrimSpace(m[4])
			balance := strings.TrimSpace(m[5])

			if paidOut != "" && paidIn != "" {
				// All three amount columns present — paid out is unambiguous
				amt, _ := parseAmount(paidOut)
				txn.Amount = amt
				txn.Type = "DEBIT"
				txn.Balance, _ = parseAmount(balance)
			} else if paidOut != "" {
				// Only one amount column + balance.
				// The regex always puts the first number in group 3 (paidOut),
				// so we cannot tell from the regex alone whether this is
				// paid out or paid in. Use balance progression to decide.
				amt, _ := parseAmount(paidOut)
				bal, _ := parseAmount(balance)
				txn.Amount = amt
				txn.Balance = bal
				txn.Type = classifyByBalance(amt, bal, lastBalance, txn.Description)
			} else if paidIn != "" {
				amt, _ := parseAmount(paidIn)
				txn.Amount = amt
				txn.Type = "CREDIT"
				txn.Balance, _ = parseAmount(balance)
			}

			if txn.Balance != 0 {
				lastBalance = txn.Balance
			}

			transactions = append(transactions, txn)
			continue
		}

		// Try simpler pattern (just date, description, one amount)
		if m := metroTxnSimple.FindStringSubmatch(line); m != nil {
			txn := models.Transaction{
				Date:        m[1],
				Description: strings.TrimSpace(m[2]),
			}
			amt, _ := parseAmount(m[3])
			txn.Amount = amt
			// Heuristic: if description suggests payment/debit
			if isDebitDescription(txn.Description) {
				txn.Type = "DEBIT"
			} else {
				txn.Type = "CREDIT"
			}
			transactions = append(transactions, txn)
			continue
		}

		// Handle multi-line descriptions: if previous was a transaction
		// and this line doesn't start with a date, append to description
		if len(transactions) > 0 && !startsWithDate(line) && line != "" && inTransactionSection {
			// Check it's not a summary/footer line
			if !isSummaryLine(line) {
				last := &transactions[len(transactions)-1]
				last.Description += " " + line
			}
		}
	}

	return transactions
}

// classifyByBalance determines whether a transaction is DEBIT or CREDIT
// by comparing the amount and current balance against the previous balance.
// Falls back to description-based heuristic when balance info is unavailable.
func classifyByBalance(amt, bal, prevBal float64, desc string) string {
	if prevBal != 0 {
		debitDiff := math.Abs((prevBal - amt) - bal)
		creditDiff := math.Abs((prevBal + amt) - bal)

		if debitDiff < 0.015 && creditDiff >= 0.015 {
			return "DEBIT"
		}
		if creditDiff < 0.015 && debitDiff >= 0.015 {
			return "CREDIT"
		}
		// Both are close (unlikely) or neither matches — use the closer one
		if debitDiff < 0.015 && creditDiff < 0.015 {
			if debitDiff <= creditDiff {
				return "DEBIT"
			}
			return "CREDIT"
		}
	}

	// No usable previous balance — fall back to description heuristic
	if isDebitDescription(desc) {
		return "DEBIT"
	}
	return "CREDIT"
}

// extractOpeningBalance looks for opening/brought-forward balance lines
// and returns the balance amount. Returns (0, false) if not found.
func extractOpeningBalance(line string) (float64, bool) {
	lower := strings.ToLower(line)
	if !strings.Contains(lower, "opening balance") &&
		!strings.Contains(lower, "balance brought forward") &&
		!strings.Contains(lower, "brought forward") {
		return 0, false
	}

	// Find the last amount on the line
	amounts := metroAmountPattern.FindAllString(line, -1)
	if len(amounts) == 0 {
		return 0, false
	}
	bal, err := parseAmount(amounts[len(amounts)-1])
	if err != nil {
		return 0, false
	}
	return bal, true
}

// metroAmountPattern matches numbers like 1,234.56 or 25.99
var metroAmountPattern = regexp.MustCompile(`[\d,]+\.\d{2}`)

func containsTransactionHeader(line string) bool {
	lower := strings.ToLower(line)
	// "paid" is included in the description check because HSBC PDFs use spread
	// characters in headers (e.g. "Pay m e nt t y pe and de t ails") which makes
	// "details" undetectable, but "Paid out" column header remains intact
	return strings.Contains(lower, "date") &&
		(strings.Contains(lower, "description") || strings.Contains(lower, "transaction") ||
			strings.Contains(lower, "details") || strings.Contains(lower, "paid")) &&
		(strings.Contains(lower, "amount") || strings.Contains(lower, "paid") ||
			strings.Contains(lower, "balance") || strings.Contains(lower, "money"))
}

func isDebitDescription(desc string) bool {
	lower := strings.ToLower(desc)
	debitKeywords := []string{
		"card payment", "direct debit", "debit", "payment", "withdrawal",
		"transfer out", "standing order", "dd ", "pos ", "atm ",
		"purchase", "fee", "charge",
	}
	for _, kw := range debitKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func isSummaryLine(line string) bool {
	lower := strings.ToLower(line)
	summaryKeywords := []string{
		"opening balance", "closing balance", "total paid in",
		"total paid out", "total payments", "total receipts",
		"statement period", "page ", "continued",
	}
	for _, kw := range summaryKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func extractNameNearLabel(text string, labels []string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		for _, label := range labels {
			if idx := strings.Index(line, label); idx >= 0 {
				rest := strings.TrimSpace(line[idx+len(label):])
				// Take the rest of the line as the name, up to common delimiters
				if colonIdx := strings.Index(rest, ":"); colonIdx == 0 {
					rest = strings.TrimSpace(rest[1:])
				}
				if rest != "" {
					// Trim trailing numbers or account info
					parts := strings.Split(rest, "  ")
					return strings.TrimSpace(parts[0])
				}
			}
		}
	}
	return ""
}

func extractPeriod(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "statement period") || strings.Contains(lower, "period") {
			// Try to find date range
			dates := datePatternSlash.FindAllString(line, 2)
			if len(dates) == 2 {
				return dates[0] + " to " + dates[1]
			}
			textDates := datePatternText.FindAllString(line, 2)
			if len(textDates) == 2 {
				return textDates[0] + " to " + textDates[1]
			}
		}
	}
	return ""
}
