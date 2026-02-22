import { useState } from 'react'
import * as pdfjsLib from 'pdfjs-dist'
import FileUpload from './components/FileUpload'
import Results from './components/Results'
import './App.css'

// Configure pdf.js worker
pdfjsLib.GlobalWorkerOptions.workerSrc = new URL(
  'pdfjs-dist/build/pdf.worker.mjs',
  import.meta.url,
).toString()

// Extract text from PDF using pdf.js (Mozilla's PDF library)
async function extractTextFromPDF(file) {
  const arrayBuffer = await file.arrayBuffer()
  const pdf = await pdfjsLib.getDocument({ data: arrayBuffer }).promise
  const pages = []

  for (let i = 1; i <= pdf.numPages; i++) {
    const page = await pdf.getPage(i)
    const textContent = await page.getTextContent()

    // Group text items by Y position to reconstruct lines
    const items = textContent.items.filter((item) => item.str.trim() !== '')
    if (items.length === 0) continue

    // Sort by Y (descending) then X (ascending) — PDF Y goes bottom-to-top
    const rows = new Map()
    for (const item of items) {
      const y = Math.round(item.transform[5]) // Y position
      if (!rows.has(y)) rows.set(y, [])
      rows.get(y).push({ x: item.transform[4], text: item.str, width: item.width })
    }

    // Sort rows top-to-bottom
    const sortedYs = [...rows.keys()].sort((a, b) => b - a)
    const lines = []
    for (const y of sortedYs) {
      const items = rows.get(y).sort((a, b) => a.x - b.x)
      // Join items with appropriate spacing
      let line = ''
      let prevEnd = 0
      for (const item of items) {
        const gap = item.x - prevEnd
        if (line && gap > 10) {
          line += '  ' // column separator
        } else if (line && gap > 2) {
          line += ' '
        }
        line += item.text
        prevEnd = item.x + (item.width || 0)
      }
      if (line.trim()) lines.push(line.trim())
    }

    pages.push(lines.join('\n'))
  }

  return pages
}

function App() {
  const [result, setResult] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const handleConvert = async (file, bank) => {
    setLoading(true)
    setError(null)
    setResult(null)

    try {
      // Step 1: Extract text from PDF client-side using pdf.js
      let extractedText = ''
      try {
        const pages = await extractTextFromPDF(file)
        if (pages.length > 0) {
          extractedText = pages.join('\n---PAGE_BREAK---\n')
        }
      } catch (pdfErr) {
        console.warn('Client-side PDF extraction failed, falling back to server:', pdfErr)
      }

      // Step 2: Send to backend for parsing
      const formData = new FormData()
      formData.append('file', file)
      if (bank) formData.append('bank', bank)
      if (extractedText) formData.append('extractedText', extractedText)

      const res = await fetch('/api/convert', {
        method: 'POST',
        body: formData,
      })
      const data = await res.json()
      if (!data.success) {
        setError(data.error || 'Conversion failed.')
      } else {
        setResult(data)
      }
    } catch {
      setError('Failed to connect to server. Make sure the backend is running.')
    } finally {
      setLoading(false)
    }
  }

  const handleReset = () => {
    setResult(null)
    setError(null)
  }

  return (
    <div className="app">
      <header className="app-header">
        <div className="logo">
          <div className="logo-icon">
            <svg width="36" height="36" viewBox="0 0 120 120" fill="none">
              <circle cx="60" cy="60" r="50" stroke="#E86E29" strokeWidth="6" fill="none" />
              <circle cx="60" cy="60" r="20" stroke="#ffffff" strokeWidth="4" fill="none" />
              <line x1="60" y1="10" x2="60" y2="35" stroke="#ffffff" strokeWidth="3" />
              <line x1="60" y1="85" x2="60" y2="110" stroke="#ffffff" strokeWidth="3" />
              <line x1="10" y1="60" x2="35" y2="60" stroke="#ffffff" strokeWidth="3" />
              <line x1="85" y1="60" x2="110" y2="60" stroke="#ffffff" strokeWidth="3" />
            </svg>
          </div>
          <div className="logo-text">
            <span className="logo-qea">QEA</span>
            <span className="logo-sep">/</span>
            <span className="logo-auto">Auto</span>
            <span className="logo-lens">Lens</span>
          </div>
        </div>
        <h1>Bank Statement Converter</h1>
        <p className="subtitle">
          Convert PDF bank statements to CSV — Metro Bank, HSBC, Barclays
        </p>
      </header>

      <main className="app-main">
        {!result ? (
          <FileUpload
            onConvert={handleConvert}
            loading={loading}
            error={error}
          />
        ) : (
          <Results data={result} onReset={handleReset} />
        )}
      </main>

      <footer className="app-footer">
        <p>Insight Delivered — See Every Deal. Know Every Margin.</p>
      </footer>
    </div>
  )
}

export default App
