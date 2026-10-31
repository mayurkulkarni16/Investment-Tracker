import { useState, useCallback, useMemo, useRef, type ReactNode } from 'react';
import { ToastContext, type ToastType } from './Toast';

interface ToastItem {
  id: number;
  message: string;
  type: ToastType;
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const nextId = useRef(0);

  const toast = useCallback((message: string, type: ToastType = 'success') => {
    const id = ++nextId.current;
    setToasts(prev => [...prev, { id, message, type }]);
    setTimeout(() => setToasts(prev => prev.filter(t => t.id !== id)), 3000);
  }, []);

  const value = useMemo(() => ({ toast }), [toast]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div style={{ position: 'fixed', bottom: 24, right: 24, zIndex: 9999, display: 'flex', flexDirection: 'column', gap: 8 }}>
        {toasts.map(t => (
          <div
            key={t.id}
            onClick={() => setToasts(prev => prev.filter(x => x.id !== t.id))}
            style={{
              padding: '12px 20px',
              borderRadius: 8,
              color: '#fff',
              fontSize: 14,
              fontWeight: 500,
              cursor: 'pointer',
              boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
              animation: 'slideIn 0.2s ease-out',
              background: t.type === 'success' ? '#0f9d58' : t.type === 'error' ? '#ea4335' : '#1a73e8',
              minWidth: 250,
              maxWidth: 420,
            }}
          >
            {t.type === 'success' ? '✓ ' : t.type === 'error' ? '✗ ' : 'ℹ '}{t.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
