import { useEffect, useState } from 'react';
import { getProfiles, createProfile, updateProfile } from '../api/profiles';
import { exportCSV, exportBackup, restoreBackup } from '../api/exports';
import { authApi } from '../api/auth';
import { useAuth } from '../context/AuthContext';
import type { Profile, CreateProfileRequest, UpdateProfileRequest } from '../types';
import { useToast } from '../components/Toast';
import { useConfirm } from '../components/ConfirmDialog';

export default function SettingsPage() {
  const { user, updateUser, viewAsUserId } = useAuth();
  const [profile, setProfile] = useState<Profile | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState<UpdateProfileRequest>({});
  const [createForm, setCreateForm] = useState<CreateProfileRequest>({ name: '', relationship: 'self', color: '#1a73e8' });
  const [msg, setMsg] = useState('');
  const [restoring, setRestoring] = useState(false);
  const [editingAccount, setEditingAccount] = useState(false);
  const [accountName, setAccountName] = useState(user?.name || '');
  const [changingPw, setChangingPw] = useState(false);
  const [pwForm, setPwForm] = useState({ current_password: '', new_password: '', confirm_password: '' });
  const { toast } = useToast();
  const { confirm } = useConfirm();

  const load = () => { getProfiles().then(r => { const p = r.data?.[0] || null; setProfile(p); setForm(p ? { name: p.name, relationship: p.relationship, color: p.color } : {}); if (!p) setCreating(true); }).catch(() => {}).finally(() => setLoading(false)); };
  useEffect(() => { load(); }, [viewAsUserId]);

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

  const handleRestore = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!await confirm({ message: 'This will replace ALL existing data with the backup. Are you sure?', danger: true, confirmLabel: 'Restore' })) {
      e.target.value = '';
      return;
    }
    setRestoring(true);
    try {
      const text = await file.text();
      JSON.parse(text); // validate JSON
      const res = await restoreBackup(text);
      toast(`Restored ${res.data.collections_restored} collections`);
      setMsg(`Restore complete — ${res.data.collections_restored} collections restored.${res.data.errors?.length ? ' Errors: ' + res.data.errors.join(', ') : ''}`);
    } catch (err) {
      toast('Failed to restore backup', 'error');
    } finally {
      setRestoring(false);
      e.target.value = '';
    }
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
        <p className="text-muted" style={{ fontSize: 13, marginBottom: 12 }}>Download a full backup, restore from a previous backup, or export your data.</p>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
          <button className="btn btn-primary" onClick={handleBackup}>Download Backup (JSON)</button>
          <label className="btn btn-outline" style={{ cursor: restoring ? 'wait' : 'pointer', opacity: restoring ? 0.6 : 1 }}>
            {restoring ? 'Restoring...' : 'Restore from Backup'}
            <input type="file" accept=".json" onChange={handleRestore} disabled={restoring} style={{ display: 'none' }} />
          </label>
          <button className="btn btn-outline" onClick={handleExportCSV}>Export CSV</button>
        </div>
        <p className="text-muted" style={{ fontSize: 11, marginTop: 8 }}>⚠️ Restore will replace all existing data with the backup contents.</p>
      </div>

      {/* Account section */}
      <div className="card" style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
          <h3 style={{ fontSize: 16, fontWeight: 600 }}>Account</h3>
          {!editingAccount && <button className="btn btn-sm btn-primary" onClick={() => { setEditingAccount(true); setAccountName(user?.name || ''); }}>Edit</button>}
        </div>
        {!editingAccount ? (
          <div className="payout-list">
            <div className="payout-item"><span>Name</span><span>{user?.name || '-'}</span></div>
            <div className="payout-item"><span>Email</span><span>{user?.email || '-'}</span></div>
            <div className="payout-item"><span>Role</span><span style={{ textTransform: 'capitalize' }}>{user?.role || '-'}</span></div>
          </div>
        ) : (
          <form onSubmit={async (e) => {
            e.preventDefault();
            try {
              const res = await authApi.updateMe({ name: accountName });
              updateUser(res.data);
              setEditingAccount(false);
              toast('Account updated');
            } catch { toast('Failed to update account', 'error'); }
          }}>
            <div className="form-group"><label>Name</label><input required value={accountName} onChange={e => setAccountName(e.target.value)} /></div>
            <div className="form-group"><label>Email</label><input value={user?.email || ''} disabled style={{ opacity: 0.6 }} /></div>
            <div className="modal-actions">
              <button type="button" className="btn btn-outline" onClick={() => setEditingAccount(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Save</button>
            </div>
          </form>
        )}
      </div>

      {/* Change Password */}
      <div className="card" style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
          <h3 style={{ fontSize: 16, fontWeight: 600 }}>Change Password</h3>
        </div>
        <form onSubmit={async (e) => {
          e.preventDefault();
          if (pwForm.new_password !== pwForm.confirm_password) {
            toast('Passwords do not match', 'error');
            return;
          }
          setChangingPw(true);
          try {
            await authApi.changePassword({ current_password: pwForm.current_password, new_password: pwForm.new_password });
            toast('Password changed');
            setPwForm({ current_password: '', new_password: '', confirm_password: '' });
          } catch (err: any) {
            toast(err.response?.data?.error || 'Failed to change password', 'error');
          } finally {
            setChangingPw(false);
          }
        }}>
          <div className="form-group"><label>Current Password</label><input type="password" required value={pwForm.current_password} onChange={e => setPwForm({ ...pwForm, current_password: e.target.value })} /></div>
          <div className="form-row">
            <div className="form-group"><label>New Password</label><input type="password" required minLength={6} value={pwForm.new_password} onChange={e => setPwForm({ ...pwForm, new_password: e.target.value })} /></div>
            <div className="form-group"><label>Confirm New Password</label><input type="password" required minLength={6} value={pwForm.confirm_password} onChange={e => setPwForm({ ...pwForm, confirm_password: e.target.value })} /></div>
          </div>
          <div className="modal-actions">
            <button type="submit" className="btn btn-primary" disabled={changingPw}>{changingPw ? 'Changing...' : 'Change Password'}</button>
          </div>
        </form>
      </div>
    </div>
  );
}
