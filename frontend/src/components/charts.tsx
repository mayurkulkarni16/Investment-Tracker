import type { ReactNode } from 'react';
import type { LucideIcon } from 'lucide-react';
import { formatCurrency } from '../utils/format';

interface TooltipEntry {
  name?: string | number;
  value?: number | string | (number | string)[];
  color?: string;
  dataKey?: string | number;
  payload?: Record<string, unknown>;
}

interface ChartTooltipProps {
  active?: boolean;
  payload?: ReadonlyArray<TooltipEntry>;
  label?: string | number;
  valueFormatter?: (value: number) => string;
  labelFormatter?: (label: string | number | undefined, payload: ReadonlyArray<TooltipEntry>) => ReactNode;
  totalLabel?: string;
}

// Theme-aware replacement for the default Recharts tooltip. Pass as an element:
// <Tooltip content={<ChartTooltip valueFormatter={...} />} />
export function ChartTooltip({ active, payload, label, valueFormatter = formatCurrency, labelFormatter, totalLabel }: ChartTooltipProps) {
  if (!active || !payload || payload.length === 0) return null;
  const rows = payload.filter(p => typeof p.value === 'number');
  const total = rows.reduce((sum, p) => sum + (p.value as number), 0);
  const heading = labelFormatter ? labelFormatter(label, payload) : label;
  return (
    <div className="chart-tooltip">
      {heading !== undefined && heading !== '' && <div className="chart-tooltip-label">{heading}</div>}
      {rows.map(p => (
        <div key={String(p.dataKey ?? p.name)} className="chart-tooltip-row">
          <i style={{ background: p.color || (p.payload?.fill as string | undefined) }} />
          <span>{p.name}</span>
          <b>{valueFormatter(p.value as number)}</b>
        </div>
      ))}
      {totalLabel && rows.length > 1 && (
        <div className="chart-tooltip-row chart-tooltip-total">
          <span>{totalLabel}</span>
          <b>{valueFormatter(total)}</b>
        </div>
      )}
    </div>
  );
}

interface SegmentedControlProps<T extends string> {
  options: { value: T; label: string }[];
  value: T;
  onChange: (value: T) => void;
  ariaLabel: string;
}

export function SegmentedControl<T extends string>({ options, value, onChange, ariaLabel }: SegmentedControlProps<T>) {
  return (
    <div className="segmented" role="group" aria-label={ariaLabel}>
      {options.map(o => (
        <button key={o.value} type="button" className={o.value === value ? 'active' : ''} aria-pressed={o.value === value} onClick={() => onChange(o.value)}>
          {o.label}
        </button>
      ))}
    </div>
  );
}

export function ChartEmpty({ icon: Icon, title, children, height }: { icon: LucideIcon; title: string; children?: ReactNode; height?: number }) {
  return (
    <div className="chart-empty" style={height ? { minHeight: height } : undefined}>
      <Icon size={28} strokeWidth={1.6} aria-hidden="true" />
      <strong>{title}</strong>
      {children && <span>{children}</span>}
    </div>
  );
}
