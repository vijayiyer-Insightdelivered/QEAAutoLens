package api

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/insightdelivered/bank-statement-converter/internal/extractor"
	"github.com/insightdelivered/bank-statement-converter/internal/models"
	"github.com/insightdelivered/bank-statement-converter/internal/parser"
	"github.com/insightdelivered/bank-statement-converter/internal/writer"
)

// ConvertResponse is the JSON response from the /api/convert endpoint.
type ConvertResponse struct {
	Success      bool                 `json:"success"`
	Error        string               `json:"error,omitempty"`
	Bank         string               `json:"bank,omitempty"`
	AccountInfo  *AccountInfo         `json:"accountInfo,omitempty"`
	Transactions []models.Transaction `json:"transactions"`
	CSV          string               `json:"csv,omitempty"`
	TotalDebit   float64              `json:"totalDebit"`
	TotalCredit  float64              `json:"totalCredit"`
	Count        int                  `json:"count"`
	RawText      string               `json:"rawText,omitempty"`
	Version      string               `json:"version,omitempty"`
	DebugLines   []models.DebugLine   `json:"debugLines,omitempty"`
}

// AccountInfo holds account metadata for the JSON response.
type AccountInfo struct {
	Holder         string  `json:"holder,omitempty"`
	Number         string  `json:"number,omitempty"`
	SortCode       string  `json:"sortCode,omitempty"`
	Period         string  `json:"period,omitempty"`
	OpeningBalance float64 `json:"openingBalance,omitempty"`
}

const apiVersion = "2.0.0"

// HandleHealth returns a simple health check.
func HandleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"version": apiVersion,
		"engine":  "fiber",
	})
}

// HandleConvert processes a PDF upload and returns parsed transactions.
func HandleConvert(c *fiber.Ctx) error {
	// Get the uploaded file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "No file uploaded. Use form field 'file'.")
	}

	// Validate it's a PDF
	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".pdf") {
		return writeError(c, fiber.StatusBadRequest, "Only PDF files are supported.")
	}

	// Get optional parameters
	bankParam := c.FormValue("bank")
	includeHeader := c.FormValue("header") != "false"

	// Check if pre-extracted text was provided (from client-side pdf.js extraction)
	extractedText := c.FormValue("extractedText")
	var pages []string

	if extractedText != "" {
		// Normalize CRLF to LF — browsers convert \n to \r\n when encoding
		// FormData values (per the HTML spec), but the page separator uses \n.
		extractedText = strings.ReplaceAll(extractedText, "\r\n", "\n")
		// Use the client-side extracted text (split by page separator)
		var candidatePages []string
		for _, page := range strings.Split(extractedText, "\n---PAGE_BREAK---\n") {
			page = strings.TrimSpace(page)
			if page != "" {
				candidatePages = append(candidatePages, page)
			}
		}
		// Only use client-side text if it's readable (not garbage)
		if extractor.IsReadableText(candidatePages) {
			pages = candidatePages
		}
	}

	// If no pre-extracted text, try server-side extraction
	if len(pages) == 0 {
		tmpFile, err := os.CreateTemp("", "statement-*.pdf")
		if err != nil {
			return writeError(c, fiber.StatusInternalServerError, "Failed to create temp file.")
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		// Save the uploaded file to a temp location
		if err := c.SaveFile(fileHeader, tmpFile.Name()); err != nil {
			return writeError(c, fiber.StatusInternalServerError, "Failed to save uploaded file.")
		}

		var extractErr error
		pages, extractErr = extractor.ExtractText(tmpFile.Name())
		if extractErr != nil {
			return writeError(c, fiber.StatusUnprocessableEntity, fmt.Sprintf("PDF extraction failed: %v", extractErr))
		}
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
			return writeError(c, fiber.StatusBadRequest, fmt.Sprintf("Unknown bank: %q. Use metro, hsbc, or barclays.", bankParam))
		}
	} else {
		detected, err := parser.AutoDetect(pages)
		if err != nil {
			return writeError(c, fiber.StatusUnprocessableEntity, err.Error())
		}
		bankType = detected
	}

	// Parse
	p, err := parser.New(bankType)
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, err.Error())
	}

	info, err := p.Parse(pages)
	if err != nil {
		return writeError(c, fiber.StatusUnprocessableEntity, fmt.Sprintf("Parsing failed: %v", err))
	}

	// Generate CSV string
	var csvBuf bytes.Buffer
	csvWriter := &writer.CSVWriter{IncludeHeader: includeHeader}
	if err := csvWriter.Write(&csvBuf, info); err != nil {
		return writeError(c, fiber.StatusInternalServerError, fmt.Sprintf("CSV generation failed: %v", err))
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

	// Ensure transactions is never nil (nil marshals to JSON null, not [])
	txns := info.Transactions
	if txns == nil {
		txns = []models.Transaction{}
	}

	resp := ConvertResponse{
		Success:      true,
		Bank:         string(bankType),
		Transactions: txns,
		CSV:          csvBuf.String(),
		TotalDebit:   totalDebit,
		TotalCredit:  totalCredit,
		Count:        len(txns),
		Version:      apiVersion,
	}

	if info.AccountHolder != "" || info.AccountNumber != "" || info.SortCode != "" || info.StatementPeriod != "" || info.OpeningBalance != 0 {
		resp.AccountInfo = &AccountInfo{
			Holder:         info.AccountHolder,
			Number:         info.AccountNumber,
			SortCode:       info.SortCode,
			Period:         info.StatementPeriod,
			OpeningBalance: info.OpeningBalance,
		}
	}

	// Always include raw extracted text (helps debug parser issues)
	resp.RawText = strings.Join(pages, "\n--- PAGE BREAK ---\n")

	// Include debug lines for diagnosing parse issues
	resp.DebugLines = info.DebugLines

	return c.JSON(resp)
}

func writeError(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(ConvertResponse{
		Success: false,
		Error:   msg,
	})
}
