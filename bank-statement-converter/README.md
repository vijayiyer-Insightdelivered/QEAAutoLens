# Bank Statement PDF to CSV Converter

A standalone Go CLI tool that converts UK bank statement PDFs into structured CSV files. Built by **Insight Delivered** as part of the QEA AutoLens product suite.

## Supported Banks

| Bank | Date Format | Statement Layout |
|------|------------|-----------------|
| **Metro Bank** | DD/MM/YYYY | Date, Description, Paid out, Paid in, Balance |
| **HSBC** | DD Mon YY / DD Mon YYYY | Date, Payment type and details, Paid out, Paid in, Balance |
| **Barclays** | DD/MM/YYYY / DD Mon YYYY | Date, Description, Money out, Money in, Balance |

## Installation

### From Source

Requires Go 1.21+.

```bash
cd bank-statement-converter
go build -o bank-statement-converter .
```

The binary will be created in the current directory.

## Usage

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

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--bank` | (auto-detect) | Bank type: `metro`, `hsbc`, `barclays` |
| `--output` | `<input>.csv` | Output CSV file path |
| `--header` | `true` | Include account metadata rows in CSV |
| `--version` | | Print version and exit |
| `--help` | | Show usage help |

## CSV Output Format

The converter produces standard CSV with these columns:

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
├── main.go                          # CLI entry point
├── go.mod                           # Go module definition
├── go.sum                           # Dependency checksums
├── internal/
│   ├── models/
│   │   └── transaction.go           # Data types (Transaction, StatementInfo)
│   ├── extractor/
│   │   └── pdf.go                   # PDF text extraction using ledongthuc/pdf
│   ├── parser/
│   │   ├── parser.go                # Parser interface + auto-detection
│   │   ├── util.go                  # Shared parsing utilities
│   │   ├── metro.go                 # Metro Bank parser
│   │   ├── hsbc.go                  # HSBC parser
│   │   ├── barclays.go              # Barclays parser
│   │   ├── parser_test.go           # Parser interface tests
│   │   ├── util_test.go             # Utility function tests
│   │   ├── metro_test.go            # Metro Bank parser tests
│   │   ├── hsbc_test.go             # HSBC parser tests
│   │   └── barclays_test.go         # Barclays parser tests
│   └── writer/
│       ├── csv.go                   # CSV output writer
│       └── csv_test.go              # CSV writer tests
└── README.md
```

## Architecture

1. **PDF Extraction** (`internal/extractor`): Uses `github.com/ledongthuc/pdf` to extract text row-by-row from PDF pages, preserving layout structure.

2. **Bank Detection** (`internal/parser`): Auto-detects the bank by scanning for identifying keywords (e.g., "Metro Bank", "HSBC", "Barclays").

3. **Statement Parsing** (`internal/parser`): Each bank has a dedicated parser that uses regex patterns matched to the bank's specific statement format to extract:
   - Transaction dates, descriptions, amounts, and balances
   - Account metadata (holder name, account number, sort code, statement period)

4. **CSV Output** (`internal/writer`): Writes structured transaction data to CSV with optional metadata headers.

## Running Tests

```bash
go test ./... -v
```

## Limitations

- Works with **text-based PDFs** only. Scanned/image-based statements require OCR preprocessing.
- Parsing accuracy depends on the PDF's text extraction quality. Some PDFs with complex layouts may need manual adjustment.
- Multi-line transaction descriptions are supported but depend on consistent formatting.

## License

Proprietary — Insight Delivered.
