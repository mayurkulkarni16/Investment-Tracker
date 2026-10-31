import React, { useState } from 'react';
import { brokerImportApi } from '../api/planning';
import type { BrokerImportResult, BrokerName } from '../api/planning';
import { formatCurrency, formatDate } from '../utils/format';
import { useToast } from './Toast';
import { apiErrorMessage } from '../utils/errors';

interface Props {
  open: boolean;
  onClose: () => void;
  onImported: () => void;
}

const ACTION_STYLES: Record<string, { label: string; bg: string; color: string }> = {
  create: { label: 'New stock', bg: 'var(--success-soft)', color: 'var(--success-text)' },
  add_trade: { label: 'Add trade', bg: 'var(--primary-soft)', color: '#4f46e5' },
  skip_duplicate: { label: 'Duplicate', bg: 'var(--surface-muted)', color: 'var(--text-secondary)' },
  skip_exists: { label: 'Already exists', bg: 'var(--surface-muted)', color: 'var(--text-secondary)' },
  conflict: { label: 'Conflict', bg: 'var(--warning-soft)', color: 'var(--warning-text)' },
  error: { label: 'Error', bg: 'var(--danger-soft)', color: 'var(--danger-text)' },
};

function todayISO(): string {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

function ActionBadge({ action }: { action: string }) {
  const s = ACTION_STYLES[action] || { label: action, bg: 'var(--surface-muted)', color: 'var(--text-secondary)' };
  return <span className="badge nowrap" style={{ background: s.bg, color: s.color }}>{s.label}</span>;
}

export default function BrokerImportModal({ open, onClose, onImported }: Props) {
  const { toast } = useToast();
  const [file, setFile] = useState<File | null>(null);
  const [broker, setBroker] = useState<BrokerName>('auto');
  const [asOf, setAsOf] = useState(todayISO);
  const [preview, setPreview] = useState<BrokerImportResult | null>(null);
  const [busy, setBusy] = useState<'preview' | 'import' | null>(null);
  const [showHelp, setShowHelp] = useState(false);

  if (!open) return null;

  const reset = () => {
    setFile(null);
    setBroker('auto');
    setAsOf(todayISO());
    setPreview(null);
    setBusy(null);
  };

  const close = () => {
    if (busy) return;
    reset();
    onClose();
  };

  const onFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFile(e.target.files?.[0] || null);
    setPreview(null);
  };

  const run = async (dryRun: boolean) => {
    if (!file) { toast('Choose a CSV file first', 'error'); return; }
    setBusy(dryRun ? 'preview' : 'import');
    try {
      const r = await brokerImportApi.importCSV(file, broker, dryRun, asOf || undefined);
      if (dryRun) {
        setPreview(r.data);
      } else {
        const res = r.data;
        toast(`Imported: ${res.stocks_created} new stock${res.stocks_created === 1 ? '' : 's'}, ${res.trades_added} trade${res.trades_added === 1 ? '' : 's'} added, ${res.skipped} skipped`);
        reset();
        onImported();
        onClose();
        return;
      }
    } catch (err) {
      toast(apiErrorMessage(err, dryRun ? 'Failed to read the CSV' : 'Import failed'), 'error');
    }
    setBusy(null);
  };

  const rows = preview?.items || [];
  const errors = preview?.errors || [];
  const actionable = rows.filter(r => r.action === 'create' || r.action === 'add_trade').length;

  return (
    <div className="modal-overlay" onClick={close}>
      <div className="modal" style={{ maxWidth: 860 }} onClick={e => e.stopPropagation()}>
        <h2>Import from broker</h2>

        <div className="form-group">
          <label>CSV file</label>
          <input type="file" accept=".csv,text/csv" onChange={onFileChange} />
        </div>
        <div className="form-row">
          <div className="form-group">
            <label>Broker</label>
            <select value={broker} onChange={e => { setBroker(e.target.value as BrokerName); setPreview(null); }}>
              <option value="auto">Auto-detect</option>
              <option value="zerodha">Zerodha (Console)</option>
              <option value="groww">Groww</option>
            </select>
          </div>
          <div className="form-group">
            <label>Holdings date</label>
            <input type="date" value={asOf} max={todayISO()} onChange={e => { setAsOf(e.target.value); setPreview(null); }} />
            <div className="text-muted text-xs" style={{ marginTop: 2 }}>
              Used as the buy date for holdings files (tradebooks use each trade's own date).
            </div>
          </div>
        </div>

        <button type="button" className="btn btn-sm btn-outline mb-3" onClick={() => setShowHelp(s => !s)}>
          {showHelp ? '▼' : '▶'} Where do I get this file?
        </button>
        {showHelp && (
          <div className="card text-md mb-3" style={{ background: 'var(--bg-secondary, #f8f9fa)' }}>
            <p style={{ margin: '0 0 6px' }}>
              <strong>Zerodha</strong> — log in to <em>console.zerodha.com</em>:
            </p>
            <ul style={{ margin: '0 0 10px 18px' }}>
              <li>Holdings: <em>Portfolio → Holdings → Download (CSV)</em></li>
              <li>Tradebook: <em>Reports → Tradebook</em>, segment Equity, pick a date range → <em>Download CSV</em></li>
            </ul>
            <p style={{ margin: '0 0 6px' }}>
              <strong>Groww</strong> — on the web app:
            </p>
            <ul style={{ margin: '0 0 10px 18px' }}>
              <li>Holdings: <em>Stocks → Holdings → Download holdings statement</em></li>
              <li>Order history: <em>Profile → Reports → Stocks order history</em></li>
              <li>Groww downloads are XLSX — open in Excel/Sheets and <em>Save as / Download → CSV</em> first.</li>
            </ul>
            <p className="text-muted m-0">
              Holdings files add one position per stock at its average cost; tradebook/order files add each buy and sell.
              Already-imported trades are skipped, so re-importing is safe.
            </p>
          </div>
        )}

        {preview && (
          <div className="mt-2">
            <div className="flex gap-3 flex-wrap text-md mb-2">
              <span>Detected: <strong className="capitalize">{preview.broker}</strong> {preview.kind}</span>
              <span>Rows: <strong>{preview.rows}</strong></span>
              <span className="text-success">New stocks: <strong>{preview.stocks_created}</strong></span>
              <span style={{ color: '#4f46e5' }}>Trades to add: <strong>{preview.trades_added}</strong></span>
              <span className="text-muted">Skipped: <strong>{preview.skipped}</strong></span>
            </div>
            {errors.length > 0 && (
              <div className="card text-danger text-md mb-2 p-3">
                {errors.map((e, i) => <div key={i}>• {e}</div>)}
              </div>
            )}
            {rows.length > 0 ? (
              <div className="table-container" style={{ maxHeight: 320, overflowY: 'auto' }}>
                <table>
                  <thead>
                    <tr>
                      <th>Action</th>
                      <th>Symbol</th>
                      <th>Type</th>
                      <th>Date</th>
                      <th className="text-right">Qty</th>
                      <th className="text-right">Price</th>
                      <th>Note</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((r, i) => (
                      <tr key={`${r.symbol}-${i}`}>
                        <td><ActionBadge action={r.action} /></td>
                        <td>
                          <strong>{r.symbol}</strong>
                          {r.name && r.name !== r.symbol && <div className="text-muted text-xs">{r.name}</div>}
                        </td>
                        <td className="capitalize">{r.type || '-'}</td>
                        <td>{r.date ? formatDate(r.date) : '-'}</td>
                        <td className="text-right">{r.quantity}</td>
                        <td className="text-right">{r.price ? formatCurrency(r.price) : '-'}</td>
                        <td className="text-muted text-sm">{r.message || ''}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="text-muted">No rows found in this file.</p>
            )}
          </div>
        )}

        <div className="modal-actions">
          <button type="button" className="btn btn-outline" onClick={close} disabled={busy !== null}>Cancel</button>
          <button type="button" className="btn btn-outline" onClick={() => run(true)} disabled={!file || busy !== null}>
            {busy === 'preview' ? 'Reading…' : 'Preview'}
          </button>
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => run(false)}
            disabled={!file || busy !== null || (preview !== null && actionable === 0)}
            title={preview ? undefined : 'Tip: Preview first to see what will change'}
          >
            {busy === 'import' ? 'Importing…' : preview ? `Import ${actionable} change${actionable === 1 ? '' : 's'}` : 'Import'}
          </button>
        </div>
      </div>
    </div>
  );
}
