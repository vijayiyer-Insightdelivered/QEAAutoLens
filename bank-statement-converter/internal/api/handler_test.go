package api

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func setupTestApp() *fiber.App {
	app := fiber.New()
	app.Get("/api/health", HandleHealth)
	app.Post("/api/convert", HandleConvert)
	return app
}

func TestHealthEndpoint(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("GET", "/api/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", result["status"])
	}

	if result["engine"] != "fiber" {
		t.Errorf("expected engine=fiber, got %q", result["engine"])
	}
}

func TestConvertEndpointRequiresFile(t *testing.T) {
	app := setupTestApp()

	req := httptest.NewRequest("POST", "/api/convert", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----test")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Should fail because no file in the body
	if resp.StatusCode == fiber.StatusOK {
		t.Error("expected non-200 for missing file")
	}
}
