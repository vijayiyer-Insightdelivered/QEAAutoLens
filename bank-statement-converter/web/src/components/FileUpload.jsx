import { useState, useRef } from 'react'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Typography from '@mui/material/Typography'
import Button from '@mui/material/Button'
import Alert from '@mui/material/Alert'
import CircularProgress from '@mui/material/CircularProgress'
import Stack from '@mui/material/Stack'
import UploadFileIcon from '@mui/icons-material/UploadFile'
import InsertDriveFileIcon from '@mui/icons-material/InsertDriveFile'
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
    <Paper component="form" onSubmit={handleSubmit} sx={{ p: 3 }}>
      <Typography variant="h6" sx={{ mb: 2, fontWeight: 600 }}>
        Upload Bank Statement PDF
      </Typography>

      {/* Drop Zone */}
      <Box
        onClick={() => inputRef.current?.click()}
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        sx={{
          border: '2px dashed',
          borderColor: dragOver ? 'secondary.main' : 'divider',
          borderRadius: 2,
          p: 5,
          textAlign: 'center',
          cursor: 'pointer',
          bgcolor: dragOver ? 'rgba(232, 110, 41, 0.06)' : 'background.default',
          transition: 'all 0.2s ease',
          '&:hover': {
            borderColor: 'secondary.main',
            bgcolor: 'rgba(232, 110, 41, 0.06)',
          },
        }}
      >
        <UploadFileIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 1 }} />
        <Typography color="text.secondary" variant="body2">
          Drag and drop your PDF here, or{' '}
          <Box component="span" sx={{ color: 'secondary.main', fontWeight: 600, textDecoration: 'underline' }}>
            browse
          </Box>
        </Typography>
        <input
          ref={inputRef}
          type="file"
          accept=".pdf"
          onChange={handleFileChange}
          style={{ display: 'none' }}
        />
      </Box>

      {/* Selected File */}
      {file && (
        <Alert
          icon={<InsertDriveFileIcon />}
          severity="success"
          sx={{ mt: 2 }}
          onClose={() => setFile(null)}
        >
          <Typography variant="body2" component="span" sx={{ fontWeight: 600 }}>
            {file.name}
          </Typography>
          <Typography variant="caption" color="text.secondary" sx={{ ml: 1 }}>
            ({formatSize(file.size)})
          </Typography>
        </Alert>
      )}

      {/* Bank Selector */}
      <Box sx={{ mt: 3 }}>
        <Typography variant="body2" sx={{ fontWeight: 600, mb: 1 }}>
          Select Bank (or leave on auto-detect)
        </Typography>
        <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap', gap: 1 }}>
          {BANKS.map((b) => (
            <Paper
              key={b.value}
              onClick={() => setBank(b.value)}
              sx={{
                flex: 1,
                minWidth: 120,
                p: 1.5,
                textAlign: 'center',
                cursor: 'pointer',
                border: '2px solid',
                borderColor: bank === b.value ? 'secondary.main' : 'divider',
                bgcolor: bank === b.value ? 'rgba(232, 110, 41, 0.06)' : 'background.paper',
                transition: 'all 0.2s ease',
                '&:hover': {
                  borderColor: 'secondary.light',
                },
              }}
            >
              <Typography variant="body2" sx={{ fontWeight: 600 }}>
                {b.label}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {b.hint}
              </Typography>
            </Paper>
          ))}
        </Stack>
      </Box>

      {/* Convert Button */}
      <Button
        type="submit"
        variant="contained"
        color="secondary"
        fullWidth
        size="large"
        disabled={!file || loading}
        sx={{ mt: 3, py: 1.2 }}
      >
        {loading ? (
          <>
            <CircularProgress size={20} sx={{ color: 'white', mr: 1 }} />
            Converting...
          </>
        ) : (
          'Convert to CSV'
        )}
      </Button>

      {/* Error */}
      {error && (
        <Alert severity="error" sx={{ mt: 2 }}>
          {error}
        </Alert>
      )}
    </Paper>
  )
}

export default FileUpload
