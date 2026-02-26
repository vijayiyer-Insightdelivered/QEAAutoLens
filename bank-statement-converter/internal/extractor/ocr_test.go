package extractor

import (
	"os/exec"
	"testing"
)

func TestIsOCRAvailable(t *testing.T) {
	// This test simply verifies the function runs without panic.
	// The result depends on the system's installed tools.
	result := IsOCRAvailable()
	t.Logf("IsOCRAvailable() = %v", result)

	// Verify consistency with direct LookPath checks
	_, err1 := exec.LookPath("pdftoppm")
	_, err2 := exec.LookPath("tesseract")
	expected := err1 == nil && err2 == nil
	if result != expected {
		t.Errorf("IsOCRAvailable() = %v, but direct check says %v", result, expected)
	}
}

func TestExtractWithOCR_MissingTools(t *testing.T) {
	if IsOCRAvailable() {
		t.Skip("OCR tools are installed; cannot test missing-tool error path")
	}

	_, err := extractWithOCR("/nonexistent/file.pdf")
	if err == nil {
		t.Error("expected error when OCR tools are not installed")
	}
}

func TestExtractWithOCR_NonexistentFile(t *testing.T) {
	if !IsOCRAvailable() {
		t.Skip("OCR tools not installed; skipping")
	}

	_, err := extractWithOCR("/tmp/nonexistent-file-12345.pdf")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestGetPageCountForOCR(t *testing.T) {
	// Test with nonexistent file — should return 0 without error
	count := getPageCountForOCR("/tmp/nonexistent-file-12345.pdf")
	if count != 0 {
		t.Errorf("expected 0 pages for nonexistent file, got %d", count)
	}
}
