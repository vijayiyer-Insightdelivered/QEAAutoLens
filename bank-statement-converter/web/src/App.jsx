import { useState } from 'react'
import FileUpload from './components/FileUpload'
import Results from './components/Results'
import './App.css'

function App() {
  const [result, setResult] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const handleConvert = async (file, bank) => {
    setLoading(true)
    setError(null)
    setResult(null)

    const formData = new FormData()
    formData.append('file', file)
    if (bank) {
      formData.append('bank', bank)
    }

    try {
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
