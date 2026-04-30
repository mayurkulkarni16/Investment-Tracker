import { NavLink } from 'react-router-dom';
import { useEffect, useState, useRef, useCallback } from 'react';
import { getUnreadNotifications, generateNotifications, markNotificationRead, markAllNotificationsRead } from '../api/notifications';
import { useAuth } from '../context/AuthContext';
import { authApi, type User as AuthUser } from '../api/auth';
import type { Notification } from '../types';

export default function Sidebar() {
  const { user, isAdmin, viewAsUserId, logout, setViewAsUser } = useAuth();
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [showPanel, setShowPanel] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [dark, setDark] = useState(() => localStorage.getItem('theme') === 'dark');
  const [allUsers, setAllUsers] = useState<AuthUser[]>([]);
  const panelRef = useRef<HTMLDivElement>(null);

  const loadNotifications = () => {
    getUnreadNotifications().then(r => setNotifications(r.data || [])).catch(() => {});
  };

  useEffect(() => {
    loadNotifications();
    const interval = setInterval(loadNotifications, 60000); // refresh every minute
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (isAdmin) {
      authApi.getUsers().then(r => setAllUsers(r.data || [])).catch(() => {});
    }
  }, [isAdmin]);

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        setShowPanel(false);
      }
    };
    if (showPanel) document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [showPanel]);

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark);
    localStorage.setItem('theme', dark ? 'dark' : 'light');
  }, [dark]);

  const closeMobile = useCallback(() => setMobileOpen(false), []);

  const handleGenerate = async () => {
    await generateNotifications();
    loadNotifications();
  };

  const handleMarkRead = async (id: string) => {
    await markNotificationRead(id);
    setNotifications(prev => prev.filter(n => n.id !== id));
  };

  const handleMarkAllRead = async () => {
    await markAllNotificationsRead();
    setNotifications([]);
  };

  const getIcon = (type: string) => {
    switch (type) {
      case 'emi_due': return '🏦';
      case 'fd_maturity': return '📅';
      case 'sip_due': return '📊';
      case 'bond_coupon': return '💰';
      case 'credit_card_due': return '💳';
      case 'goal_milestone': return '🎯';
      default: return '🔔';
    }
  };

  return (
    <>
      <button className="hamburger" onClick={() => setMobileOpen(o => !o)} aria-label="Toggle menu">☰</button>
      {mobileOpen && <div className="sidebar-overlay" onClick={closeMobile} />}
      <aside className={`sidebar${mobileOpen ? ' open' : ''}`}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '0 20px 20px', borderBottom: '1px solid var(--border)', marginBottom: 8 }}>
        <h2 style={{ padding: 0, border: 'none', marginBottom: 0 }}>My Investments</h2>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <button
            onClick={() => setDark(d => !d)}
            style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: 18, padding: '4px 6px', borderRadius: 6 }}
            title={dark ? 'Switch to light mode' : 'Switch to dark mode'}
          >
            {dark ? '☀️' : '🌙'}
          </button>
          <div ref={panelRef} style={{ position: 'relative' }}>
          <button
            onClick={() => setShowPanel(!showPanel)}
            style={{
              background: 'none', border: 'none', cursor: 'pointer', fontSize: 20, position: 'relative',
              padding: '4px 6px', borderRadius: 6, transition: 'background 0.15s',
            }}
            onMouseOver={e => (e.currentTarget.style.background = '#f0f4ff')}
            onMouseOut={e => (e.currentTarget.style.background = 'none')}
            title="Notifications"
          >
            🔔
            {notifications.length > 0 && (
              <span style={{
                position: 'absolute', top: 0, right: 0, background: 'var(--danger)', color: 'white',
                borderRadius: '50%', width: 18, height: 18, fontSize: 10, fontWeight: 700,
                display: 'flex', alignItems: 'center', justifyContent: 'center',
              }}>
                {notifications.length > 9 ? '9+' : notifications.length}
              </span>
            )}
          </button>

          {showPanel && (
            <div style={{
              position: 'absolute', top: 36, right: -60, width: 340, maxHeight: 440,
              background: 'var(--card-bg)', borderRadius: 'var(--radius)', boxShadow: '0 8px 30px rgba(0,0,0,0.18)',
              border: '1px solid var(--border)', zIndex: 1001, overflow: 'hidden',
            }}>
              <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--border)', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <strong style={{ fontSize: 14 }}>Notifications</strong>
                <div style={{ display: 'flex', gap: 6 }}>
                  <button className="btn btn-sm btn-outline" onClick={handleGenerate} title="Scan for upcoming due dates">Scan</button>
                  {notifications.length > 0 && <button className="btn btn-sm btn-outline" onClick={handleMarkAllRead}>Clear All</button>}
                </div>
              </div>
              <div style={{ maxHeight: 380, overflowY: 'auto' }}>
                {notifications.length === 0 ? (
                  <div style={{ padding: 24, textAlign: 'center', color: 'var(--text-secondary)', fontSize: 13 }}>
                    No new notifications.<br />
                    <span style={{ fontSize: 12 }}>Click "Scan" to check for upcoming dues.</span>
                  </div>
                ) : (
                  notifications.map(n => (
                    <div
                      key={n.id}
                      style={{
                        padding: '10px 16px', borderBottom: '1px solid var(--border)',
                        display: 'flex', gap: 10, alignItems: 'flex-start', cursor: 'pointer',
                        transition: 'background 0.1s',
                      }}
                      onClick={() => handleMarkRead(n.id)}
                      onMouseOver={e => (e.currentTarget.style.background = '#f8f9fa')}
                      onMouseOut={e => (e.currentTarget.style.background = 'transparent')}
                    >
                      <span style={{ fontSize: 20, flexShrink: 0, marginTop: 2 }}>{getIcon(n.type)}</span>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ fontSize: 13, fontWeight: 600 }}>{n.title}</div>
                        <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginTop: 2 }}>{n.message}</div>
                        <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 4 }}>
                          {new Date(n.date).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })}
                        </div>
                      </div>
                      <button
                        style={{ background: 'none', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer', fontSize: 16, padding: 2, flexShrink: 0 }}
                        onClick={e => { e.stopPropagation(); handleMarkRead(n.id); }}
                        title="Dismiss"
                      >✕</button>
                    </div>
                  ))
                )}
              </div>
            </div>
          )}
        </div>
        </div>
      </div>
      <nav onClick={closeMobile}>
        <NavLink to="/" end>Dashboard</NavLink>
        <NavLink to="/net-worth">Net Worth</NavLink>
        <NavLink to="/mutual-funds">Mutual Funds</NavLink>
        <NavLink to="/stocks">Stocks</NavLink>
        <NavLink to="/fixed-deposits">Fixed Deposits</NavLink>
        <NavLink to="/corporate-bonds">Corporate Bonds</NavLink>
        <NavLink to="/provident-fund">Provident Fund</NavLink>
        <NavLink to="/nps">NPS</NavLink>
        <NavLink to="/home-loans">Home Loans</NavLink>
        <NavLink to="/personal-loans">Personal Loans</NavLink>
        <NavLink to="/credit-cards">Credit Cards</NavLink>
        <NavLink to="/goals">Goals</NavLink>
        <NavLink to="/tax-center">Tax Center</NavLink>
        <NavLink to="/projections">Projections</NavLink>
        <NavLink to="/cashflow">Cashflow</NavLink>
        <NavLink to="/insights">Insights</NavLink>
        <NavLink to="/settings">Settings</NavLink>
      </nav>

      {/* Admin view-as dropdown */}
      {isAdmin && allUsers.length > 0 && (
        <div style={{ padding: '12px 20px', borderTop: '1px solid var(--border)' }}>
          <label style={{ fontSize: 11, color: 'var(--text-secondary)', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.5px' }}>
            View as User
          </label>
          <select
            value={viewAsUserId || ''}
            onChange={e => setViewAsUser(e.target.value || null)}
            style={{
              width: '100%', marginTop: 4, padding: '6px 8px', borderRadius: 6,
              border: '1px solid var(--border)', background: 'var(--bg)', color: 'var(--text)',
              fontSize: 13, cursor: 'pointer',
            }}
          >
            <option value="">My Data</option>
            {allUsers.filter(u => u.id !== user?.id).map(u => (
              <option key={u.id} value={u.id}>{u.name} ({u.email})</option>
            ))}
          </select>
        </div>
      )}

      {/* User info + logout */}
      <div style={{ padding: '12px 20px', borderTop: '1px solid var(--border)', marginTop: 'auto' }}>
        <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--text)', marginBottom: 2 }}>{user?.name}</div>
        <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginBottom: 8 }}>{user?.email}</div>
        <button
          onClick={logout}
          className="btn btn-sm btn-outline"
          style={{ width: '100%' }}
        >
          Sign Out
        </button>
      </div>
    </aside>
    </>
  );
}
