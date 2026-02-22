import { useState, useRef } from 'react'

const BANKS = [
  { value: '', label: 'Auto-detect', hint: 'Detect from PDF' },
  { value: 'metro', label: 'Metro Bank', hint: 'DD/MM/YYYY' },
  { value: 'hsbc', label: 'HSBC', hint: 'DD Mon YY' },
  { value: 'barclays', label: 'Barclays', hint: 'DD/MM/YYYY' },
]

function FileUpload({ onConvert, loading, error }) {
  const [file, setFile] = useState(null)
  const [bank, setBank] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const inputRef = useRef()

  const handleDrop = (e) => {
    e.preventDefault()
    setDragOver(false)
    const dropped = e.dataTransfer.files[0]
    if (dropped && dropped.name.toLowerCase().endsWith('.pdf')) {
      setFile(dropped)
    }
  }

  const handleDragOver = (e) => {
    e.preventDefault()
    setDragOver(true)
  }

  const handleDragLeave = () => setDragOver(false)

  const handleFileChange = (e) => {
    if (e.target.files[0]) {
      setFile(e.target.files[0])
    }
  }

  const handleSubmit = (e) => {
    e.preventDefault()
    if (file && !loading) {
      onConvert(file, bank)
    }
  }

  const formatSize = (bytes) => {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB'
    return (bytes / 1048576).toFixed(1) + ' MB'
  }

  return (
    <form className="upload-card" onSubmit={handleSubmit}>
      <h2>Upload Bank Statement PDF</h2>

      <div
        className={`drop-zone ${dragOver ? 'drag-over' : ''}`}
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onClick={() => inputRef.current?.click()}
      >
        <div className="drop-zone-icon">
          <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
            <rect x="8" y="6" width="32" height="36" rx="4" stroke="#8899aa" strokeWidth="2" fill="none" />
            <path d="M16 20l8-8 8 8" stroke="#E86E29" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
            <line x1="24" y1="12" x2="24" y2="32" stroke="#E86E29" strokeWidth="2" strokeLinecap="round" />
            <text x="24" y="41" textAnchor="middle" fontSize="7" fill="#8899aa" fontWeight="600">PDF</text>
          </svg>
        </div>
        <p>Drag and drop your PDF here, or <span className="browse-link">browse</span></p>
        <input
          ref={inputRef}
          type="file"
          accept=".pdf"
          onChange={handleFileChange}
          style={{ display: 'none' }}
        />
      </div>

      {file && (
        <div className="file-selected">
          <div>
            <span className="file-name">{file.name}</span>
            <span className="file-size"> ({formatSize(file.size)})</span>
          </div>
          <button
            type="button"
            className="file-remove"
            onClick={() => setFile(null)}
            title="Remove file"
          >
            &times;
          </button>
        </div>
      )}

      <div className="bank-selector">
        <label>Select Bank (or leave on auto-detect)</label>
        <div className="bank-options">
          {BANKS.map((b) => (
            <button
              key={b.value}
              type="button"
              className={`bank-option ${bank === b.value ? 'selected' : ''}`}
              onClick={() => setBank(b.value)}
            >
              <div className="bank-label">{b.label}</div>
              <div className="bank-hint">{b.hint}</div>
            </button>
          ))}
        </div>
      </div>

      <button
        type="submit"
        className="convert-btn"
        disabled={!file || loading}
      >
        {loading ? (
          <>
            <span className="spinner" />
            Converting...
          </>
        ) : (
          'Convert to CSV'
        )}
      </button>

      {error && <div className="error-banner">{error}</div>}
    </form>
  )
}

export default FileUpload
