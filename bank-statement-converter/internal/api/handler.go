package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/insightdelivered/bank-statement-converter/internal/extractor"
	"github.com/insightdelivered/bank-statement-converter/internal/models"
	"github.com/insightdelivered/bank-statement-converter/internal/parser"
	"github.com/insightdelivered/bank-statement-converter/internal/writer"
)

// ConvertResponse is the JSON response from the /api/convert endpoint.
type ConvertResponse struct {
	Success      bool                  `json:"success"`
	Error        string                `json:"error,omitempty"`
	Bank         string                `json:"bank,omitempty"`
	AccountInfo  *AccountInfo          `json:"accountInfo,omitempty"`
	Transactions []models.Transaction  `json:"transactions,omitempty"`
	CSV          string                `json:"csv,omitempty"`
	TotalDebit   float64               `json:"totalDebit"`
	TotalCredit  float64               `json:"totalCredit"`
	Count        int                   `json:"count"`
}

// AccountInfo holds account metadata for the JSON response.
type AccountInfo struct {
	Holder   string `json:"holder,omitempty"`
	Number   string `json:"number,omitempty"`
	SortCode string `json:"sortCode,omitempty"`
	Period   string `json:"period,omitempty"`
}

// Handler holds the HTTP handlers for the API.
type Handler struct {
	StaticDir string
}

// RegisterRoutes sets up the HTTP routes.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/convert", h.handleConvert)
	mux.HandleFunc("/api/health", h.handleHealth)

	// Serve React static files
	if h.StaticDir != "" {
		fs := http.FileServer(http.Dir(h.StaticDir))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// For SPA: serve index.html for non-file routes
			path := r.URL.Path
			if path != "/" && !strings.HasPrefix(path, "/api/") {
				fullPath := filepath.Join(h.StaticDir, path)
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					http.ServeFile(w, r, filepath.Join(h.StaticDir, "index.html"))
					return
				}
			}
			fs.ServeHTTP(w, r)
		})
	}
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"version": "1.0.0",
	})
}

func (h *Handler) handleConvert(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	// Parse multipart form (max 32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse form: %v", err))
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "No file uploaded. Use form field 'file'.")
		return
	}
	defer file.Close()

	// Validate it's a PDF
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".pdf") {
		writeError(w, http.StatusBadRequest, "Only PDF files are supported.")
		return
	}

	// Get optional bank parameter
	bankParam := r.FormValue("bank")
	includeHeader := r.FormValue("header") != "false"

	// Save uploaded file to temp location
	tmpFile, err := os.CreateTemp("", "statement-*.pdf")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create temp file.")
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, file); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save uploaded file.")
		return
	}
	tmpFile.Close()

	// Extract text from PDF
	pages, err := extractor.ExtractText(tmpFile.Name())
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("PDF extraction failed: %v", err))
		return
	}

	// Determine bank type
	var bankType models.BankType
	if bankParam != "" {
		switch strings.ToLower(bankParam) {
		case "metro", "metrobank":
			bankType = models.BankMetro
		case "hsbc":
			bankType = models.BankHSBC
		case "barclays":
			bankType = models.BankBarclays
		default:
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Unknown bank: %q. Use metro, hsbc, or barclays.", bankParam))
			return
		}
	} else {
		detected, err := parser.AutoDetect(pages)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		bankType = detected
	}

	// Parse
	p, err := parser.New(bankType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	info, err := p.Parse(pages)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("Parsing failed: %v", err))
		return
	}

	// Generate CSV string
	var csvBuf bytes.Buffer
	csvWriter := &writer.CSVWriter{IncludeHeader: includeHeader}
	if err := csvWriter.Write(&csvBuf, info); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("CSV generation failed: %v", err))
		return
	}

	// Calculate totals
	var totalDebit, totalCredit float64
	for _, txn := range info.Transactions {
		if txn.Type == "DEBIT" {
			totalDebit += txn.Amount
		} else {
			totalCredit += txn.Amount
		}
	}

	resp := ConvertResponse{
		Success:      true,
		Bank:         string(bankType),
		Transactions: info.Transactions,
		CSV:          csvBuf.String(),
		TotalDebit:   totalDebit,
		TotalCredit:  totalCredit,
		Count:        len(info.Transactions),
	}

	if info.AccountHolder != "" || info.AccountNumber != "" || info.SortCode != "" || info.StatementPeriod != "" {
		resp.AccountInfo = &AccountInfo{
			Holder:   info.AccountHolder,
			Number:   info.AccountNumber,
			SortCode: info.SortCode,
			Period:   info.StatementPeriod,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ConvertResponse{
		Success: false,
		Error:   msg,
	})
}
