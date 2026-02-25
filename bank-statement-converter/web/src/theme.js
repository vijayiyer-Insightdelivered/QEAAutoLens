import { createTheme } from '@mui/material/styles'

const BRAND = {
  navy: '#003366',
  orange: '#E86E29',
  orangeLight: '#F28C4E',
  white: '#FFFFFF',
  lightGrey: '#F4F6F8',
  midGrey: '#8899AA',
  dark: '#0A1628',
}

const theme = createTheme({
  palette: {
    primary: {
      main: BRAND.navy,
      dark: BRAND.dark,
      contrastText: BRAND.white,
    },
    secondary: {
      main: BRAND.orange,
      light: BRAND.orangeLight,
      contrastText: BRAND.white,
    },
    background: {
      default: BRAND.lightGrey,
      paper: BRAND.white,
    },
    text: {
      primary: BRAND.navy,
      secondary: BRAND.midGrey,
    },
    success: {
      main: '#2e7d32',
      light: '#e8f5e9',
    },
    error: {
      main: '#c62828',
      light: '#fdecea',
    },
  },
  typography: {
    fontFamily: "'Segoe UI', 'SF Pro Display', system-ui, sans-serif",
  },
  shape: {
    borderRadius: 10,
  },
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          fontWeight: 600,
        },
      },
    },
    MuiPaper: {
      defaultProps: {
        elevation: 0,
      },
      styleOverrides: {
        root: {
          boxShadow: '0 1px 6px rgba(0, 0, 0, 0.06)',
        },
      },
    },
  },
})

export { BRAND }
export default theme
