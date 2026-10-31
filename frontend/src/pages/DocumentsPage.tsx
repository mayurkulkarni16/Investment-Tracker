import React, { useEffect, useMemo, useState } from 'react';
import { documentsApi, familyApi, DOCUMENT_CATEGORIES, MAX_DOCUMENT_BYTES } from '../api/planning';
import type { VaultDocument, AssetItem } from '../api/planning';
import { formatDate } from '../utils/format';
import { useToast } from '../components/Toast';
import { useConfirm } from '../components/ConfirmDialog';
import { useAuth } from '../context/AuthContext';
import { apiErrorMessage } from '../utils/errors';
import { Download, Trash2, Plus } from 'lucide-react';
import RowActions from '../components/RowActions';
import { PageSkeleton } from '../components/ui';

const ACCEPT = '.pdf,.png,.jpg,.jpeg,.webp,.txt,.csv,.doc,.docx,.xls,.xlsx';
const DAY_MS = 24 * 60 * 60 * 1000;

const categoryLabel = (c: string) => DOCUMENT_CATEGORIES.find(x => x.value === c)?.label || c;

function formatSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${bytes} B`;
}

function daysUntil(dateStr: string | undefined, now: number): number | null {
  if (!dateStr) return null;
  const t = new Date(dateStr).getTime();
  if (Number.isNaN(t)) return null;
  return Math.ceil((t - now) / DAY_MS);
}

const emptyForm = { title: '', category: 'other', link: '', expiry_date: '', notes: '' };

export default function DocumentsPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [docs, setDocs] = useState<VaultDocument[] | null>(null);
  const [items, setItems] = useState<AssetItem[]>([]);
  const [reloadTick, setReloadTick] = useState(0);
  const [loadError, setLoadError] = useState('');
  const [category, setCategory] = useState('all');
  const [search, setSearch] = useState('');
  const [showUpload, setShowUpload] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [form, setForm] = useState(emptyForm);
  const [uploading, setUploading] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [now] = useState(() => Date.now());

  useEffect(() => {
    let cancelled = false;
    documentsApi.list()
      .then(r => { if (!cancelled) { setDocs(r.data || []); setLoadError(''); } })
      .catch(err => { if (!cancelled) { setDocs([]); setLoadError(apiErrorMessage(err, 'Failed to load documents')); } });
    return () => { cancelled = true; };
  }, [viewAsUserId, reloadTick]);

  useEffect(() => {
    let cancelled = false;
    familyApi.summary()
      .then(r => { if (!cancelled) setItems(r.data.items || []); })
      .catch(() => { if (!cancelled) setItems([]); });
    return () => { cancelled = true; };
  }, [viewAsUserId]);

  const itemByKey = useMemo(() => new Map(items.map(i => [`${i.asset_type}:${i.asset_id}`, i])), [items]);
  const groupedItems = useMemo(() => {
    const groups = new Map<string, AssetItem[]>();
    for (const it of items) {
      const list = groups.get(it.category) || [];
      list.push(it);
      groups.set(it.category, list);
    }
    return [...groups.entries()].sort((a, b) => a[0].localeCompare(b[0]));
  }, [items]);

  const reload = () => setReloadTick(t => t + 1);

  const closeUpload = () => {
    setShowUpload(false);
    setFile(null);
    setForm(emptyForm);
  };

  const onFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] || null;
    if (f && f.size > MAX_DOCUMENT_BYTES) {
      toast('File is larger than 10 MB', 'error');
      e.target.value = '';
      setFile(null);
      return;
    }
    setFile(f);
    if (f && !form.title) {
      setForm(prev => ({ ...prev, title: f.name.replace(/\.[^.]+$/, '') }));
    }
  };

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file) { toast('Choose a file to upload', 'error'); return; }
    if (file.size > MAX_DOCUMENT_BYTES) { toast('File is larger than 10 MB', 'error'); return; }
    const [asset_type, asset_id] = form.link ? form.link.split(':') : ['', ''];
    setUploading(true);
    try {
      await documentsApi.upload({
        file,
        title: form.title.trim() || file.name,
        category: form.category,
        asset_type,
        asset_id,
        expiry_date: form.expiry_date || undefined,
        notes: form.notes.trim() || undefined,
      });
      toast('Document uploaded');
      closeUpload();
      reload();
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to upload document'), 'error');
    } finally {
      setUploading(false);
    }
  };

  const handleDownload = async (doc: VaultDocument) => {
    setBusyId(doc.id);
    try {
      const r = await documentsApi.download(doc.id);
      const blob = r.data instanceof Blob ? r.data : new Blob([r.data], { type: doc.content_type });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = doc.file_name || doc.title;
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to download document'), 'error');
    } finally {
      setBusyId(null);
    }
  };

  const handleDelete = async (doc: VaultDocument) => {
    if (!await confirm({ title: 'Delete document', message: `Delete "${doc.title}"? This cannot be undone.`, danger: true, confirmLabel: 'Delete' })) return;
    setBusyId(doc.id);
    try {
      await documentsApi.remove(doc.id);
      toast('Document deleted');
      reload();
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to delete document'), 'error');
    } finally {
      setBusyId(null);
    }
  };

  if (docs === null) return <PageSkeleton label="Loading documents" />;

  const q = search.trim().toLowerCase();
  const filtered = docs.filter(d => {
    if (category !== 'all' && d.category !== category) return false;
    if (!q) return true;
    const linked = itemByKey.get(`${d.asset_type}:${d.asset_id}`)?.name || '';
    return [d.title, d.file_name, d.notes, linked].some(s => (s || '').toLowerCase().includes(q));
  });
  const expiringSoon = docs.filter(d => { const n = daysUntil(d.expiry_date, now); return n !== null && n >= 0 && n <= 30; }).length;
  const expired = docs.filter(d => { const n = daysUntil(d.expiry_date, now); return n !== null && n < 0; }).length;
  const totalSize = docs.reduce((s, d) => s + (d.size || 0), 0);

  return (
    <div>
      <div className="page-header">
        <h1>Documents</h1>
        {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowUpload(true)}><Plus size={16} aria-hidden="true" />Upload</button>}
      </div>

      {loadError && <div className="card text-danger mb-4">{loadError}</div>}

      {docs.length > 0 && (
        <div className="card-grid stat-grid mb-5">
          <div className="stat-card"><div className="label">Documents</div><div className="value">{docs.length}</div></div>
          <div className="stat-card"><div className="label">Storage used</div><div className="value">{formatSize(totalSize)}</div></div>
          <div className="stat-card"><div className="label">Expiring in 30 days</div><div className="value" style={{ color: expiringSoon ? 'var(--warning-text)' : undefined }}>{expiringSoon}</div></div>
          <div className="stat-card"><div className="label">Expired</div><div className={`value ${expired ? 'negative' : ''}`}>{expired}</div></div>
        </div>
      )}

      {docs.length === 0 ? (
        <div className="empty-state">
          <h3>No documents yet</h3>
          <p>Keep agreements, policies, receipts, statements and ID proofs in one place, linked to the investments they belong to.</p>
          {!isViewOnly && <button className="btn btn-primary" onClick={() => setShowUpload(true)}><Plus size={16} aria-hidden="true" />Upload document</button>}
        </div>
      ) : (
        <>
          <div className="search-bar">
            <input type="search" placeholder="Search title, file name, notes or linked item..." value={search} onChange={e => setSearch(e.target.value)} />
            <div className="flex gap-1 flex-wrap">
              <button className={`chip${category === 'all' ? ' active' : ''}`} aria-pressed={category === 'all'} onClick={() => setCategory('all')}>All</button>
              {DOCUMENT_CATEGORIES.map(c => (
                <button key={c.value} className={`chip${category === c.value ? ' active' : ''}`} aria-pressed={category === c.value} onClick={() => setCategory(c.value)}>{c.label}</button>
              ))}
            </div>
          </div>
          <div className="table-container">
            <table>
              <thead>
                <tr>
                  <th>Title</th>
                  <th>Category</th>
                  <th>Linked to</th>
                  <th>Size</th>
                  <th>Expiry</th>
                  <th>Uploaded</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.length === 0 && (
                  <tr><td colSpan={7} className="text-center text-muted">No documents match the filters.</td></tr>
                )}
                {filtered.map(d => {
                  const days = daysUntil(d.expiry_date, now);
                  const linked = d.asset_type ? itemByKey.get(`${d.asset_type}:${d.asset_id}`) : undefined;
                  const rowStyle = days !== null && days < 0
                    ? { background: 'rgba(234,67,53,0.06)' }
                    : days !== null && days <= 30 ? { background: 'rgba(249,171,0,0.08)' } : undefined;
                  return (
                    <tr key={d.id} style={rowStyle}>
                      <td>
                        <div className="fw-medium">{d.title}</div>
                        <div className="text-muted text-sm">{d.file_name}</div>
                        {d.notes && <div className="text-muted text-sm italic">{d.notes}</div>}
                      </td>
                      <td><span className="badge badge-completed">{categoryLabel(d.category)}</span></td>
                      <td>
                        {linked
                          ? <>{linked.name}<div className="text-muted text-sm">{linked.category}</div></>
                          : d.asset_type ? <span className="text-muted">{d.asset_type.replace(/_/g, ' ')}</span> : <span className="text-muted">-</span>}
                      </td>
                      <td>{formatSize(d.size)}</td>
                      <td>
                        {d.expiry_date ? (
                          <>
                            {formatDate(d.expiry_date)}
                            {days !== null && days < 0 && <div><span className="badge badge-overdue">Expired</span></div>}
                            {days !== null && days >= 0 && days <= 30 && (
                              <div><span className="badge badge-pending">{days === 0 ? 'Expires today' : `${days} day${days === 1 ? '' : 's'} left`}</span></div>
                            )}
                          </>
                        ) : '-'}
                      </td>
                      <td>{formatDate(d.created_at)}</td>
                      <td className="cell-actions">
                        <RowActions
                          inline={1}
                          label={`Actions for ${d.title}`}
                          actions={[
                            { label: 'Download', icon: Download, onClick: () => handleDownload(d), disabled: busyId === d.id },
                            { label: 'Delete', icon: Trash2, onClick: () => handleDelete(d), danger: true, hidden: isViewOnly, disabled: busyId === d.id },
                          ]}
                        />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </>
      )}

      {showUpload && !isViewOnly && (
        <div className="modal-overlay" onClick={closeUpload}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h2>Upload document</h2>
            <form onSubmit={handleUpload}>
              <div className="form-group">
                <label>File (max 10 MB)</label>
                <input type="file" accept={ACCEPT} onChange={onFileChange} required />
                {file && <div className="text-muted text-sm mt-1">{file.name} · {formatSize(file.size)}</div>}
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Title</label>
                  <input value={form.title} onChange={e => setForm({ ...form, title: e.target.value })} placeholder="e.g. HDFC FD receipt" required />
                </div>
                <div className="form-group">
                  <label>Category</label>
                  <select value={form.category} onChange={e => setForm({ ...form, category: e.target.value })}>
                    {DOCUMENT_CATEGORIES.map(c => <option key={c.value} value={c.value}>{c.label}</option>)}
                  </select>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>Linked to (optional)</label>
                  <select value={form.link} onChange={e => setForm({ ...form, link: e.target.value })}>
                    <option value="">— Not linked —</option>
                    {groupedItems.map(([cat, list]) => (
                      <optgroup key={cat} label={cat}>
                        {list.map(it => <option key={`${it.asset_type}:${it.asset_id}`} value={`${it.asset_type}:${it.asset_id}`}>{it.name}</option>)}
                      </optgroup>
                    ))}
                  </select>
                </div>
                <div className="form-group">
                  <label>Expiry date (optional)</label>
                  <input type="date" value={form.expiry_date} onChange={e => setForm({ ...form, expiry_date: e.target.value })} />
                </div>
              </div>
              <div className="form-group">
                <label>Notes</label>
                <textarea rows={2} value={form.notes} onChange={e => setForm({ ...form, notes: e.target.value })} />
              </div>
              <div className="modal-actions">
                <button type="button" className="btn btn-outline" onClick={closeUpload}>Cancel</button>
                <button type="submit" className="btn btn-primary" disabled={uploading || !file}>{uploading ? 'Uploading…' : 'Upload'}</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
