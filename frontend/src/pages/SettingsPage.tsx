import { useEffect, useState } from 'react';
import { getProfiles, createProfile, updateProfile } from '../api/profiles';
import { exportCSV, exportBackup } from '../api/exports';
import type { Profile, CreateProfileRequest, UpdateProfileRequest } from '../types';
import { useToast } from '../components/Toast';

export default function SettingsPage() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState<UpdateProfileRequest>({});
  const [createForm, setCreateForm] = useState<CreateProfileRequest>({ name: '', relationship: 'self', color: '#1a73e8' });
  const [msg, setMsg] = useState('');
  const { toast } = useToast();

  const load = () => { getProfiles().then(r => { const p = r.data?.[0] || null; setProfile(p); setForm(p ? { name: p.name, relationship: p.relationship, color: p.color } : {}); if (!p) setCreating(true); }).catch(() => {}).finally(() => setLoading(false)); };
  useEffect(() => { load(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try { await createProfile(createForm); setCreating(false); toast('Profile created'); load(); }
    catch { toast('Failed to create profile', 'error'); }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    try { if (profile?.id) await updateProfile(profile.id, form); setEditing(false); toast('Profile saved'); load(); }
    catch { toast('Failed to save profile', 'error'); }
  };

  const handleBackup = async () => {
    try {
      const res = await exportBackup();
      const blob = new Blob([res.data], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a'); a.href = url; a.download = `backup-${Date.now()}.json`; a.click(); URL.revokeObjectURL(url);
      toast('Backup downloaded');
    } catch { toast('Failed to download backup', 'error'); }
  };

  const handleExportCSV = async () => {
    try {
      const res = await exportCSV();
      const blob = new Blob([res.data], { type: 'text/csv' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a'); a.href = url; a.download = `investments.csv`; a.click(); URL.revokeObjectURL(url);
      toast('CSV exported');
    } catch { toast('Failed to export CSV', 'error'); }
  };

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h1>Settings</h1>
      </div>

      {msg && (
        <div style={{ padding: '10px 16px', marginBottom: 16, borderRadius: 8, background: '#e6f4ea', color: '#0f9d58', fontWeight: 500 }}>
          {msg}
        </div>
      )}

      <div className="card" style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
          <h3 style={{ fontSize: 16, fontWeight: 600 }}>Profile</h3>
          {!editing && !creating && profile && <button className="btn btn-sm btn-primary" onClick={() => setEditing(true)}>Edit</button>}
        </div>
        {creating ? (
          <form onSubmit={handleCreate}>
            <p className="text-muted" style={{ marginBottom: 16 }}>No profile found. Create one to get started.</p>
            <div className="form-row">
              <div className="form-group"><label>Name</label><input required value={createForm.name} onChange={e => setCreateForm({ ...createForm, name: e.target.value })} /></div>
              <div className="form-group">
                <label>Relationship</label>
                <select value={createForm.relationship} onChange={e => setCreateForm({ ...createForm, relationship: e.target.value })}>
                  <option value="self">Self</option><option value="spouse">Spouse</option><option value="parent">Parent</option>
                  <option value="child">Child</option><option value="sibling">Sibling</option><option value="other">Other</option>
                </select>
              </div>
            </div>
            <div className="form-group">
              <label>Color</label>
              <input type="color" value={createForm.color} onChange={e => setCreateForm({ ...createForm, color: e.target.value })} style={{ width: 60, height: 36, padding: 2 }} />
            </div>
            <div className="modal-actions">
              <button type="submit" className="btn btn-primary">Create Profile</button>
            </div>
          </form>
        ) : !editing ? (
          <div className="payout-list">
            <div className="payout-item"><span>Name</span><span>{profile?.name || '-'}</span></div>
            <div className="payout-item"><span>Relationship</span><span style={{ textTransform: 'capitalize' }}>{profile?.relationship || '-'}</span></div>
            <div className="payout-item"><span>Color</span><span style={{ display: 'flex', alignItems: 'center', gap: 8 }}><span style={{ width: 16, height: 16, borderRadius: '50%', background: profile?.color || '#ccc', display: 'inline-block' }}></span>{profile?.color || '-'}</span></div>
            <div className="payout-item"><span>Default</span><span>{profile?.is_default ? 'Yes' : 'No'}</span></div>
          </div>
        ) : (
          <form onSubmit={handleSave}>
            <div className="form-row">
              <div className="form-group"><label>Name</label><input value={form.name || ''} onChange={e => setForm({ ...form, name: e.target.value })} /></div>
              <div className="form-group">
                <label>Relationship</label>
                <select value={form.relationship || 'self'} onChange={e => setForm({ ...form, relationship: e.target.value })}>
                  <option value="self">Self</option><option value="spouse">Spouse</option><option value="parent">Parent</option>
                  <option value="child">Child</option><option value="sibling">Sibling</option><option value="other">Other</option>
                </select>
              </div>
            </div>
            <div className="form-group">
              <label>Color</label>
              <input type="color" value={form.color || '#1a73e8'} onChange={e => setForm({ ...form, color: e.target.value })} style={{ width: 60, height: 36, padding: 2 }} />
            </div>
            <div className="modal-actions">
              <button type="button" className="btn btn-outline" onClick={() => setEditing(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Save</button>
            </div>
          </form>
        )}
      </div>

      <div className="card" style={{ marginBottom: 16 }}>
        <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 12 }}>Backup & Export</h3>
        <p className="text-muted" style={{ fontSize: 13, marginBottom: 12 }}>Download a full backup or export your data.</p>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          <button className="btn btn-primary" onClick={handleBackup}>Download Backup (JSON)</button>
          <button className="btn btn-outline" onClick={handleExportCSV}>Export CSV</button>
        </div>
      </div>
    </div>
  );
}
