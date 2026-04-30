import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

const SHORTCUTS: Record<string, string> = {
  'd': '/',
  'm': '/mutual-funds',
  's': '/stocks',
  'f': '/fixed-deposits',
  'b': '/corporate-bonds',
  'p': '/provident-fund',
  'n': '/nps',
  'g': '/goals',
  'w': '/net-worth',
  'c': '/cashflow',
  'i': '/insights',
};

export function useKeyboardShortcuts() {
  const navigate = useNavigate();

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Don't trigger when typing in inputs
      const tag = (e.target as HTMLElement).tagName;
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;
      if (e.ctrlKey || e.metaKey || e.shiftKey) return;

      if (e.altKey) {
        const path = SHORTCUTS[e.key.toLowerCase()];
        if (path) {
          e.preventDefault();
          navigate(path);
        }
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [navigate]);
}
