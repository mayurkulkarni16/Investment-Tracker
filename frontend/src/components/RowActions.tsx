import { useEffect, useLayoutEffect, useRef, useState, type MouseEvent } from 'react';
import { createPortal } from 'react-dom';
import { Ellipsis, type LucideIcon } from 'lucide-react';

export interface RowAction {
  label: string;
  onClick: () => void;
  icon?: LucideIcon;
  // Shorter text when the action is shown as a visible button (the full label becomes its tooltip).
  inlineLabel?: string;
  danger?: boolean;
  hidden?: boolean;
  disabled?: boolean;
}

interface RowActionsProps {
  actions: RowAction[];
  // How many of the leading actions to show as visible buttons; the rest go in the ⋯ menu.
  inline?: number;
  label?: string;
}

const MENU_WIDTH = 190;

export default function RowActions({ actions, inline = 0, label = 'More actions' }: RowActionsProps) {
  const visible = actions.filter(a => !a.hidden);
  // Destructive actions always live at the bottom of the menu, never inline.
  const safe = visible.filter(a => !a.danger);
  const danger = visible.filter(a => a.danger);
  const shown = safe.slice(0, inline);
  const menu = [...safe.slice(inline), ...danger];

  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState<{ top?: number; bottom?: number; left: number } | null>(null);
  const btnRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const close = () => setOpen(false);

  useLayoutEffect(() => {
    if (!open || !btnRef.current) return;
    const r = btnRef.current.getBoundingClientRect();
    const estHeight = menu.length * 36 + 12;
    const up = r.bottom + estHeight + 8 > window.innerHeight && r.top > estHeight;
    const left = Math.max(8, Math.min(window.innerWidth - MENU_WIDTH - 8, r.right - MENU_WIDTH));
    setPos(up ? { bottom: window.innerHeight - r.top + 4, left } : { top: r.bottom + 4, left });
  }, [open, menu.length]);

  useEffect(() => {
    if (!open) return;
    const dismiss = () => setOpen(false);
    const onDown = (e: Event) => {
      const t = e.target as Node;
      if (menuRef.current?.contains(t) || btnRef.current?.contains(t)) return;
      dismiss();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { e.stopPropagation(); dismiss(); btnRef.current?.focus(); }
    };
    document.addEventListener('mousedown', onDown);
    document.addEventListener('keydown', onKey, true);
    window.addEventListener('scroll', dismiss, true);
    window.addEventListener('resize', dismiss);
    return () => {
      document.removeEventListener('mousedown', onDown);
      document.removeEventListener('keydown', onKey, true);
      window.removeEventListener('scroll', dismiss, true);
      window.removeEventListener('resize', dismiss);
    };
  }, [open]);

  useEffect(() => {
    if (open) menuRef.current?.querySelector<HTMLButtonElement>('button:not(:disabled)')?.focus();
  }, [open, pos]);

  if (visible.length === 0) return null;

  // Rows are often clickable (expand/collapse); keep action clicks from reaching them.
  const stop = (e: MouseEvent) => e.stopPropagation();

  return (
    <div className="row-actions" onClick={stop}>
      {shown.map(a => {
        const Icon = a.icon;
        return (
          <button key={a.label} type="button" className="btn btn-sm btn-ghost" disabled={a.disabled} onClick={a.onClick} title={a.inlineLabel ? a.label : undefined}>
            {Icon && <Icon size={14} aria-hidden="true" />}{a.inlineLabel || a.label}
          </button>
        );
      })}
      {menu.length > 0 && (
        <button
          ref={btnRef}
          type="button"
          className={`icon-btn row-actions-trigger${open ? ' open' : ''}`}
          aria-label={label}
          aria-haspopup="menu"
          aria-expanded={open}
          title={label}
          onClick={() => setOpen(o => !o)}
        >
          <Ellipsis size={17} />
        </button>
      )}
      {open && pos && createPortal(
        <div
          ref={menuRef}
          role="menu"
          className="dropdown row-actions-menu"
          style={{ position: 'fixed', top: pos.top ?? 'auto', bottom: pos.bottom ?? 'auto', left: pos.left, right: 'auto', width: MENU_WIDTH }}
          onClick={stop}
          onKeyDown={e => {
            if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
            e.preventDefault();
            const items = [...(menuRef.current?.querySelectorAll<HTMLButtonElement>('button:not(:disabled)') || [])];
            const i = items.indexOf(document.activeElement as HTMLButtonElement);
            items[(i + (e.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length]?.focus();
          }}
        >
          {menu.map((a, i) => {
            const Icon = a.icon;
            const firstDanger = a.danger && (i === 0 || !menu[i - 1].danger);
            return (
              <div key={a.label}>
                {firstDanger && i > 0 && <div className="dropdown-divider" />}
                <button
                  type="button"
                  role="menuitem"
                  className={`dropdown-item${a.danger ? ' danger' : ''}`}
                  disabled={a.disabled}
                  onClick={() => { close(); a.onClick(); }}
                >
                  {Icon && <Icon size={15} aria-hidden="true" />}{a.label}
                </button>
              </div>
            );
          })}
        </div>,
        document.body,
      )}
    </div>
  );
}
