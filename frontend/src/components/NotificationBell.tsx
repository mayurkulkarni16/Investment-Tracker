import { useCallback, useEffect, useRef, useState } from 'react';
import {
  Bell, Bike, CalendarClock, ChartPie, Coins, CreditCard, FileText, Hourglass, Landmark, ShieldCheck, Target,
  TriangleAlert, Wallet, X, type LucideIcon,
} from 'lucide-react';
import { generateNotifications, getUnreadNotifications, markAllNotificationsRead, markNotificationRead } from '../api/notifications';
import { useAuth } from '../context/AuthContext';
import type { Notification } from '../types';

const TYPE_ICONS: Record<string, LucideIcon> = {
  emi_due: Landmark,
  fd_maturity: CalendarClock,
  sip_due: ChartPie,
  bond_coupon: Coins,
  credit_card_due: CreditCard,
  goal_milestone: Target,
  rental_overdue: TriangleAlert,
  rental_buyback: Bike,
  insurance_premium: ShieldCheck,
  policy_expiry: Hourglass,
  budget_over: Wallet,
  document_expiry: FileText,
};

export default function NotificationBell() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [open, setOpen] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);

  const load = useCallback(() => {
    getUnreadNotifications().then(r => setNotifications(r.data || [])).catch(() => {});
  }, []);

  // Scan for new alerts on load (the server de-duplicates, so repeat scans are
  // harmless) and refresh the list every minute. Scanning is a write, so it's
  // skipped while an admin is viewing another user's data.
  useEffect(() => {
    if (isViewOnly) {
      load();
    } else {
      generateNotifications().catch(() => {}).finally(load);
    }
    const interval = setInterval(load, 60000);
    return () => clearInterval(interval);
  }, [isViewOnly, viewAsUserId, load]);

  useEffect(() => {
    if (!open) return;
    const onClick = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false); };
    document.addEventListener('mousedown', onClick);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onClick);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  const handleGenerate = async () => {
    await generateNotifications();
    load();
  };

  const handleMarkRead = async (id: string) => {
    await markNotificationRead(id);
    setNotifications(prev => prev.filter(n => n.id !== id));
  };

  const handleMarkAllRead = async () => {
    await markAllNotificationsRead();
    setNotifications([]);
  };

  const count = notifications.length;

  return (
    <div ref={panelRef} className="menu-anchor">
      <button
        className="icon-btn"
        onClick={() => setOpen(o => !o)}
        title="Notifications"
        aria-label={count > 0 ? `Notifications (${count} unread)` : 'Notifications'}
        aria-expanded={open}
      >
        <Bell size={19} strokeWidth={1.9} />
        {count > 0 && <span className="count-badge">{count > 9 ? '9+' : count}</span>}
      </button>

      {open && (
        <div className="dropdown notif-panel" role="dialog" aria-label="Notifications">
          <div className="notif-panel-header">
            <strong className="text-base">Notifications</strong>
            <div className="flex gap-2">
              {!isViewOnly && <button className="btn btn-sm btn-outline" onClick={handleGenerate} title="Scan for upcoming due dates">Scan</button>}
              {count > 0 && <button className="btn btn-sm btn-outline" onClick={handleMarkAllRead}>Clear all</button>}
            </div>
          </div>
          <div className="notif-list">
            {count === 0 ? (
              <div className="notif-empty">
                No new notifications.<br />
                <span className="text-sm">Due dates are checked automatically every day.</span>
              </div>
            ) : (
              notifications.map(n => {
                const Icon = TYPE_ICONS[n.type] || Bell;
                return (
                  <div key={n.id} className="notif-item" onClick={() => handleMarkRead(n.id)}>
                    <span className="notif-icon"><Icon size={16} aria-hidden="true" /></span>
                    <div className="flex-1" style={{ minWidth: 0 }}>
                      <div className="text-md fw-semibold">{n.title}</div>
                      <div className="text-sm text-muted" style={{ marginTop: 2 }}>{n.message}</div>
                      <div className="text-xs text-muted mt-1">
                        {new Date(n.date).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })}
                      </div>
                    </div>
                    <button
                      className="icon-btn"
                      style={{ width: 26, height: 26 }}
                      onClick={e => { e.stopPropagation(); handleMarkRead(n.id); }}
                      title="Dismiss"
                      aria-label="Dismiss notification"
                    >
                      <X size={14} />
                    </button>
                  </div>
                );
              })
            )}
          </div>
        </div>
      )}
    </div>
  );
}
