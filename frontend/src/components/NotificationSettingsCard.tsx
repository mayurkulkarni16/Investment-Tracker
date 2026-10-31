import { useEffect, useState } from 'react';
import { notificationSettingsApi } from '../api/planning';
import type { NotificationSettings, NotificationSettingsRequest } from '../api/planning';
import { useToast } from './Toast';
import { useAuth } from '../context/AuthContext';
import { apiErrorMessage } from '../utils/errors';
import { formatDate } from '../utils/format';

const emptyForm: NotificationSettingsRequest = { email_enabled: false, email: '', telegram_enabled: false, telegram_chat_id: '' };

function toForm(s: NotificationSettings): NotificationSettingsRequest {
  return {
    email_enabled: s.email_enabled,
    email: s.email || '',
    telegram_enabled: s.telegram_enabled,
    telegram_chat_id: s.telegram_chat_id || '',
  };
}

export default function NotificationSettingsCard() {
  const { isViewOnly, viewAsUserId } = useAuth();
  const { toast } = useToast();
  const [settings, setSettings] = useState<NotificationSettings | null>(null);
  const [form, setForm] = useState<NotificationSettingsRequest>(emptyForm);
  const [loadError, setLoadError] = useState('');
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);

  useEffect(() => {
    let cancelled = false;
    notificationSettingsApi.get()
      .then(r => {
        if (cancelled) return;
        setSettings(r.data);
        setForm(toForm(r.data));
        setLoadError('');
      })
      .catch(err => { if (!cancelled) setLoadError(apiErrorMessage(err, 'Failed to load reminder settings')); });
    return () => { cancelled = true; };
  }, [viewAsUserId]);

  const emailAvailable = settings?.email_available ?? false;
  const telegramAvailable = settings?.telegram_available ?? false;
  const dirty = settings !== null && JSON.stringify(toForm(settings)) !== JSON.stringify(form);
  const disabled = isViewOnly || settings === null;

  const save = async () => {
    if (form.email_enabled && !form.email.trim()) { toast('Enter an email address', 'error'); return; }
    if (form.telegram_enabled && !form.telegram_chat_id.trim()) { toast('Enter your Telegram chat ID', 'error'); return; }
    setSaving(true);
    try {
      const r = await notificationSettingsApi.update({ ...form, email: form.email.trim(), telegram_chat_id: form.telegram_chat_id.trim() });
      setSettings(r.data);
      setForm(toForm(r.data));
      toast('Reminder settings saved');
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to save reminder settings'), 'error');
    } finally {
      setSaving(false);
    }
  };

  const sendTest = async () => {
    setTesting(true);
    try {
      const r = await notificationSettingsApi.test();
      const sent = r.data.sent || [];
      toast(sent.length ? `Test sent via ${sent.join(' & ')}` : (r.data.message || 'Test sent'));
    } catch (err) {
      toast(apiErrorMessage(err, 'Failed to send test message'), 'error');
    } finally {
      setTesting(false);
    }
  };

  const canTest = !!settings && !dirty && ((settings.email_enabled && emailAvailable) || (settings.telegram_enabled && telegramAvailable));

  return (
    <div className="card" style={{ marginBottom: 16 }}>
      <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 4 }}>Reminders</h3>
      <p className="text-muted" style={{ fontSize: 13, marginBottom: 12 }}>
        Get maturity, premium, EMI and document-expiry reminders outside the app.
        {settings?.last_dispatch_at && <> Last sent {formatDate(settings.last_dispatch_at)}.</>}
      </p>

      {loadError && <div className="text-danger" style={{ fontSize: 13, marginBottom: 12 }}>{loadError}</div>}
      {!settings && !loadError && <div className="text-muted" style={{ fontSize: 13 }}>Loading…</div>}

      {settings && (
        <>
          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontWeight: 500, opacity: emailAvailable ? 1 : 0.6 }}>
              <input
                type="checkbox"
                checked={form.email_enabled}
                disabled={disabled || (!emailAvailable && !form.email_enabled)}
                onChange={e => setForm({ ...form, email_enabled: e.target.checked })}
              />
              Email
            </label>
            {!emailAvailable && (
              <div className="text-muted" style={{ fontSize: 12, margin: '4px 0 0 24px' }}>
                Email is not configured on this server. The server admin must set the SMTP_* environment variables.
              </div>
            )}
            {form.email_enabled && (
              <div className="form-group" style={{ margin: '8px 0 0 24px', maxWidth: 360 }}>
                <input type="email" placeholder="you@example.com" value={form.email} disabled={disabled}
                  onChange={e => setForm({ ...form, email: e.target.value })} />
              </div>
            )}
          </div>

          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: 8, fontWeight: 500, opacity: telegramAvailable ? 1 : 0.6 }}>
              <input
                type="checkbox"
                checked={form.telegram_enabled}
                disabled={disabled || (!telegramAvailable && !form.telegram_enabled)}
                onChange={e => setForm({ ...form, telegram_enabled: e.target.checked })}
              />
              Telegram
            </label>
            {!telegramAvailable && (
              <div className="text-muted" style={{ fontSize: 12, margin: '4px 0 0 24px' }}>
                Telegram is not configured on this server. The server admin must set the TELEGRAM_BOT_TOKEN environment variable.
              </div>
            )}
            {form.telegram_enabled && (
              <div style={{ margin: '8px 0 0 24px', maxWidth: 360 }}>
                <div className="form-group" style={{ marginBottom: 4 }}>
                  <input placeholder="Chat ID, e.g. 123456789" value={form.telegram_chat_id} disabled={disabled}
                    onChange={e => setForm({ ...form, telegram_chat_id: e.target.value })} />
                </div>
                <div className="text-muted" style={{ fontSize: 12 }}>
                  First send any message to this app's Telegram bot (so it is allowed to message you), then get your
                  numeric chat ID by messaging <strong>@userinfobot</strong> on Telegram.
                </div>
              </div>
            )}
          </div>

          {!isViewOnly && (
            <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
              <button className="btn btn-sm btn-primary" onClick={save} disabled={saving || !dirty}>
                {saving ? 'Saving…' : 'Save'}
              </button>
              <button className="btn btn-sm btn-outline" onClick={sendTest} disabled={testing || !canTest}
                title={dirty ? 'Save your changes first' : canTest ? undefined : 'Enable a configured channel first'}>
                {testing ? 'Sending…' : 'Send test'}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
