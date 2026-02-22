package extractor

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ExtractText reads a PDF file and returns the text content of each page.
func ExtractText(filePath string) ([]string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF %q: %w", filePath, err)
	}
	defer f.Close()

	numPages := r.NumPage()
	if numPages == 0 {
		return nil, fmt.Errorf("PDF %q has no pages", filePath)
	}

	var pages []string
	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := extractPageText(page)
		if err != nil {
			return nil, fmt.Errorf("failed to extract text from page %d: %w", i, err)
		}
		pages = append(pages, text)
	}

	return pages, nil
}

// ExtractTextCombined reads a PDF and returns all text combined into one string.
func ExtractTextCombined(filePath string) (string, error) {
	pages, err := ExtractText(filePath)
	if err != nil {
		return "", err
	}
	return strings.Join(pages, "\n\n"), nil
}

func extractPageText(page pdf.Page) (string, error) {
	rows, err := page.GetTextByRow()
	if err != nil {
		return "", err
	}

	var lines []string
	for _, row := range rows {
		var parts []string
		for _, word := range row.Content {
			parts = append(parts, word.S)
		}
		line := strings.Join(parts, " ")
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n"), nil
}
