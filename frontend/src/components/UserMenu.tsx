import { useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { ChevronDown, LogOut, Settings } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { authApi, type User as AuthUser } from '../api/auth';

function initials(name?: string) {
  const parts = (name || '').trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  return (parts[0][0] + (parts.length > 1 ? parts[parts.length - 1][0] : '')).toUpperCase();
}

export default function UserMenu() {
  const { user, isAdmin, viewAsUserId, logout, setViewAsUser } = useAuth();
  const [open, setOpen] = useState(false);
  const [allUsers, setAllUsers] = useState<AuthUser[]>([]);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (isAdmin) {
      authApi.getUsers().then(r => setAllUsers(r.data || [])).catch(() => {});
    }
  }, [isAdmin]);

  useEffect(() => {
    if (!open) return;
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false); };
    document.addEventListener('mousedown', onClick);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onClick);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  const otherUsers = allUsers.filter(u => u.id !== user?.id);
  const firstName = user?.name?.split(' ')[0] || 'Account';

  return (
    <div ref={ref} className="menu-anchor">
      <button className="user-menu-btn" onClick={() => setOpen(o => !o)} aria-haspopup="menu" aria-expanded={open} aria-label="Account menu">
        <span className="avatar">{initials(user?.name)}</span>
        <span className="user-name">{firstName}</span>
        <ChevronDown size={15} />
      </button>

      {open && (
        <div className="dropdown" role="menu">
          <div className="dropdown-header">
            <div className="name">{user?.name}</div>
            <div className="email">{user?.email}</div>
          </div>

          {isAdmin && otherUsers.length > 0 && (
            <>
              <label className="dropdown-label" htmlFor="view-as-user">View as user</label>
              <select
                id="view-as-user"
                value={viewAsUserId || ''}
                onChange={e => { setViewAsUser(e.target.value || null); setOpen(false); }}
              >
                <option value="">My data</option>
                {otherUsers.map(u => (
                  <option key={u.id} value={u.id}>{u.name} ({u.email})</option>
                ))}
              </select>
              <div className="dropdown-divider" />
            </>
          )}

          <Link to="/settings" className="dropdown-item" role="menuitem" onClick={() => setOpen(false)}>
            <Settings size={16} /> Settings
          </Link>
          <button className="dropdown-item danger" role="menuitem" onClick={logout}>
            <LogOut size={16} /> Sign out
          </button>
        </div>
      )}
    </div>
  );
}
