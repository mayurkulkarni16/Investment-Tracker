import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react';
import { createPortal } from 'react-dom';
import { useNavigate } from 'react-router-dom';
import { ArrowDown, ArrowUp, CornerDownLeft, Search } from 'lucide-react';
import { NAV_SECTIONS, type NavItem } from './navigation';
import { SHORTCUTS } from '../utils/shortcuts';

const SHORTCUT_BY_PATH = Object.fromEntries(Object.entries(SHORTCUTS).map(([key, path]) => [path, key.toUpperCase()]));

interface Result extends NavItem {
  section: string;
}

export default function CommandPalette({ onClose }: { onClose: () => void }) {
  const navigate = useNavigate();
  const [query, setQuery] = useState('');
  const [active, setActive] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);

  const results = useMemo<Result[]>(() => {
    const q = query.trim().toLowerCase();
    const all = NAV_SECTIONS.flatMap(s => s.items.map(item => ({ ...item, section: s.title })));
    if (!q) return all;
    const terms = q.split(/\s+/);
    return all.filter(r => {
      const words = `${r.label} ${r.section} ${r.keywords || ''}`.toLowerCase().split(/[^a-z0-9]+/);
      return terms.every(t => words.some(w => w.startsWith(t)));
    });
  }, [query]);

  useEffect(() => {
    listRef.current?.querySelector('.palette-item.active')?.scrollIntoView({ block: 'nearest' });
  }, [active]);

  const go = (r?: Result) => {
    if (!r) return;
    navigate(r.to);
    onClose();
  };

  const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setActive(i => (results.length ? (i + 1) % results.length : 0));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setActive(i => (results.length ? (i - 1 + results.length) % results.length : 0));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      go(results[active]);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      onClose();
    }
  };

  return createPortal(
    <div className="palette-overlay" onMouseDown={e => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="palette" role="dialog" aria-modal="true" aria-label="Search pages">
        <div className="palette-input-row">
          <Search size={18} aria-hidden="true" />
          <input
            autoFocus
            value={query}
            onChange={e => { setQuery(e.target.value); setActive(0); }}
            onKeyDown={onKeyDown}
            placeholder="Search pages, e.g. stocks, emi, 80c…"
            aria-label="Search pages"
            role="combobox"
            aria-expanded="true"
            aria-controls="palette-results"
            aria-activedescendant={results[active] ? `palette-${results[active].to}` : undefined}
          />
          <kbd>Esc</kbd>
        </div>
        <div className="palette-list" id="palette-results" role="listbox" ref={listRef}>
          {results.length === 0 && <div className="palette-empty">No pages match “{query}”</div>}
          {results.map((r, i) => {
            const showSection = i === 0 || r.section !== results[i - 1].section;
            const Icon = r.icon;
            const shortcut = SHORTCUT_BY_PATH[r.to];
            return (
              <div key={r.to}>
                {showSection && <div className="palette-group-title">{r.section}</div>}
                <div
                  id={`palette-${r.to}`}
                  role="option"
                  aria-selected={i === active}
                  className={`palette-item${i === active ? ' active' : ''}`}
                  onMouseMove={() => setActive(i)}
                  onClick={() => go(r)}
                >
                  <Icon size={17} aria-hidden="true" />
                  {r.label}
                  {shortcut && <kbd>Alt {shortcut}</kbd>}
                </div>
              </div>
            );
          })}
        </div>
        <div className="palette-footer">
          <span><ArrowUp size={12} /><ArrowDown size={12} /> navigate</span>
          <span><CornerDownLeft size={12} /> open</span>
        </div>
      </div>
    </div>,
    document.body,
  );
}
