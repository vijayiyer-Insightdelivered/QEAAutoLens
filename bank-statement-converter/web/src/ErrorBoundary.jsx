import { Component } from 'react'
import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Typography from '@mui/material/Typography'
import Button from '@mui/material/Button'

class ErrorBoundary extends Component {
  constructor(props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
          <Paper
            sx={{
              maxWidth: 600,
              p: 4,
              textAlign: 'center',
              bgcolor: '#1a1a2e',
              border: '1px solid',
              borderColor: 'secondary.main',
              color: '#ccd6e0',
            }}
          >
            <Typography variant="h5" sx={{ color: 'secondary.main', mb: 2 }}>
              Something went wrong
            </Typography>
            <Typography sx={{ mb: 2 }}>
              The application encountered an error while displaying results.
            </Typography>
            <Box
              component="pre"
              sx={{
                textAlign: 'left',
                p: 2,
                bgcolor: '#0A1628',
                borderRadius: 2,
                fontSize: '0.8rem',
                overflow: 'auto',
                maxHeight: 200,
                mb: 2,
              }}
            >
              {this.state.error?.toString()}
            </Box>
            <Button
              variant="contained"
              color="secondary"
              size="large"
              onClick={() => {
                this.setState({ hasError: false, error: null })
                window.location.reload()
              }}
            >
              Reload App
            </Button>
          </Paper>
        </Box>
      )
    }

    return this.props.children
  }
}

export default ErrorBoundary
