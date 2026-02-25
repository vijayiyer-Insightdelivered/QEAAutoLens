package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/insightdelivered/bank-statement-converter/internal/api"
	"github.com/insightdelivered/bank-statement-converter/internal/extractor"
	"github.com/insightdelivered/bank-statement-converter/internal/models"
	"github.com/insightdelivered/bank-statement-converter/internal/parser"
	"github.com/insightdelivered/bank-statement-converter/internal/writer"
)

const version = "2.0.0"

func main() {
	// CLI flags
	bankFlag := flag.String("bank", "", "Bank type: metro, hsbc, barclays (auto-detected if omitted)")
	outputFlag := flag.String("output", "", "Output CSV file path (defaults to input filename with .csv extension)")
	headerFlag := flag.Bool("header", true, "Include account metadata header rows in CSV")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	helpFlag := flag.Bool("help", false, "Show usage help")
	serveFlag := flag.Bool("serve", false, "Start web UI server instead of CLI mode")
	portFlag := flag.String("port", "8080", "Port for web UI server (used with --serve)")
	staticFlag := flag.String("static", "", "Path to React build directory (used with --serve)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Bank Statement PDF to CSV Converter (Fiber v2)
by Insight Delivered (QEA AutoLens)

Converts bank statement PDFs from Metro Bank, HSBC, and Barclays
into structured CSV files for analysis.

Usage:
  bank-statement-converter [flags] <input.pdf> [input2.pdf ...]

  Web UI mode:
  bank-statement-converter --serve [--port=8080] [--static=./web/dist]

Flags:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # Auto-detect bank and convert
  bank-statement-converter statement.pdf

  # Specify bank explicitly
  bank-statement-converter --bank=hsbc statement.pdf

  # Custom output path
  bank-statement-converter --bank=metro --output=transactions.csv statement.pdf

  # Convert multiple files
  bank-statement-converter --bank=barclays jan.pdf feb.pdf mar.pdf

  # Start web UI (Go Fiber)
  bank-statement-converter --serve --port=3001

Supported Banks:
  metro     - Metro Bank (DD/MM/YYYY format)
  hsbc      - HSBC UK (DD Mon YY format)
  barclays  - Barclays (DD/MM/YYYY or DD Mon YYYY format)
`)
	}

	flag.Parse()

	if *versionFlag {
		fmt.Printf("bank-statement-converter v%s (Go Fiber)\n", version)
		os.Exit(0)
	}

	// Web server mode
	if *serveFlag {
		startServer(*portFlag, *staticFlag)
		return
	}

	if *helpFlag || flag.NArg() == 0 {
		flag.Usage()
		os.Exit(0)
	}

	inputFiles := flag.Args()

	// Validate bank flag if provided
	var bankType models.BankType
	if *bankFlag != "" {
		switch strings.ToLower(*bankFlag) {
		case "metro", "metrobank":
			bankType = models.BankMetro
		case "hsbc":
			bankType = models.BankHSBC
		case "barclays":
			bankType = models.BankBarclays
		default:
			fatalf("Unknown bank type %q. Supported: metro, hsbc, barclays\n", *bankFlag)
		}
	}

	// Process each input file
	for _, inputPath := range inputFiles {
		if err := processFile(inputPath, bankType, *outputFlag, *headerFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", inputPath, err)
			os.Exit(1)
		}
	}
}

func startServer(port, staticDir string) {
	app := fiber.New(fiber.Config{
		AppName:   "Bank Statement Converter v" + version,
		BodyLimit: 32 * 1024 * 1024, // 32MB max upload
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,OPTIONS",
		AllowHeaders: "Content-Type",
	}))

	// API routes
	apiGroup := app.Group("/api")
	apiGroup.Get("/health", api.HandleHealth)
	apiGroup.Post("/convert", api.HandleConvert)

	// Serve React static files (SPA)
	if staticDir != "" {
		app.Static("/", staticDir, fiber.Static{
			Index: "index.html",
		})
		// SPA fallback: serve index.html for any non-file, non-API route
		app.Get("/*", func(c *fiber.Ctx) error {
			path := c.Path()
			if strings.HasPrefix(path, "/api/") {
				return c.SendStatus(fiber.StatusNotFound)
			}
			fullPath := filepath.Join(staticDir, path)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				return c.SendFile(filepath.Join(staticDir, "index.html"))
			}
			return c.Next()
		})
	}

	addr := ":" + port
	fmt.Printf("Bank Statement Converter v%s — Go Fiber\n", version)
	fmt.Printf("Server starting on http://localhost%s\n", addr)
	if staticDir != "" {
		fmt.Printf("Serving UI from: %s\n", staticDir)
	} else {
		fmt.Printf("API-only mode (no --static dir specified)\n")
		fmt.Printf("Run React dev server separately: cd web && npm run dev\n")
	}

	log.Fatal(app.Listen(addr))
}

func processFile(inputPath string, bankType models.BankType, outputPath string, includeHeader bool) error {
	// Validate input file
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	ext := strings.ToLower(filepath.Ext(inputPath))
	if ext != ".pdf" {
		return fmt.Errorf("expected .pdf file, got %q", ext)
	}

	fmt.Printf("Processing: %s\n", inputPath)

	// Extract text from PDF
	pages, err := extractor.ExtractText(inputPath)
	if err != nil {
		return fmt.Errorf("PDF extraction failed: %w", err)
	}

	fmt.Printf("  Extracted text from %d page(s)\n", len(pages))

	// Auto-detect bank if not specified
	effectiveBank := bankType
	if effectiveBank == "" {
		detected, err := parser.AutoDetect(pages)
		if err != nil {
			return err
		}
		effectiveBank = detected
		fmt.Printf("  Auto-detected bank: %s\n", effectiveBank)
	}

	// Create parser for the bank
	p, err := parser.New(effectiveBank)
	if err != nil {
		return err
	}

	fmt.Printf("  Using %s parser\n", p.BankName())

	// Parse the statement
	info, err := p.Parse(pages)
	if err != nil {
		return fmt.Errorf("parsing failed: %w", err)
	}

	fmt.Printf("  Found %d transaction(s)\n", len(info.Transactions))

	if len(info.Transactions) == 0 {
		fmt.Println("  Warning: No transactions found. The PDF format may not match expected patterns.")
		fmt.Println("  Try specifying the bank explicitly with --bank flag if auto-detection was used.")
	}

	// Determine output path
	outPath := outputPath
	if outPath == "" {
		base := strings.TrimSuffix(inputPath, filepath.Ext(inputPath))
		outPath = base + ".csv"
	}

	// Write CSV
	w := &writer.CSVWriter{IncludeHeader: includeHeader}
	if err := w.WriteToFile(outPath, info); err != nil {
		return fmt.Errorf("CSV write failed: %w", err)
	}

	fmt.Printf("  Output: %s\n", outPath)

	// Print summary
	if info.AccountHolder != "" {
		fmt.Printf("  Account holder: %s\n", info.AccountHolder)
	}
	if info.AccountNumber != "" {
		fmt.Printf("  Account number: %s\n", info.AccountNumber)
	}
	if info.SortCode != "" {
		fmt.Printf("  Sort code: %s\n", info.SortCode)
	}
	if info.StatementPeriod != "" {
		fmt.Printf("  Period: %s\n", info.StatementPeriod)
	}

	fmt.Println("  Done.")
	return nil
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
