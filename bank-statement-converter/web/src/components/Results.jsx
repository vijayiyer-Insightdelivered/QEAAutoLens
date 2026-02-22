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

  return (
    <div className="results">
      {/* Header */}
      <div className="results-header">
        <h2>Conversion Complete — {bankNames[bank] || bank}</h2>
        <div className="results-actions">
          <button className="btn-download" onClick={handleDownload}>
            Download CSV
          </button>
          <button className="btn-reset" onClick={onReset}>
            Convert Another
          </button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="summary-cards">
        <div className="summary-card">
          <div className="card-label">Transactions</div>
          <div className="card-value">{count}</div>
        </div>
        <div className="summary-card">
          <div className="card-label">Total Debits</div>
          <div className="card-value debit">&pound;{fmt(totalDebit)}</div>
        </div>
        <div className="summary-card">
          <div className="card-label">Total Credits</div>
          <div className="card-value credit">&pound;{fmt(totalCredit)}</div>
        </div>
        <div className="summary-card">
          <div className="card-label">Net</div>
          <div className={`card-value ${totalCredit - totalDebit >= 0 ? 'credit' : 'debit'}`}>
            &pound;{fmt(Math.abs(totalCredit - totalDebit))}
            {totalCredit - totalDebit >= 0 ? ' in' : ' out'}
          </div>
        </div>
      </div>

      {/* Account Info */}
      {accountInfo && (
        <div className="account-info">
          <h3>Account Details</h3>
          <div className="account-fields">
            {accountInfo.holder && (
              <div className="account-field">
                <div className="field-label">Account Holder</div>
                <div className="field-value">{accountInfo.holder}</div>
              </div>
            )}
            {accountInfo.number && (
              <div className="account-field">
                <div className="field-label">Account Number</div>
                <div className="field-value">{accountInfo.number}</div>
              </div>
            )}
            {accountInfo.sortCode && (
              <div className="account-field">
                <div className="field-label">Sort Code</div>
                <div className="field-value">{accountInfo.sortCode}</div>
              </div>
            )}
            {accountInfo.period && (
              <div className="account-field">
                <div className="field-label">Statement Period</div>
                <div className="field-value">{accountInfo.period}</div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Transactions Table */}
      <div className="transactions-card">
        <h3>Transactions ({count})</h3>
        {transactions.length === 0 ? (
          <div style={{ padding: '2rem', color: '#8899AA' }}>
            <p style={{ fontSize: '1.1rem', marginBottom: '0.5rem', textAlign: 'center' }}>
              No transactions could be extracted from this statement.
            </p>
            <p style={{ fontSize: '0.9rem', textAlign: 'center', marginBottom: '1rem' }}>
              The PDF format may not match the expected layout. Try selecting the bank manually or check that the PDF is not image-based (scanned).
            </p>
            {data.rawText && (
              <details style={{ marginTop: '1rem' }}>
                <summary style={{ cursor: 'pointer', color: '#E86E29', fontWeight: 600 }}>
                  Show extracted PDF text (for debugging)
                </summary>
                <pre style={{
                  marginTop: '0.5rem',
                  padding: '1rem',
                  background: '#0A1628',
                  color: '#ccd6e0',
                  borderRadius: '8px',
                  fontSize: '0.75rem',
                  lineHeight: '1.4',
                  maxHeight: '400px',
                  overflow: 'auto',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-all',
                }}>{data.rawText}</pre>
              </details>
            )}
          </div>
        ) : (
          <div className="table-wrapper">
            <table className="txn-table">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Description</th>
                  <th>Type</th>
                  <th style={{ textAlign: 'right' }}>Amount</th>
                  <th style={{ textAlign: 'right' }}>Balance</th>
                </tr>
              </thead>
              <tbody>
                {transactions.map((txn, i) => (
                  <tr key={i}>
                    <td>{txn.date}</td>
                    <td>{txn.description}</td>
                    <td>
                      <span className={`type-badge ${txn.type === 'DEBIT' ? 'debit' : 'credit'}`}>
                        {txn.type}
                      </span>
                    </td>
                    <td className={`amount ${txn.type === 'DEBIT' ? 'debit' : 'credit'}`}>
                      {txn.type === 'DEBIT' ? '-' : '+'}
                      &pound;{txn.amount ? fmt(txn.amount) : '0.00'}
                    </td>
                    <td className="balance">
                      {txn.balance ? `\u00A3${fmt(txn.balance)}` : ''}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Debug: Raw extracted text (always available) */}
      {data.rawText && (
        <details style={{ marginTop: '1.5rem' }}>
          <summary style={{ cursor: 'pointer', color: '#8899AA', fontSize: '0.85rem', fontWeight: 600 }}>
            Debug: Show raw extracted text
          </summary>
          <pre style={{
            marginTop: '0.5rem',
            padding: '1rem',
            background: '#0A1628',
            color: '#ccd6e0',
            borderRadius: '8px',
            fontSize: '0.72rem',
            lineHeight: '1.4',
            maxHeight: '400px',
            overflow: 'auto',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-all',
          }}>{data.rawText.replace(/\t/g, ' → ')}</pre>
        </details>
      )}
    </div>
  )
}

export default Results
