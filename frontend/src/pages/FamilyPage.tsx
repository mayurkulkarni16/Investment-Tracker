import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { familyApi } from '../api/planning';
import type { FamilySummary, MemberSummary, AssetItem, OwnershipShare } from '../api/planning';
import { getProfiles, createProfile, updateProfile, deleteProfile } from '../api/profiles';
import type { Profile, Occupation, CreateProfileRequest } from '../types';
import { OCCUPATION_LABELS } from '../types';
import { formatCurrency, toInputDate } from '../utils/format';
import { useToast } from '../components/Toast';
import { useConfirm } from '../components/ConfirmDialog';
import { useAuth } from '../context/AuthContext';
import { apiErrorMessage } from '../utils/errors';
import { PageSkeleton } from '../components/ui';
import { SERIES } from '../components/chartUtils';

const UNASSIGNED_COLOR = '#9aa0a6';
const RELATIONSHIPS = ['self', 'spouse', 'parent', 'child', 'sibling', 'other'];

function itemKey(it: AssetItem) {
  return `${it.asset_type}:${it.asset_id}`;
}

function topEntries(map: Record<string, number> | null | undefined, n = 5) {
  return Object.entries(map || {})
    .filter(([, v]) => v !== 0)
    .sort((a, b) => b[1] - a[1])
    .slice(0, n);
}

function MemberCard({ m, profile, canEdit, onEdit, onDelete }: {
  m: MemberSummary; profile?: Profile; canEdit: boolean; onEdit: () => void; onDelete: () => void;
}) {
  const unassigned = m.profile_id === '';
  const color = unassigned ? UNASSIGNED_COLOR : (m.color || SERIES.primary);
  const assets = topEntries(m.assets);
  const liabilities = topEntries(m.liabilities, 3);
  const occupation = profile?.occupation ? OCCUPATION_LABELS[profile.occupation as Occupation] : '';
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
      {!unassigned && (occupation || profile?.age) && (
        <div className="text-muted text-sm mb-2">
          {[occupation, profile?.age ? `${profile.age} yrs` : '', profile?.residential_status === 'nri' ? 'NRI' : ''].filter(Boolean).join(' · ')}
        </div>
      )}
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
      {!unassigned && (
        <div className="flex gap-2 mt-2 flex-wrap" style={{ marginTop: 12 }}>
          <Link className="btn btn-sm btn-primary" to={`/family/${m.profile_id}/income`}>Income &amp; tax</Link>
          {canEdit && <button type="button" className="btn btn-sm btn-outline" onClick={onEdit}>Edit</button>}
          {canEdit && <button type="button" className="btn btn-sm btn-ghost text-danger" onClick={onDelete}>Delete</button>}
        </div>
      )}
    </div>
  );
}

type MemberForm = CreateProfileRequest & { metro_city: boolean };

const emptyMember = (hasSelf: boolean): MemberForm => ({
  name: '', relationship: hasSelf ? 'spouse' : 'self', color: '#4f46e5', occupation: '', date_of_birth: '',
  residential_status: 'resident', disability: '', spouse_profile_id: '', preferred_regime: '', metro_city: false,
});

