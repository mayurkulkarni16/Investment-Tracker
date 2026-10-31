import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { familyApi } from '../api/planning';
import type { FamilySummary, MemberSummary, AssetItem } from '../api/planning';
import { getProfiles } from '../api/profiles';
import type { Profile } from '../types';
import { formatCurrency } from '../utils/format';
import { useToast } from '../components/Toast';
import { useAuth } from '../context/AuthContext';
import { apiErrorMessage } from '../utils/errors';
import { PageSkeleton } from '../components/ui';
import { SERIES } from '../components/chartUtils';

const UNASSIGNED_COLOR = '#9aa0a6';

function itemKey(it: AssetItem) {
  return `${it.asset_type}:${it.asset_id}`;
}

function topEntries(map: Record<string, number> | null | undefined, n = 5) {
  return Object.entries(map || {})
    .filter(([, v]) => v !== 0)
    .sort((a, b) => b[1] - a[1])
    .slice(0, n);
}

function MemberCard({ m }: { m: MemberSummary }) {
  const unassigned = m.profile_id === '';
  const color = unassigned ? UNASSIGNED_COLOR : (m.color || SERIES.primary);
  const assets = topEntries(m.assets);
  const liabilities = topEntries(m.liabilities, 3);
  return (
    <div className="stat-card" style={{ borderTop: `4px solid ${color}`, opacity: unassigned && m.item_count === 0 ? 0.6 : 1 }}>
      <div className="flex items-center gap-2 mb-2">
        <span style={{ width: 12, height: 12, borderRadius: '50%', background: color, display: 'inline-block' }} />
        <strong className="text-lg">{unassigned ? 'Unassigned' : m.name}</strong>
        {!unassigned && m.relationship && (
          <span className="badge text-muted capitalize" style={{ background: 'var(--bg-secondary, #f1f3f4)' }}>
            {m.relationship}
          </span>
        )}
        <span className="text-muted ml-auto text-sm">{m.item_count} item{m.item_count === 1 ? '' : 's'}</span>
      </div>
      <div className="label">Net Worth</div>
      <div className={`value ${m.net_worth >= 0 ? '' : 'negative'}`}>{formatCurrency(m.net_worth)}</div>
      <div className="flex gap-4 text-md mt-2">
        <span>Assets: <span className="text-success">{formatCurrency(m.total_assets)}</span></span>
        <span>Liabilities: <span className="text-danger">{formatCurrency(m.total_liabilities)}</span></span>
      </div>
      {(assets.length > 0 || liabilities.length > 0) && (
        <div className="text-sm" style={{ marginTop: 10 }}>
          {assets.map(([cat, v]) => {
            const pct = m.total_assets > 0 ? (v / m.total_assets) * 100 : 0;
            return (
              <div className="mb-1" key={`a-${cat}`}>
                <div className="flex justify-between">
                  <span className="text-muted">{cat}</span>
                  <span>{formatCurrency(v)}</span>
                </div>
                <div style={{ height: 4, background: 'var(--border, #e0e0e0)', borderRadius: 2 }}>
                  <div style={{ width: `${Math.min(100, pct)}%`, height: 4, background: color, borderRadius: 2 }} />
                </div>
              </div>
            );
          })}
          {liabilities.map(([cat, v]) => (
            <div className="flex justify-between" key={`l-${cat}`} style={{ marginBottom: 2 }}>
              <span className="text-muted">{cat}</span>
              <span className="text-danger">−{formatCurrency(v)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default function FamilyPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const { toast } = useToast();
  const [summary, setSummary] = useState<FamilySummary | null>(null);
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [error, setError] = useState('');
  const [reloadTick, setReloadTick] = useState(0);
  const [category, setCategory] = useState('all');
  const [search, setSearch] = useState('');
  const [onlyUnassigned, setOnlyUnassigned] = useState(false);
  const [saving, setSaving] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    Promise.all([familyApi.summary(), getProfiles().catch(() => ({ data: [] as Profile[] }))])
      .then(([s, p]) => {
        if (cancelled) return;
        setSummary(s.data);
        setProfiles(p.data || []);
        setError('');
      })
      .catch(err => { if (!cancelled) setError(apiErrorMessage(err, 'Failed to load family summary')); });
    return () => { cancelled = true; };
  }, [viewAsUserId, reloadTick]);

  const items = useMemo(() => summary?.items || [], [summary]);
  const categories = useMemo(() => [...new Set(items.map(i => i.category))].sort(), [items]);
  const members = useMemo(() => {
    const list = summary?.members || [];
    // Real members first, unassigned bucket last
    return [...list].sort((a, b) => (a.profile_id === '' ? 1 : 0) - (b.profile_id === '' ? 1 : 0));
  }, [summary]);

  const profileById = useMemo(() => new Map(profiles.map(p => [p.id, p])), [profiles]);

  const filtered = items.filter(it => {
    if (category !== 'all' && it.category !== category) return false;
    if (onlyUnassigned && it.profile_id) return false;
    if (search && !it.name.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  });

  const handleAssign = async (it: AssetItem, profileId: string) => {
    const key = itemKey(it);
    setSaving(key);
    try {
      await familyApi.assign({ asset_type: it.asset_type, asset_id: it.asset_id, profile_id: profileId });
      toast(profileId ? `Assigned to ${profileById.get(profileId)?.name || 'member'}` : 'Owner cleared');
      setReloadTick(t => t + 1);
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to assign owner'), 'error');
    } finally {
      setSaving(null);
    }
  };

  if (error && !summary) {
    return (
      <div className="empty-state">
        <h3>Couldn't load family view</h3>
        <p>{error}</p>
        <button className="btn btn-primary" onClick={() => setReloadTick(t => t + 1)}>Retry</button>
      </div>
    );
  }
  if (!summary) return <PageSkeleton label="Loading family" />;

  const totalNet = members.reduce((s, m) => s + m.net_worth, 0);
  const unassignedCount = items.filter(i => !i.profile_id).length;

  return (
    <div>
      <div className="page-header">
        <h1>Family</h1>
        <div className="text-muted text-base">
          Household net worth: <strong style={{ color: 'var(--text)' }}>{formatCurrency(totalNet)}</strong>
        </div>
      </div>

      {profiles.length === 0 && (
        <div className="card mb-5" style={{ borderLeft: '4px solid var(--warning)' }}>
          <strong>No family members yet.</strong>{' '}
          <span className="text-muted">
            Add profiles (you, spouse, parents, children…) in <Link to="/settings">Settings</Link>, then come back to assign
            each investment or loan to its owner.
          </span>
        </div>
      )}

      <div className="card-grid mb-6">
        {members.map(m => <MemberCard key={m.profile_id || 'unassigned'} m={m} />)}
      </div>

      <div className="page-header mb-3">
        <h2 className="text-xl">Assign owners</h2>
        {unassignedCount > 0 && <span className="badge badge-pending">{unassignedCount} unassigned</span>}
      </div>

      {items.length === 0 ? (
        <div className="empty-state">
          <h3>Nothing to assign yet</h3>
          <p>Add investments, accounts or loans and they will show up here.</p>
        </div>
      ) : (
        <>
          <div className="search-bar">
            <input type="search" placeholder="Search by name..." value={search} onChange={e => setSearch(e.target.value)} />
            <select value={category} onChange={e => setCategory(e.target.value)} style={{ padding: '8px 12px', borderRadius: 6 }}>
              <option value="all">All categories</option>
              {categories.map(c => <option key={c} value={c}>{c}</option>)}
            </select>
            <label className="text-md flex items-center gap-1">
              <input type="checkbox" checked={onlyUnassigned} onChange={e => setOnlyUnassigned(e.target.checked)} />
              Unassigned only
            </label>
          </div>
          <div className="table-container">
            <table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Category</th>
                  <th className="text-right">Value</th>
                  <th>Type</th>
                  <th>Owner</th>
                </tr>
              </thead>
              <tbody>
                {filtered.length === 0 && (
                  <tr><td colSpan={5} className="text-center text-muted">No items match the filters.</td></tr>
                )}
                {filtered.map(it => {
                  const key = itemKey(it);
                  const owner = it.profile_id ? profileById.get(it.profile_id) : undefined;
                  return (
                    <tr key={key}>
                      <td>{it.name}</td>
                      <td>{it.category}</td>
                      <td className={`text-right ${it.is_liability ? 'text-danger' : ''}`}>
                        {it.is_liability ? '−' : ''}{formatCurrency(it.value)}
                      </td>
                      <td>
                        {it.is_liability
                          ? <span className="badge badge-overdue">Liability</span>
                          : <span className="badge badge-active">Asset</span>}
                      </td>
                      <td>
                        {isViewOnly || profiles.length === 0 ? (
                          <span className="inline-flex items-center gap-2">
                            <span style={{ width: 10, height: 10, borderRadius: '50%', background: owner?.color || UNASSIGNED_COLOR, display: 'inline-block' }} />
                            {owner?.name || 'Unassigned'}
                          </span>
                        ) : (
                          <select
                            value={it.profile_id || ''}
                            disabled={saving === key}
                            onChange={e => handleAssign(it, e.target.value)}
                            style={{ padding: '4px 8px', borderRadius: 6, borderLeft: `4px solid ${owner?.color || UNASSIGNED_COLOR}` }}
                          >
                            <option value="">Unassigned</option>
                            {profiles.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
                          </select>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}
