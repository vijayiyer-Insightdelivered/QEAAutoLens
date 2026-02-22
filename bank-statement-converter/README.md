# Bank Statement PDF to CSV Converter

A standalone Go application with a React web UI that converts UK bank statement PDFs into structured CSV files. Built by **Insight Delivered** as part of the QEA AutoLens product suite.

## Supported Banks

| Bank | Date Format | Statement Layout |
|------|------------|-----------------|
| **Metro Bank** | DD/MM/YYYY | Date, Description, Paid out, Paid in, Balance |
| **HSBC** | DD Mon YY / DD Mon YYYY | Date, Payment type and details, Paid out, Paid in, Balance |
| **Barclays** | DD/MM/YYYY / DD Mon YYYY | Date, Description, Money out, Money in, Balance |

## Quick Start — Web UI

### 1. Build the Go backend

```bash
cd bank-statement-converter
go build -o bank-statement-converter .
```

### 2. Build the React frontend

```bash
cd web
npm install
npm run build
```

### 3. Run the server

```bash
# Serve API + React UI together
./bank-statement-converter --serve --static=./web/dist

# Open http://localhost:8080 in your browser
```

### Development mode (hot-reload)

Run the Go backend and Vite dev server separately:

```bash
# Terminal 1: Go API server
./bank-statement-converter --serve --port=8080

# Terminal 2: React dev server (auto-proxies /api to :8080)
cd web && npm run dev
# Open http://localhost:5173
```

## Web UI Features

- **Drag-and-drop** PDF upload
- **Bank auto-detection** or manual selection (Metro Bank, HSBC, Barclays)
- **Summary dashboard** — transaction count, total debits/credits, net
- **Account details** — holder, number, sort code, statement period
- **Transactions table** — scrollable, color-coded debits and credits
- **CSV download** — one-click download of the converted file

## CLI Usage

The tool also works as a standalone CLI:

```bash
# Auto-detect bank type and convert
./bank-statement-converter statement.pdf

# Specify bank explicitly
./bank-statement-converter --bank=hsbc statement.pdf

# Custom output path
./bank-statement-converter --bank=metro --output=transactions.csv statement.pdf

# Convert multiple files
./bank-statement-converter --bank=barclays jan.pdf feb.pdf mar.pdf

# Suppress account metadata in CSV header
./bank-statement-converter --header=false statement.pdf
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--bank` | (auto-detect) | Bank type: `metro`, `hsbc`, `barclays` |
| `--output` | `<input>.csv` | Output CSV file path |
| `--header` | `true` | Include account metadata rows in CSV |
| `--serve` | `false` | Start web UI server instead of CLI mode |
| `--port` | `8080` | Port for web UI server |
| `--static` | | Path to React build directory (`web/dist`) |
| `--version` | | Print version and exit |
| `--help` | | Show usage help |

## CSV Output Format

```
Date,Description,Type,Amount,Balance
15/01/2024,CARD PAYMENT TESCO STORES,DEBIT,25.99,1234.56
16/01/2024,DIRECT DEBIT SKY UK LTD,DEBIT,45.00,1189.56
17/01/2024,BANK CREDIT SALARY,CREDIT,2500.00,3689.56
```

When `--header=true` (default), metadata rows are prepended:

```
# Bank,metro
# Account Holder,John Smith
# Account Number,12345678
# Sort Code,23-05-80
# Statement Period,01/01/2024 to 31/01/2024
Date,Description,Type,Amount,Balance
...
```

## Project Structure

```
bank-statement-converter/
├── main.go                          # CLI + server entry point
├── go.mod / go.sum                  # Go module
├── internal/
│   ├── api/
│   │   ├── handler.go               # HTTP API (POST /api/convert, GET /api/health)
│   │   └── handler_test.go          # API endpoint tests
│   ├── models/
│   │   └── transaction.go           # Data types (Transaction, StatementInfo)
│   ├── extractor/
│   │   └── pdf.go                   # PDF text extraction
│   ├── parser/
│   │   ├── parser.go                # Parser interface + auto-detection
│   │   ├── util.go                  # Shared parsing utilities
│   │   ├── metro.go                 # Metro Bank parser
│   │   ├── hsbc.go                  # HSBC parser
│   │   ├── barclays.go              # Barclays parser
│   │   └── *_test.go                # Parser tests
│   └── writer/
│       ├── csv.go                   # CSV output writer
│       └── csv_test.go              # Writer tests
└── web/                             # React frontend (Vite)
    ├── index.html
    ├── vite.config.js               # Vite config with API proxy
    ├── package.json
    └── src/
        ├── main.jsx                 # React entry point
        ├── App.jsx                  # Root component
        ├── App.css                  # All styles
        ├── index.css                # Global styles (brand tokens)
        └── components/
            ├── FileUpload.jsx       # Upload + bank selector form
            └── Results.jsx          # Summary + transactions table + download
```

## Architecture

1. **PDF Extraction** (`internal/extractor`): Uses `github.com/ledongthuc/pdf` to extract text row-by-row from PDF pages.

2. **Bank Detection** (`internal/parser`): Auto-detects the bank by scanning for identifying keywords.

3. **Statement Parsing** (`internal/parser`): Bank-specific regex parsers extract transactions, amounts, and metadata.

4. **CSV Output** (`internal/writer`): Writes structured transaction data to CSV.

5. **HTTP API** (`internal/api`): POST `/api/convert` accepts multipart PDF upload, returns JSON with transactions + CSV string.

6. **React UI** (`web/`): Single-page app with drag-and-drop upload, bank selection, results dashboard, and CSV download.

## Running Tests

```bash
# Go tests (33 tests)
go test ./... -v

# React dev server
cd web && npm run dev
```

## Limitations

- Works with **text-based PDFs** only. Scanned/image-based statements require OCR preprocessing.
- Parsing accuracy depends on the PDF's text extraction quality.
- Multi-line transaction descriptions are supported but depend on consistent formatting.

## License

Proprietary — Insight Delivered.