function MemberModal({ initial, editing, profiles, onClose, onSaved }: {
  initial: MemberForm; editing: Profile | null; profiles: Profile[]; onClose: () => void; onSaved: () => void;
}) {
  const { toast } = useToast();
  const [form, setForm] = useState<MemberForm>(initial);
  const [saving, setSaving] = useState(false);
  const set = (patch: Partial<MemberForm>) => setForm(f => ({ ...f, ...patch }));
  const others = profiles.filter(p => p.id !== editing?.id);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      if (editing) {
        await updateProfile(editing.id, form);
        toast('Member updated');
      } else {
        await createProfile(form);
        toast('Member added');
      }
      onSaved();
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to save member'), 'error');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={e => e.stopPropagation()}>
        <h2>{editing ? `Edit ${editing.name}` : 'Add family member'}</h2>
        <form onSubmit={submit}>
          <div className="form-row">
            <div className="form-group">
              <label>Name</label>
              <input required value={form.name} onChange={e => set({ name: e.target.value })} />
            </div>
            <div className="form-group">
              <label>Relationship to you</label>
              <select value={form.relationship} onChange={e => set({ relationship: e.target.value })}>
                {RELATIONSHIPS.map(r => <option key={r} value={r} className="capitalize">{r.charAt(0).toUpperCase() + r.slice(1)}</option>)}
              </select>
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label>Occupation</label>
              <select value={form.occupation || ''} onChange={e => set({ occupation: e.target.value })}>
                <option value="">Select…</option>
                {(Object.keys(OCCUPATION_LABELS) as Occupation[]).map(o => <option key={o} value={o}>{OCCUPATION_LABELS[o]}</option>)}
              </select>
            </div>
            <div className="form-group">
              <label>Date of birth</label>
              <input type="date" value={form.date_of_birth || ''} onChange={e => set({ date_of_birth: e.target.value })} />
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label>Tax regime declared / planned</label>
              <select value={form.preferred_regime || ''} onChange={e => set({ preferred_regime: e.target.value })}>
                <option value="">Not decided (recommend one)</option>
                <option value="new">New regime</option>
                <option value="old">Old regime</option>
              </select>
            </div>
            <div className="form-group">
              <label>Residential status</label>
              <select value={form.residential_status || 'resident'} onChange={e => set({ residential_status: e.target.value })}>
                <option value="resident">Resident</option>
                <option value="nri">Non-resident (NRI)</option>
              </select>
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label>Spouse (for clubbing rules)</label>
              <select value={form.spouse_profile_id || ''} onChange={e => set({ spouse_profile_id: e.target.value })}>
                <option value="">{form.relationship === 'self' || form.relationship === 'spouse' ? 'Auto (self ↔ spouse)' : 'None'}</option>
                {others.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
              </select>
            </div>
            <div className="form-group">
              <label>Disability (80U)</label>
              <select value={form.disability || ''} onChange={e => set({ disability: e.target.value })}>
                <option value="">None</option>
                <option value="disabled">40% or more</option>
                <option value="severe">80% or more (severe)</option>
              </select>
            </div>
          </div>
          <div className="form-row">
            <div className="form-group">
              <label className="flex items-center gap-2">
                <input type="checkbox" checked={form.metro_city} onChange={e => set({ metro_city: e.target.checked })} />
                Lives in a metro (Delhi, Mumbai, Kolkata, Chennai) — 50% HRA
              </label>
            </div>
            <div className="form-group">
              <label>Colour</label>
              <input type="color" value={form.color} onChange={e => set({ color: e.target.value })} style={{ width: 60, height: 36, padding: 2 }} />
            </div>
          </div>
          <p className="text-muted text-sm">PAN and Aadhaar are never needed or stored.</p>
          <div className="modal-actions">
            <button type="button" className="btn btn-outline" onClick={onClose}>Cancel</button>
            <button type="submit" className="btn btn-primary" disabled={saving}>{editing ? 'Save changes' : 'Add member'}</button>
          </div>
        </form>
      </div>
    </div>
  );
}

