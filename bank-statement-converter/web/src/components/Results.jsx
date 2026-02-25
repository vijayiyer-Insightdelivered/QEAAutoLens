import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Typography from '@mui/material/Typography'
import Button from '@mui/material/Button'
import Stack from '@mui/material/Stack'
import Chip from '@mui/material/Chip'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Accordion from '@mui/material/Accordion'
import AccordionSummary from '@mui/material/AccordionSummary'
import AccordionDetails from '@mui/material/AccordionDetails'
import DownloadIcon from '@mui/icons-material/Download'
import ReplayIcon from '@mui/icons-material/Replay'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import { BRAND } from '../theme'

function Results({ data, onReset }) {
  const { bank, accountInfo, csv } = data
  const transactions = data.transactions || []
  const totalDebit = data.totalDebit || 0
  const totalCredit = data.totalCredit || 0
  const count = data.count || transactions.length

  const bankNames = { metro: 'Metro Bank', hsbc: 'HSBC', barclays: 'Barclays' }

  const handleDownload = () => {
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `statement_${bank}_${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  const fmt = (n) =>
    n.toLocaleString('en-GB', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

  const summaryCards = [
    { label: 'Transactions', value: count, color: 'primary.main' },
    { label: 'Total Debits', value: `£${fmt(totalDebit)}`, color: 'error.main' },
    { label: 'Total Credits', value: `£${fmt(totalCredit)}`, color: 'success.main' },
    {
      label: 'Net',
      value: `£${fmt(Math.abs(totalCredit - totalDebit))} ${totalCredit - totalDebit >= 0 ? 'in' : 'out'}`,
      color: totalCredit - totalDebit >= 0 ? 'success.main' : 'error.main',
    },
  ]

  return (
    <Stack spacing={2}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 1 }}>
        <Typography variant="h6" sx={{ fontWeight: 600 }}>
          Conversion Complete — {bankNames[bank] || bank}
        </Typography>
        <Stack direction="row" spacing={1}>
          <Button
            variant="contained"
            color="secondary"
            startIcon={<DownloadIcon />}
            onClick={handleDownload}
          >
            Download CSV
          </Button>
          <Button
            variant="outlined"
            startIcon={<ReplayIcon />}
            onClick={onReset}
          >
            Convert Another
          </Button>
        </Stack>
      </Box>

      {/* Summary Cards */}
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: 2 }}>
        {summaryCards.map((card) => (
          <Paper key={card.label} sx={{ p: 2, textAlign: 'center' }}>
            <Typography variant="overline" color="text.secondary">
              {card.label}
            </Typography>
            <Typography variant="h5" sx={{ fontWeight: 700, color: card.color }}>
              {card.value}
            </Typography>
          </Paper>
        ))}
      </Box>

      {/* Account Info */}
      {accountInfo && (
        <Paper sx={{ p: 2 }}>
          <Typography variant="overline" color="text.secondary" sx={{ mb: 1, display: 'block' }}>
            Account Details
          </Typography>
          <Stack direction="row" spacing={3} sx={{ flexWrap: 'wrap' }}>
            {accountInfo.holder && (
              <Box>
                <Typography variant="caption" color="text.secondary">Account Holder</Typography>
                <Typography variant="body2" sx={{ fontWeight: 600 }}>{accountInfo.holder}</Typography>
              </Box>
            )}
            {accountInfo.number && (
              <Box>
                <Typography variant="caption" color="text.secondary">Account Number</Typography>
                <Typography variant="body2" sx={{ fontWeight: 600 }}>{accountInfo.number}</Typography>
              </Box>
            )}
            {accountInfo.sortCode && (
              <Box>
                <Typography variant="caption" color="text.secondary">Sort Code</Typography>
                <Typography variant="body2" sx={{ fontWeight: 600 }}>{accountInfo.sortCode}</Typography>
              </Box>
            )}
            {accountInfo.period && (
              <Box>
                <Typography variant="caption" color="text.secondary">Statement Period</Typography>
                <Typography variant="body2" sx={{ fontWeight: 600 }}>{accountInfo.period}</Typography>
              </Box>
            )}
          </Stack>
        </Paper>
      )}

      {/* Transactions Table */}
      <Paper>
        <Typography variant="subtitle1" sx={{ p: 2, pb: 1, fontWeight: 600 }}>
          Transactions ({count})
        </Typography>
        {transactions.length === 0 ? (
          <Box sx={{ p: 3, textAlign: 'center' }}>
            <Typography color="text.secondary" sx={{ mb: 0.5 }}>
              No transactions could be extracted from this statement.
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              The PDF format may not match the expected layout. Try selecting the bank manually or check that the PDF is not image-based (scanned).
            </Typography>
            {data.rawText && (
              <Accordion>
                <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                  <Typography variant="body2" sx={{ color: 'secondary.main', fontWeight: 600 }}>
                    Show extracted PDF text (for debugging)
                  </Typography>
                </AccordionSummary>
                <AccordionDetails>
                  <Box
                    component="pre"
                    sx={{
                      p: 2,
                      bgcolor: BRAND.dark,
                      color: '#ccd6e0',
                      borderRadius: 2,
                      fontSize: '0.75rem',
                      lineHeight: 1.4,
                      maxHeight: 400,
                      overflow: 'auto',
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-all',
                    }}
                  >
                    {data.rawText}
                  </Box>
                </AccordionDetails>
              </Accordion>
            )}
          </Box>
        ) : (
          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow sx={{ bgcolor: 'background.default' }}>
                  <TableCell sx={{ fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5, color: 'text.secondary' }}>Date</TableCell>
                  <TableCell sx={{ fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5, color: 'text.secondary' }}>Description</TableCell>
                  <TableCell sx={{ fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5, color: 'text.secondary' }}>Type</TableCell>
                  <TableCell align="right" sx={{ fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5, color: 'text.secondary' }}>Amount</TableCell>
                  <TableCell align="right" sx={{ fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: 0.5, color: 'text.secondary' }}>Balance</TableCell>
                  <TableCell sx={{ fontWeight: 600, fontSize: '0.7rem', color: 'text.secondary' }}>Method</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {transactions.map((txn, i) => (
                  <TableRow key={i} hover>
                    <TableCell>{txn.date}</TableCell>
                    <TableCell>{txn.description}</TableCell>
                    <TableCell>
                      <Chip
                        label={txn.type}
                        size="small"
                        sx={{
                          fontWeight: 700,
                          fontSize: '0.7rem',
                          bgcolor: txn.type === 'DEBIT' ? 'error.light' : 'success.light',
                          color: txn.type === 'DEBIT' ? 'error.main' : 'success.main',
                        }}
                      />
                    </TableCell>
                    <TableCell
                      align="right"
                      sx={{
                        fontWeight: 600,
                        fontVariantNumeric: 'tabular-nums',
                        color: txn.type === 'DEBIT' ? 'error.main' : 'success.main',
                      }}
                    >
                      {txn.type === 'DEBIT' ? '-' : '+'}£{txn.amount ? fmt(txn.amount) : '0.00'}
                    </TableCell>
                    <TableCell
                      align="right"
                      sx={{ fontVariantNumeric: 'tabular-nums', color: 'text.secondary' }}
                    >
                      {txn.balance ? `£${fmt(txn.balance)}` : ''}
                    </TableCell>
                    <TableCell sx={{ fontSize: '0.7rem', color: 'text.secondary', fontFamily: 'monospace' }}>
                      {txn.parseMethod || ''}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Paper>

      {/* Version indicator */}
      <Paper sx={{ p: 1.5, bgcolor: '#1a2a3a', display: 'flex', gap: 2, flexWrap: 'wrap' }}>
        <Typography variant="caption" sx={{ color: BRAND.midGrey }}>
          Backend:{' '}
          <Box component="strong" sx={{ color: data.version ? '#4ade80' : '#ef4444' }}>
            {data.version || 'OLD (no version)'}
          </Box>
        </Typography>
        <Typography variant="caption" sx={{ color: BRAND.midGrey }}>
          Transactions: {count}
        </Typography>
        {transactions.length > 0 && transactions[0].parseMethod && (
          <Typography variant="caption" sx={{ color: BRAND.midGrey }}>
            Parse method:{' '}
            <Box component="strong" sx={{ color: '#60a5fa' }}>
              {transactions[0].parseMethod}
            </Box>
          </Typography>
        )}
      </Paper>

      {/* Debug: Parser line-by-line analysis */}
      {data.debugLines && data.debugLines.length > 0 && (
        <Accordion defaultExpanded>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="body2" sx={{ color: 'secondary.main', fontWeight: 600 }}>
              Debug: Parser line-by-line ({data.debugLines.length} lines processed)
            </Typography>
          </AccordionSummary>
          <AccordionDetails sx={{ p: 0 }}>
            <TableContainer sx={{ maxHeight: 400, bgcolor: BRAND.dark }}>
              <Table size="small" stickyHeader>
                <TableHead>
                  <TableRow>
                    {['#', 'Result', 'Date?', 'Tabs', 'Method', 'Line text'].map((h) => (
                      <TableCell
                        key={h}
                        sx={{
                          bgcolor: '#1a2a3a',
                          color: BRAND.midGrey,
                          fontSize: '0.7rem',
                          fontFamily: 'monospace',
                          py: 0.5,
                        }}
                      >
                        {h}
                      </TableCell>
                    ))}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {data.debugLines.map((dl, i) => {
                    const colors = {
                      parsed: '#4ade80',
                      header: '#60a5fa',
                      continuation: '#fbbf24',
                      unmatched: '#ef4444',
                      'skipped-pre-section': '#6b7280',
                    }
                    const c = colors[dl.result] || '#ccd6e0'
                    return (
                      <TableRow key={i} sx={{ '& td': { color: c, fontSize: '0.7rem', fontFamily: 'monospace', py: 0.3, borderColor: '#111a28' } }}>
                        <TableCell>{dl.lineNum}</TableCell>
                        <TableCell sx={{ fontWeight: 600 }}>{dl.result}</TableCell>
                        <TableCell align="center">{dl.hasDate ? 'Y' : ''}</TableCell>
                        <TableCell align="center">{dl.tabParts || ''}</TableCell>
                        <TableCell>{dl.method || ''}</TableCell>
                        <TableCell sx={{ maxWidth: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {dl.text.replace(/\t/g, ' → ')}
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </TableContainer>
          </AccordionDetails>
        </Accordion>
      )}

      {/* Debug: Raw extracted text */}
      {(data.rawText || data.frontendText) && (
        <Accordion>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="body2" sx={{ color: 'text.secondary', fontWeight: 600 }}>
              Debug: Show raw extracted text
            </Typography>
          </AccordionSummary>
          <AccordionDetails>
            {data.rawText && (
              <Box
                component="pre"
                sx={{
                  p: 2,
                  bgcolor: BRAND.dark,
                  color: '#ccd6e0',
                  borderRadius: 2,
                  fontSize: '0.72rem',
                  lineHeight: 1.4,
                  maxHeight: 300,
                  overflow: 'auto',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-all',
                }}
              >
                {data.rawText.replace(/\t/g, ' → ')}
              </Box>
            )}
          </AccordionDetails>
        </Accordion>
      )}
    </Stack>
  )
}

export default Results
