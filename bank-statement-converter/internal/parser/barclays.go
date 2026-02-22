package parser

import (
	"regexp"
	"strings"

	"github.com/insightdelivered/bank-statement-converter/internal/models"
)

// BarclaysParser handles Barclays bank statement PDFs.
//
// Barclays statements typically have this layout:
//   Date | Description | Money out | Money in | Balance
//
// Date format: DD/MM/YYYY or DD Mon YYYY
// Example: "15/01/2024  CARD PAYMENT TO TESCO STORES 2602  25.99  1,234.56"
type BarclaysParser struct{}

func (p *BarclaysParser) BankName() string {
	return "Barclays"
}

// Barclays uses "Money out" and "Money in" columns
var barclaysTxnPattern = regexp.MustCompile(
	`^(\d{1,2}/\d{1,2}/\d{2,4})\s+(.+?)\s+` +
		`£?([\d,]+\.\d{2})?\s*£?([\d,]+\.\d{2})?\s*£?([\d,]+\.\d{2})\s*$`,
)

var barclaysTxnSimple = regexp.MustCompile(
	`^(\d{1,2}/\d{1,2}/\d{2,4})\s+(.+?)\s+£?([\d,]+\.\d{2})\s*$`,
)

// Some Barclays statements use text dates
var barclaysTextDatePattern = regexp.MustCompile(
	`^(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{2,4})\s+` +
		`(.+?)\s+£?([\d,]+\.\d{2})?\s*£?([\d,]+\.\d{2})?\s*£?([\d,]+\.\d{2})\s*$`,
)

// Barclays sometimes uses format: DD Mon  Description  Amount  Balance
var barclaysCompactPattern = regexp.MustCompile(
	`^(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*)\s+` +
		`(.+?)\s+£?([\d,]+\.\d{2})\s+£?([\d,]+\.\d{2})\s*$`,
)

func (p *BarclaysParser) Parse(pages []string) (*models.StatementInfo, error) {
	info := &models.StatementInfo{
		Bank: models.BankBarclays,
	}

	allText := strings.Join(pages, "\n")

	info.AccountNumber = findAccountNumber(allText)
	info.SortCode = findSortCode(allText)
	info.AccountHolder = extractBarclaysName(allText)
	info.StatementPeriod = extractPeriod(allText)

	for _, page := range pages {
		lines := strings.Split(page, "\n")
		txns := p.parseLines(lines)
		info.Transactions = append(info.Transactions, txns...)
	}

	return info, nil
}

func (p *BarclaysParser) parseLines(lines []string) []models.Transaction {
	var transactions []models.Transaction
	inTransactionSection := false

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if containsBarclaysHeader(line) {
			inTransactionSection = true
			continue
		}

		if !inTransactionSection && !startsWithDate(line) {
			continue
		}

		if startsWithDate(line) {
			inTransactionSection = true
		}

		// Try full pattern with slash dates (DD/MM/YYYY)
		if txn, ok := p.tryFullPattern(barclaysTxnPattern, line); ok {
			transactions = append(transactions, txn)
			continue
		}

		// Try text date pattern (DD Mon YYYY)
		if txn, ok := p.tryFullPattern(barclaysTextDatePattern, line); ok {
			transactions = append(transactions, txn)
			continue
		}

		// Try compact pattern (DD Mon  Description  Amount  Balance)
		if m := barclaysCompactPattern.FindStringSubmatch(line); m != nil {
			txn := models.Transaction{
				Date:        m[1],
				Description: strings.TrimSpace(m[2]),
			}
			amt, _ := parseAmount(m[3])
			txn.Amount = amt
			txn.Balance, _ = parseAmount(m[4])

			if isDebitDescription(txn.Description) {
				txn.Type = "DEBIT"
			} else {
				txn.Type = "CREDIT"
			}
			transactions = append(transactions, txn)
			continue
		}

		// Simple pattern: just date + description + one amount
		if m := barclaysTxnSimple.FindStringSubmatch(line); m != nil {
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
			if !isSummaryLine(line) && !isBarclaysFooter(line) {
				last := &transactions[len(transactions)-1]
				last.Description += " " + line
			}
		}
	}

	return transactions
}

func (p *BarclaysParser) tryFullPattern(pat *regexp.Regexp, line string) (models.Transaction, bool) {
	m := pat.FindStringSubmatch(line)
	if m == nil {
		return models.Transaction{}, false
	}

	txn := models.Transaction{
		Date:        m[1],
		Description: strings.TrimSpace(m[2]),
	}

	moneyOut := strings.TrimSpace(m[3])
	moneyIn := strings.TrimSpace(m[4])
	balance := strings.TrimSpace(m[5])

	if moneyOut != "" {
		txn.Amount, _ = parseAmount(moneyOut)
		txn.Type = "DEBIT"
	} else if moneyIn != "" {
		txn.Amount, _ = parseAmount(moneyIn)
		txn.Type = "CREDIT"
	}

	if balance != "" {
		txn.Balance, _ = parseAmount(balance)
	}

	return txn, true
}

func containsBarclaysHeader(line string) bool {
	lower := strings.ToLower(line)
	return (strings.Contains(lower, "date") &&
		(strings.Contains(lower, "money out") || strings.Contains(lower, "money in") ||
			strings.Contains(lower, "description") || strings.Contains(lower, "details"))) ||
		containsTransactionHeader(line)
}

func isBarclaysFooter(line string) bool {
	lower := strings.ToLower(line)
	footerKeywords := []string{
		"barclays bank", "registered in", "authorised by",
		"financial conduct", "please check", "if you find",
	}
	for _, kw := range footerKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func extractBarclaysName(text string) string {
	// Barclays often has name at the top of the statement
	name := extractNameNearLabel(text, []string{"Account holder", "Account name", "Mr ", "Mrs ", "Ms ", "Miss "})
	if name != "" {
		return name
	}

	// Try to find name after sort code/account number line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.Contains(line, "Sort code") || strings.Contains(line, "Account number") {
			if i+1 < len(lines) {
				candidate := strings.TrimSpace(lines[i+1])
				if candidate != "" && !strings.ContainsAny(candidate, "0123456789") {
					return candidate
				}
			}
		}
	}

	return ""
}
