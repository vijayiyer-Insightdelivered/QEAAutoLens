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

// extractWithOCR converts PDF pages to images using pdftoppm, then runs
// Tesseract OCR on each image to extract text. This handles scanned /
// image-based PDFs that have no embedded text layer.
//
// Requirements (external tools):
//   - pdftoppm (from poppler-utils)
//   - tesseract (Tesseract OCR engine)
func extractWithOCR(filePath string) ([]string, error) {
	// Check that both tools are available
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return nil, fmt.Errorf("pdftoppm not available (install poppler-utils): %v", err)
	}
	if _, err := exec.LookPath("tesseract"); err != nil {
		return nil, fmt.Errorf("tesseract not available (install tesseract-ocr): %v", err)
	}

	// Create temp directory for intermediate images
	tmpDir, err := os.MkdirTemp("", "ocr-pages-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Convert PDF pages to PNG images using pdftoppm.
	// -png: output PNG format
	// -r 300: 300 DPI for good OCR accuracy
	// Output files will be named like: <prefix>-1.png, <prefix>-2.png, ...
	prefix := filepath.Join(tmpDir, "page")
	cmd := exec.Command("pdftoppm", "-png", "-r", "300", filePath, prefix)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("pdftoppm failed: %v — %s", err, string(output))
	}

	// Find generated page images and sort them by page number
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read temp dir: %v", err)
	}

	var imageFiles []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".ppm") {
			imageFiles = append(imageFiles, filepath.Join(tmpDir, name))
		}
	}

	if len(imageFiles) == 0 {
		return nil, fmt.Errorf("pdftoppm produced no image files")
	}

	// Sort by filename to maintain page order
	sort.Strings(imageFiles)

	// OCR each page image with Tesseract
	var pages []string
	for _, imgPath := range imageFiles {
		text, err := ocrImage(imgPath)
		if err != nil {
			// Log the error but continue with other pages
			continue
		}
		text = strings.TrimSpace(text)
		if text != "" {
			pages = append(pages, text)
		}
	}

	if len(pages) == 0 {
		return nil, fmt.Errorf("tesseract OCR produced no text from %d page image(s)", len(imageFiles))
	}

	return pages, nil
}

// ocrImage runs Tesseract on a single image file and returns the extracted text.
func ocrImage(imagePath string) (string, error) {
	// tesseract <input> stdout  →  writes OCR text to stdout
	// --psm 6: assume a single uniform block of text (good for statement tables)
	cmd := exec.Command("tesseract", imagePath, "stdout", "--psm", "6", "-l", "eng")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tesseract failed on %s: %v", filepath.Base(imagePath), err)
	}
	return string(output), nil
}

// IsOCRAvailable checks whether the external OCR tools (pdftoppm and tesseract)
// are installed and available on the system PATH.
func IsOCRAvailable() bool {
	_, err1 := exec.LookPath("pdftoppm")
	_, err2 := exec.LookPath("tesseract")
	return err1 == nil && err2 == nil
}

// GetPageCount returns the number of pages in a PDF using pdfinfo.
// Returns 0 if pdfinfo is not available or fails.
func getPageCountForOCR(filePath string) int {
	out, err := exec.Command("pdfinfo", filePath).Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Pages:") {
			n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Pages:")))
			if err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}
