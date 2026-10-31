import { useState, useCallback, useMemo, useRef, type ReactNode } from 'react';
import { CircleCheck, CircleX, Info } from 'lucide-react';
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
      <div className="toast-stack">
        {toasts.map(t => {
          const Icon = t.type === 'success' ? CircleCheck : t.type === 'error' ? CircleX : Info;
          return (
            <div
              key={t.id}
              role={t.type === 'error' ? 'alert' : 'status'}
              className={`toast toast-${t.type}`}
              onClick={() => setToasts(prev => prev.filter(x => x.id !== t.id))}
            >
              <Icon size={18} aria-hidden="true" />
              {t.message}
            </div>
          );
        })}
      </div>
    </ToastContext.Provider>
  );
}
