import { Component } from 'react'

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
        <div style={{
          margin: '2rem auto',
          maxWidth: '600px',
          padding: '2rem',
          background: '#1a1a2e',
          borderRadius: '12px',
          border: '1px solid #E86E29',
          color: '#ccd6e0',
          textAlign: 'center',
        }}>
          <h2 style={{ color: '#E86E29', marginBottom: '1rem' }}>Something went wrong</h2>
          <p style={{ marginBottom: '1rem' }}>
            The application encountered an error while displaying results.
          </p>
          <pre style={{
            textAlign: 'left',
            padding: '1rem',
            background: '#0A1628',
            borderRadius: '8px',
            fontSize: '0.8rem',
            overflow: 'auto',
            maxHeight: '200px',
            marginBottom: '1rem',
          }}>
            {this.state.error?.toString()}
          </pre>
          <button
            onClick={() => {
              this.setState({ hasError: false, error: null })
              window.location.reload()
            }}
            style={{
              padding: '0.75rem 2rem',
              background: '#E86E29',
              color: 'white',
              border: 'none',
              borderRadius: '8px',
              cursor: 'pointer',
              fontSize: '1rem',
              fontWeight: 600,
            }}
          >
            Reload App
          </button>
        </div>
      )
    }

    return this.props.children
  }
}

export default ErrorBoundary