function ShareModal({ item, profiles, onClose, onSaved }: {
  item: AssetItem; profiles: Profile[]; onClose: () => void; onSaved: () => void;
}) {
  const { toast } = useToast();
  const initialShares: OwnershipShare[] = item.shares && item.shares.length > 0
    ? item.shares
    : [{ profile_id: item.profile_id || profiles[0]?.id || '', pct: 100 }];
  const [shares, setShares] = useState<OwnershipShare[]>(initialShares);
  const [fundedBy, setFundedBy] = useState(item.funded_by || '');
  const [saving, setSaving] = useState(false);
  const total = shares.reduce((s, x) => s + (Number(x.pct) || 0), 0);
  const valid = Math.abs(total - 100) < 0.01 && shares.every(s => s.profile_id) && new Set(shares.map(s => s.profile_id)).size === shares.length;

  const update = (i: number, patch: Partial<OwnershipShare>) => setShares(list => list.map((s, j) => (j === i ? { ...s, ...patch } : s)));
  const addRow = () => {
    const unused = profiles.find(p => !shares.some(s => s.profile_id === p.id));
    if (!unused) return;
    const remaining = Math.max(0, 100 - total);
    setShares([...shares, { profile_id: unused.id, pct: remaining }]);
  };
  const split = () => {
    const each = Math.floor((100 / shares.length) * 100) / 100;
    setShares(shares.map((s, i) => ({ ...s, pct: i === 0 ? Math.round((100 - each * (shares.length - 1)) * 100) / 100 : each })));
  };

  const save = async () => {
    setSaving(true);
    try {
      await familyApi.assign({ asset_type: item.asset_type, asset_id: item.asset_id, shares, funded_by: fundedBy });
      toast('Ownership saved');
      onSaved();
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to save ownership'), 'error');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={e => e.stopPropagation()}>
        <h2>Ownership — {item.name}</h2>
        <p className="text-muted text-sm mb-4">
          Split joint holdings (e.g. a home loan taken with your spouse) so each owner's income and deductions are counted in their own tax.
        </p>
        {shares.map((s, i) => (
          <div className="form-row" key={i}>
            <div className="form-group">
              <label>Owner {i + 1}</label>
              <select value={s.profile_id} onChange={e => update(i, { profile_id: e.target.value })}>
                {profiles.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
              </select>
            </div>
            <div className="form-group">
              <label>Share %</label>
              <div className="flex gap-2 items-center">
                <input type="number" min="0.01" max="100" step="0.01" value={s.pct} onChange={e => update(i, { pct: parseFloat(e.target.value) || 0 })} />
                {shares.length > 1 && (
                  <button type="button" className="btn btn-sm btn-ghost text-danger" onClick={() => setShares(shares.filter((_, j) => j !== i))} aria-label="Remove owner">✕</button>
                )}
              </div>
            </div>
          </div>
        ))}
        <div className="flex gap-2 mb-4">
          <button type="button" className="btn btn-sm btn-outline" onClick={addRow} disabled={shares.length >= profiles.length}>Add co-owner</button>
          {shares.length > 1 && <button type="button" className="btn btn-sm btn-outline" onClick={split}>Split equally</button>}
          <span className={`ml-auto text-sm ${valid ? 'text-muted' : 'text-danger'}`}>Total: {Math.round(total * 100) / 100}%</span>
        </div>
        <div className="form-group">
          <label>Funded by</label>
          <select value={fundedBy} onChange={e => setFundedBy(e.target.value)}>
            <option value="">The owner(s) themselves</option>
            {profiles.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
          </select>
          <div className="text-muted text-sm mt-1">
            If one person's money is in another's name, income from it is taxed back to the giver for a spouse or minor child (clubbing).
            Premiums / PPF paid for a spouse or child can be claimed by the payer.
          </div>
        </div>
        <div className="modal-actions">
          <button type="button" className="btn btn-outline" onClick={onClose}>Cancel</button>
          <button type="button" className="btn btn-primary" disabled={!valid || saving} onClick={save}>Save</button>
        </div>
      </div>
    </div>
  );
}

export default function FamilyPage() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const { toast } = useToast();
  const { confirm } = useConfirm();
  const [summary, setSummary] = useState<FamilySummary | null>(null);
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [error, setError] = useState('');
  const [reloadTick, setReloadTick] = useState(0);
  const [category, setCategory] = useState('all');
  const [search, setSearch] = useState('');
  const [onlyUnassigned, setOnlyUnassigned] = useState(false);
  const [saving, setSaving] = useState<string | null>(null);
  const [memberModal, setMemberModal] = useState<{ editing: Profile | null; form: MemberForm } | null>(null);
  const [shareItem, setShareItem] = useState<AssetItem | null>(null);

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

  const reload = () => setReloadTick(t => t + 1);
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
      await familyApi.assign({ asset_type: it.asset_type, asset_id: it.asset_id, profile_id: profileId, funded_by: profileId ? it.funded_by : '' });
      toast(profileId ? `Assigned to ${profileById.get(profileId)?.name || 'member'}` : 'Owner cleared');
      reload();
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to assign owner'), 'error');
    } finally {
      setSaving(null);
    }
  };

  const openAdd = () => setMemberModal({ editing: null, form: emptyMember(profiles.some(p => p.relationship === 'self')) });
  const openEdit = (p: Profile) => setMemberModal({
    editing: p,
    form: {
      name: p.name, relationship: p.relationship, color: p.color || '#4f46e5', occupation: p.occupation || '',
      date_of_birth: toInputDate(p.date_of_birth), residential_status: p.residential_status || 'resident',
      disability: p.disability || '', spouse_profile_id: p.spouse_profile_id || '', preferred_regime: p.preferred_regime || '',
      metro_city: !!p.metro_city,
    },
  });
  const handleDelete = async (p: Profile) => {
    if (!await confirm({
      title: 'Delete member',
      message: `Delete ${p.name}? Their income records are removed and their items become unassigned (joint shares are rescaled).`,
      danger: true, confirmLabel: 'Delete',
    })) return;
    try { await deleteProfile(p.id); toast('Member deleted'); reload(); }
    catch (err) { toast(apiErrorMessage(err, 'Failed to delete member'), 'error'); }
  };

  if (error && !summary) {
    return (
      <div className="empty-state">
        <h3>Couldn't load family view</h3>
        <p>{error}</p>
        <button className="btn btn-primary" onClick={reload}>Retry</button>
      </div>
    );
  }
  if (!summary) return <PageSkeleton label="Loading family" />;

  const totalNet = members.reduce((s, m) => s + m.net_worth, 0);
  const unassignedCount = items.filter(i => !i.profile_id).length;
  const ownerLabel = (it: AssetItem) => (it.shares || []).map(s => `${profileById.get(s.profile_id)?.name || '?'} ${s.pct}%`).join(' · ');

  return (
    <div>
      <div className="page-header">
        <h1>Family</h1>
        <div className="flex items-center gap-4">
          <span className="text-muted text-base">
            Household net worth: <strong style={{ color: 'var(--text)' }}>{formatCurrency(totalNet)}</strong>
          </span>
          <Link className="btn btn-outline" to="/tax-center">Family tax</Link>
          {!isViewOnly && <button className="btn btn-primary" onClick={openAdd}>Add member</button>}
        </div>
      </div>

      {profiles.length === 0 && (
        <div className="card mb-5" style={{ borderLeft: '4px solid var(--warning)' }}>
          <strong>No family members yet.</strong>{' '}
          <span className="text-muted">
            Add yourself, your spouse, parents and children with their occupation and date of birth. Then assign each investment or loan
            to its owner to get per-member tax estimates and regime recommendations.
          </span>
        </div>
      )}

      <div className="card-grid mb-6">
        {members.map(m => {
          const p = profileById.get(m.profile_id);
          return (
            <MemberCard key={m.profile_id || 'unassigned'} m={m} profile={p} canEdit={!isViewOnly && !!p}
              onEdit={() => p && openEdit(p)} onDelete={() => p && handleDelete(p)} />
          );
        })}
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
                  const joint = (it.shares || []).length > 1;
                  const funder = it.funded_by ? profileById.get(it.funded_by) : undefined;
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
                            {joint ? ownerLabel(it) : owner?.name || 'Unassigned'}
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-2">
                            {joint ? (
                              <span className="text-sm">{ownerLabel(it)}</span>
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
                            <button type="button" className="btn btn-sm btn-ghost" onClick={() => setShareItem(it)} title="Joint ownership and source of funds">
                              {joint ? 'Edit split' : 'Split / funded by'}
                            </button>
                          </span>
                        )}
                        {funder && <div className="text-muted text-xs">Funded by {funder.name}</div>}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </>
      )}

      {memberModal && (
        <MemberModal
          initial={memberModal.form}
          editing={memberModal.editing}
          profiles={profiles}
          onClose={() => setMemberModal(null)}
          onSaved={() => { setMemberModal(null); reload(); }}
        />
      )}
      {shareItem && (
        <ShareModal item={shareItem} profiles={profiles} onClose={() => setShareItem(null)} onSaved={() => { setShareItem(null); reload(); }} />
      )}
    </div>
  );
}
