import { useState, useCallback, useMemo, type ReactNode } from 'react';
import { ConfirmContext, type ConfirmOptions } from './ConfirmDialog';

export function ConfirmProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<(ConfirmOptions & { resolve: (v: boolean) => void }) | null>(null);

  const confirm = useCallback((options: ConfirmOptions) => {
    return new Promise<boolean>(resolve => {
      setState({ ...options, resolve });
    });
  }, []);

  const value = useMemo(() => ({ confirm }), [confirm]);

  const handleClose = (result: boolean) => {
    state?.resolve(result);
    setState(null);
  };

  return (
    <ConfirmContext.Provider value={value}>
      {children}
      {state && (
        <div className="modal-overlay" onClick={() => handleClose(false)}>
          <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 400 }}>
            <h2>{state.title || 'Confirm'}</h2>
            <p style={{ margin: '16px 0' }}>{state.message}</p>
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <button className="btn btn-outline" onClick={() => handleClose(false)}>Cancel</button>
              <button className={`btn ${state.danger ? 'btn-danger' : 'btn-primary'}`} onClick={() => handleClose(true)}>
                {state.confirmLabel || 'Confirm'}
              </button>
            </div>
          </div>
        </div>
      )}
    </ConfirmContext.Provider>
  );
}
