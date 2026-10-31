import type { ReactNode } from 'react';
import { ChevronDown, ChevronsUpDown, ChevronUp } from 'lucide-react';

export type SortDir = 'asc' | 'desc';

interface SortableThProps {
  col: string;
  sortCol: string;
  sortDir: SortDir;
  onSort: (col: string) => void;
  children: ReactNode;
  align?: 'left' | 'right';
}

export function SortableTh({ col, sortCol, sortDir, onSort, children, align = 'left' }: SortableThProps) {
  const active = sortCol === col;
  const Icon = !active ? ChevronsUpDown : sortDir === 'asc' ? ChevronUp : ChevronDown;
  return (
    <th className={align === 'right' ? 'num' : undefined} aria-sort={active ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'}>
      <button type="button" className={`th-sort${active ? ' active' : ''}`} onClick={() => onSort(col)}>
        {children}
        <Icon size={13} aria-hidden="true" />
      </button>
    </th>
  );
}

interface TabsProps<T extends string> {
  tabs: { value: T; label: string; count?: number }[];
  value: T;
  onChange: (value: T) => void;
  ariaLabel: string;
}

export function Tabs<T extends string>({ tabs, value, onChange, ariaLabel }: TabsProps<T>) {
  return (
    <div className="tabs" role="tablist" aria-label={ariaLabel}>
      {tabs.map(t => (
        <button
          key={t.value}
          type="button"
          role="tab"
          aria-selected={t.value === value}
          className={`tab${t.value === value ? ' active' : ''}`}
          onClick={() => onChange(t.value)}
        >
          {t.label}
          {t.count !== undefined && <span className="tab-count">{t.count}</span>}
        </button>
      ))}
    </div>
  );
}

interface FilterChipsProps<T extends string> {
  options: { value: T; label: string; count?: number }[];
  value: T;
  onChange: (value: T) => void;
  ariaLabel: string;
}

export function FilterChips<T extends string>({ options, value, onChange, ariaLabel }: FilterChipsProps<T>) {
  return (
    <div className="chips" role="group" aria-label={ariaLabel}>
      {options.map(o => (
        <button key={o.value} type="button" className={`chip${o.value === value ? ' active' : ''}`} aria-pressed={o.value === value} onClick={() => onChange(o.value)}>
          {o.label}
          {o.count !== undefined && <span className="chip-count">{o.count}</span>}
        </button>
      ))}
    </div>
  );
}

// Placeholder layout shown while a page loads: header, stat cards and a table.
export function PageSkeleton({ label = 'Loading' }: { label?: string }) {
  return (
    <div className="page-skeleton" aria-busy="true" aria-live="polite">
      <span className="sr-only">{label}…</span>
      <div className="page-header">
        <div className="skeleton" style={{ width: 220, height: 30 }} />
        <div className="skeleton" style={{ width: 130, height: 36 }} />
      </div>
      <div className="card-grid">
        {[0, 1, 2, 3].map(i => <div key={i} className="skeleton" style={{ height: 92, borderRadius: 12 }} />)}
      </div>
      <div className="card p-0" style={{ overflow: 'hidden' }}>
        <div className="skeleton" style={{ height: 42, borderRadius: 0 }} />
        {[0, 1, 2, 3, 4].map(i => (
          <div className="flex gap-4" key={i} style={{ padding: '14px 16px', borderTop: '1px solid var(--border)' }}>
            <div className="skeleton" style={{ flex: 3, height: 16 }} />
            <div className="skeleton flex-1" style={{ height: 16 }} />
            <div className="skeleton flex-1" style={{ height: 16 }} />
            <div className="skeleton flex-1" style={{ height: 16 }} />
          </div>
        ))}
      </div>
    </div>
  );
}
