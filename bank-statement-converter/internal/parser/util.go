package parser

import (
	"regexp"
	"strconv"
	"strings"
)

// Common date patterns found in UK bank statements.
var (
	// DD/MM/YYYY or DD/MM/YY
	datePatternSlash = regexp.MustCompile(`\b(\d{1,2}/\d{1,2}/\d{2,4})\b`)
	// DD Mon YYYY (e.g., 15 Jan 2024) — case-insensitive via alternation
	datePatternText = regexp.MustCompile(`(?i)\b(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{2,4})\b`)
	// DD-Mon-YYYY or DD-Mon-YY
	datePatternDash = regexp.MustCompile(`(?i)\b(\d{1,2}-(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*-\d{2,4})\b`)
	// DD Mon without year (e.g., "4 Dec", "15 Jan") — used by Barclays business statements
	datePatternShort = regexp.MustCompile(`(?i)^(\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec))(?:\s|→|$)`)
)

// parseAmount converts a string like "1,234.56" or "-£1,234.56" to a float64.
func parseAmount(s string) (float64, error) {
	s = strings.TrimSpace(s)
	// Remove currency symbols and whitespace (including Unicode variants)
	s = strings.ReplaceAll(s, "£", "")
	s = strings.ReplaceAll(s, "\u00A3", "") // Unicode pound sign
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "€", "")
	s = strings.ReplaceAll(s, "\u20AC", "") // Unicode euro sign
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\u00A0", "") // non-breaking space

	if s == "" || s == "-" {
		return 0, nil
	}

	return strconv.ParseFloat(s, 64)
}

// sanitizeOCRAmounts fixes common OCR errors in amount strings.
// Tesseract often misreads periods as semicolons or colons in numbers.
// E.g., "19,720; 15:" → "19,720.15", "1.00" stays "1.00".
func sanitizeOCRAmounts(line string) string {
	// Fix semicolons used as periods in numbers: "1,234; 56" → "1,234.56"
	line = regexp.MustCompile(`(\d);(\s*)(\d)`).ReplaceAllString(line, "$1.$3")
	// Fix colons used as periods in numbers: "1,234:56" → "1,234.56"
	line = regexp.MustCompile(`(\d):(\d)`).ReplaceAllString(line, "$1.$2")
	// Remove trailing colons after digits: "19,720.15:" → "19,720.15"
	line = regexp.MustCompile(`(\d):\s`).ReplaceAllString(line, "$1 ")
	line = regexp.MustCompile(`(\d):$`).ReplaceAllString(line, "$1")
	// Strip "NA" that OCR appends after amounts
	line = regexp.MustCompile(`\s+NA\b`).ReplaceAllString(line, "")
	return line
}

// startsWithDate checks if a line begins with a date pattern.
func startsWithDate(line string) bool {
	line = strings.TrimSpace(line)
	if datePatternSlash.MatchString(line) {
		loc := datePatternSlash.FindStringIndex(line)
		if loc != nil && loc[0] < 3 {
			return true
		}
	}
	if datePatternText.MatchString(line) {
		loc := datePatternText.FindStringIndex(line)
		if loc != nil && loc[0] < 3 {
			return true
		}
	}
	if datePatternDash.MatchString(line) {
		loc := datePatternDash.FindStringIndex(line)
		if loc != nil && loc[0] < 3 {
			return true
		}
	}
	return false
}

// extractDate returns the first date found at the start of a line.
func extractDate(line string) string {
	line = strings.TrimSpace(line)

	if m := datePatternSlash.FindString(line); m != "" {
		loc := datePatternSlash.FindStringIndex(line)
		if loc != nil && loc[0] < 3 {
			return m
		}
	}
	if m := datePatternText.FindString(line); m != "" {
		loc := datePatternText.FindStringIndex(line)
		if loc != nil && loc[0] < 3 {
			return m
		}
	}
	if m := datePatternDash.FindString(line); m != "" {
		loc := datePatternDash.FindStringIndex(line)
		if loc != nil && loc[0] < 3 {
			return m
		}
	}
	return ""
}

// startsWithShortDate checks if a line starts with "D Mon" format (no year).
func startsWithShortDate(line string) bool {
	return datePatternShort.MatchString(strings.TrimSpace(line))
}

// extractShortDate returns the "D Mon" date from the start of a line, or "".
func extractShortDate(line string) string {
	m := datePatternShort.FindStringSubmatch(strings.TrimSpace(line))
	if m != nil {
		return m[1]
	}
	return ""
}

// splitFields splits a line into whitespace-separated fields,
// but preserves quoted strings and description text.
func splitFields(line string) []string {
	return strings.Fields(line)
}

// extractAccountNumber finds typical UK bank account numbers (8 digits).
var accountNumberPattern = regexp.MustCompile(`\b(\d{8})\b`)

// extractSortCode finds typical UK sort codes (XX-XX-XX).
var sortCodePattern = regexp.MustCompile(`\b(\d{2}-\d{2}-\d{2})\b`)

func findAccountNumber(text string) string {
	m := accountNumberPattern.FindString(text)
	return m
}

func findSortCode(text string) string {
	m := sortCodePattern.FindString(text)
	return m
}
