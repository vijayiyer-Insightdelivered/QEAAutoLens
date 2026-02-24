package extractor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ExtractTextOCR converts PDF pages to images and runs Tesseract OCR.
// This handles scanned/image-based PDFs that have no text layer.
// Requires: pdftoppm (poppler-utils) and tesseract (tesseract-ocr).
func ExtractTextOCR(filePath string) ([]string, error) {
	// Check that required tools are available
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return nil, fmt.Errorf("pdftoppm not available (install poppler-utils): %v", err)
	}
	if _, err := exec.LookPath("tesseract"); err != nil {
		return nil, fmt.Errorf("tesseract not available (install tesseract-ocr): %v", err)
	}

	// Create temp directory for images
	tmpDir, err := os.MkdirTemp("", "ocr-pages-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Convert PDF pages to PNG images using pdftoppm
	// -r 300 = 300 DPI for good OCR quality
	// -png = PNG format output
	imgPrefix := filepath.Join(tmpDir, "page")
	cmd := exec.Command("pdftoppm", "-r", "300", "-png", filePath, imgPrefix)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("pdftoppm failed: %v (output: %s)", err, string(out))
	}

	// Find all generated page images, sorted by name
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read temp dir: %v", err)
	}

	var imageFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".png") {
			imageFiles = append(imageFiles, filepath.Join(tmpDir, e.Name()))
		}
	}
	sort.Strings(imageFiles)

	if len(imageFiles) == 0 {
		return nil, fmt.Errorf("pdftoppm produced no page images")
	}

	// OCR each page image with Tesseract
	var pages []string
	for _, imgFile := range imageFiles {
		// tesseract <input> <output_base> -l eng
		// Output goes to <output_base>.txt
		outBase := strings.TrimSuffix(imgFile, ".png") + "-ocr"
		// PSM 4 = assume single column of text of variable sizes (good for statements)
		cmd := exec.Command("tesseract", imgFile, outBase, "-l", "eng", "--psm", "4")
		if out, err := cmd.CombinedOutput(); err != nil {
			// Log but continue — some pages might work
			fmt.Fprintf(os.Stderr, "tesseract warning for %s: %v (output: %s)\n", imgFile, err, string(out))
			continue
		}

		outFile := outBase + ".txt"
		data, err := os.ReadFile(outFile)
		if err != nil {
			continue
		}

		text := strings.TrimSpace(string(data))
		if text != "" {
			pages = append(pages, text)
		}
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("tesseract OCR produced no text from %d page images", len(imageFiles))
	}

	return pages, nil
}

// ocrPageCount returns the number of pages in a PDF using pdfinfo.
func ocrPageCount(filePath string) int {
	out, err := exec.Command("pdfinfo", filePath).Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Pages:") {
			n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Pages:")))
			if err == nil {
				return n
			}
		}
	}
	return 0
}
