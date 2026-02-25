import { useState } from 'react'
import * as pdfjsLib from 'pdfjs-dist'
import Box from '@mui/material/Box'
import AppBar from '@mui/material/AppBar'
import Toolbar from '@mui/material/Toolbar'
import Typography from '@mui/material/Typography'
import Container from '@mui/material/Container'
import FileUpload from './components/FileUpload'
import Results from './components/Results'
import { BRAND } from './theme'

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

    const items = textContent.items.filter((item) => item.str.trim() !== '')
    if (items.length === 0) continue

    // Calculate average font height for adaptive column gap detection
    let totalHeight = 0
    let heightCount = 0
    for (const item of items) {
      const h = item.height || Math.abs(item.transform[3])
      if (h > 0) {
        totalHeight += h
        heightCount++
      }
    }
    const avgFontHeight = heightCount > 0 ? totalHeight / heightCount : 10
    // Column gap threshold: gaps wider than ~3 character widths are column separators
    const colGapThreshold = avgFontHeight * 2.5

    // Group text items by Y position with tolerance (items within 3px are same row)
    const groups = []
    for (const item of items) {
      const y = item.transform[5]
      let found = false
      for (const group of groups) {
        if (Math.abs(group.y - y) < 3) {
          group.items.push({
            x: item.transform[4],
            text: item.str,
            width: item.width || 0,
          })
          // Update group Y to average for better clustering
          group.y = (group.y * (group.items.length - 1) + y) / group.items.length
          found = true
          break
        }
      }
      if (!found) {
        groups.push({
          y,
          items: [{ x: item.transform[4], text: item.str, width: item.width || 0 }],
        })
      }
    }

    // Sort rows top-to-bottom (higher Y = higher on page in PDF coords)
    groups.sort((a, b) => b.y - a.y)

    const lines = []
    for (const group of groups) {
      group.items.sort((a, b) => a.x - b.x)
      let line = ''
      let prevEnd = 0
      for (const item of group.items) {
        const gap = item.x - prevEnd
        if (line && gap > colGapThreshold) {
          line += '\t' // tab = column separator
        } else if (line && gap > 1) {
          line += ' '
        }
        line += item.text
        prevEnd = item.x + item.width
      }
      if (line.trim()) lines.push(line)
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
        // Attach the frontend's own extracted text for debugging
        data.frontendText = extractedText
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
    <Box sx={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <AppBar position="static" sx={{ bgcolor: 'primary.main' }}>
        <Toolbar sx={{ flexDirection: 'column', py: 2 }}>
          {/* Logo */}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.5 }}>
            <Box
              sx={{
                width: 36,
                height: 36,
                borderRadius: '50%',
                bgcolor: 'primary.dark',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <svg width="36" height="36" viewBox="0 0 120 120" fill="none">
                <circle cx="60" cy="60" r="50" stroke={BRAND.orange} strokeWidth="6" fill="none" />
                <circle cx="60" cy="60" r="20" stroke="#ffffff" strokeWidth="4" fill="none" />
                <line x1="60" y1="10" x2="60" y2="35" stroke="#ffffff" strokeWidth="3" />
                <line x1="60" y1="85" x2="60" y2="110" stroke="#ffffff" strokeWidth="3" />
                <line x1="10" y1="60" x2="35" y2="60" stroke="#ffffff" strokeWidth="3" />
                <line x1="85" y1="60" x2="110" y2="60" stroke="#ffffff" strokeWidth="3" />
              </svg>
            </Box>
            <Typography variant="h6" sx={{ fontWeight: 700, letterSpacing: 0.5 }}>
              <span>QEA</span>
              <span style={{ color: BRAND.midGrey, margin: '0 2px' }}>/</span>
              <span>Auto</span>
              <span style={{ color: BRAND.orange }}>Lens</span>
            </Typography>
          </Box>
          <Typography variant="subtitle1" sx={{ fontWeight: 400, opacity: 0.9 }}>
            Bank Statement Converter
          </Typography>
          <Typography variant="caption" sx={{ color: 'text.secondary' }}>
            Convert PDF bank statements to CSV — Metro Bank, HSBC, Barclays
          </Typography>
        </Toolbar>
      </AppBar>

      {/* Main Content */}
      <Container maxWidth="md" sx={{ flex: 1, py: 4 }}>
        {!result ? (
          <FileUpload onConvert={handleConvert} loading={loading} error={error} />
        ) : (
          <Results data={result} onReset={handleReset} />
        )}
      </Container>

      {/* Footer */}
      <Box
        component="footer"
        sx={{
          bgcolor: 'primary.main',
          color: 'text.secondary',
          textAlign: 'center',
          py: 1.5,
        }}
      >
        <Typography variant="caption">
          Insight Delivered — See Every Deal. Know Every Margin.
        </Typography>
      </Box>
    </Box>
  )
}

export default App
