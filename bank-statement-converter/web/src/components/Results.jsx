function Results({ data, onReset }) {
  const { bank, accountInfo, transactions, csv, totalDebit, totalCredit, count } = data

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
      </div>
    </div>
  )
}

export default Results
