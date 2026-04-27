import { NavLink } from 'react-router-dom';
import { useEffect, useState, useRef } from 'react';
import { getUnreadNotifications, generateNotifications, markNotificationRead, markAllNotificationsRead } from '../api/notifications';
import type { Notification } from '../types';

export default function Sidebar() {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [showPanel, setShowPanel] = useState(false);
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
    const handleClick = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        setShowPanel(false);
      }
    };
    if (showPanel) document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [showPanel]);

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
      default: return '🔔';
    }
  };

  return (
    <aside className="sidebar">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '0 20px 20px', borderBottom: '1px solid var(--border)', marginBottom: 8 }}>
        <h2 style={{ padding: 0, border: 'none', marginBottom: 0 }}>My Investments</h2>
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
      <nav>
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
        <NavLink to="/settings">Settings</NavLink>
      </nav>
    </aside>
  );
}
