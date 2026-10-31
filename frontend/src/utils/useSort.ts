import { useMemo, useState } from 'react';
import type { SortDir } from '../components/ui';

type Accessor<T> = (row: T) => string | number | undefined | null;

export function useSort<T>(rows: T[], accessors: Record<string, Accessor<T>>, initial?: { col: string; dir: SortDir }) {
  const [sortCol, setSortCol] = useState(initial?.col ?? '');
  const [sortDir, setSortDir] = useState<SortDir>(initial?.dir ?? 'desc');

  const onSort = (col: string) => {
    if (col === sortCol) {
      setSortDir(d => (d === 'asc' ? 'desc' : 'asc'));
      return;
    }
    const sample = rows.map(r => accessors[col]?.(r)).find(v => v !== undefined && v !== null);
    setSortCol(col);
    // Money and percentages are most useful largest-first; names read best A→Z.
    setSortDir(typeof sample === 'number' ? 'desc' : 'asc');
  };

  const sorted = useMemo(() => {
    const get = accessors[sortCol];
    if (!get) return rows;
    const dir = sortDir === 'asc' ? 1 : -1;
    return [...rows].sort((a, b) => {
      const av = get(a), bv = get(b);
      if (typeof av === 'number' || typeof bv === 'number') return (Number(av ?? 0) - Number(bv ?? 0)) * dir;
      return String(av ?? '').localeCompare(String(bv ?? ''), undefined, { sensitivity: 'base' }) * dir;
    });
    // accessors is a stable module-level or memoised map in callers
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, sortCol, sortDir]);

  return { sorted, sortCol, sortDir, onSort };
}
