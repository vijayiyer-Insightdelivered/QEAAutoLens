package extractor

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ExtractText reads a PDF file and returns the text content of each page.
// It recovers from panics that the underlying PDF library may throw on
// malformed or unusual PDF files.
func ExtractText(filePath string) (pages []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("PDF library crashed (recovered): %v", r)
		}
	}()

	f, r, openErr := pdf.Open(filePath)
	if openErr != nil {
		return nil, fmt.Errorf("failed to open PDF %q: %w", filePath, openErr)
	}
	defer f.Close()

	numPages := r.NumPage()
	if numPages == 0 {
		return nil, fmt.Errorf("PDF %q has no pages", filePath)
	}

	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, extractErr := extractPageText(page)
		if extractErr != nil {
			// If one page fails, include what we have so far and note the error
			pages = append(pages, fmt.Sprintf("[Page %d extraction error: %v]", i, extractErr))
			continue
		}
		pages = append(pages, text)
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("PDF %q: no text could be extracted from any page", filePath)
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
