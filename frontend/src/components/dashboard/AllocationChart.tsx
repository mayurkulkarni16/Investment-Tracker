import { useMemo, useState } from 'react';
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts';
import { ArrowLeft, ChartPie } from 'lucide-react';
import type { MutualFund, Stock } from '../../types';
import { formatCompactINR, formatCurrency } from '../../utils/format';
import { ChartEmpty, ChartTooltip } from '../charts';
import { CHART_COLORS, truncate } from '../chartUtils';
import { categoryMeta, normalizeCategoryKey } from './categories';

interface Slice {
  name: string;
  value: number;
  color: string;
  drillKey?: string;
}

const DRILLABLE = new Set(['mutual_funds', 'stocks']);
const MAX_SLICES = 7;

function groupSmall(items: { name: string; value: number }[]): Slice[] {
  const sorted = [...items].sort((a, b) => b.value - a.value);
  const top = sorted.slice(0, MAX_SLICES);
  const rest = sorted.slice(MAX_SLICES).reduce((s, i) => s + i.value, 0);
  const slices = top.map((i, idx) => ({ ...i, color: CHART_COLORS[idx % CHART_COLORS.length] }));
  if (rest > 0) slices.push({ name: `Others (${sorted.length - MAX_SLICES})`, value: rest, color: '#94a3b8' });
  return slices;
}

export default function AllocationChart({ allocation, funds, stocks }: { allocation: Record<string, number>; funds: MutualFund[]; stocks: Stock[] }) {
  const [drill, setDrill] = useState<string | null>(null);

  const topLevel = useMemo<Slice[]>(() =>
    Object.entries(allocation || {})
      .filter(([, v]) => v > 0)
      .map(([k, v]) => {
        const meta = categoryMeta(k);
        const key = normalizeCategoryKey(k);
        return { name: meta.label, value: v, color: meta.color, drillKey: DRILLABLE.has(key) ? key : undefined };
      })
      .sort((a, b) => b.value - a.value),
  [allocation]);

  const drillSlices = useMemo<Slice[]>(() => {
    if (drill === 'mutual_funds') return groupSmall(funds.filter(f => f.current_value > 0).map(f => ({ name: f.fund_name, value: f.current_value })));
    if (drill === 'stocks') return groupSmall(stocks.filter(s => s.current_value > 0).map(s => ({ name: s.symbol, value: s.current_value })));
    return [];
  }, [drill, funds, stocks]);

  const slices = drill && drillSlices.length > 0 ? drillSlices : topLevel;
  const total = slices.reduce((s, x) => s + x.value, 0);
  const title = drill ? `${categoryMeta(drill).label} breakdown` : 'Asset allocation';

  return (
    <section className="card chart-card" aria-label={title}>
      <div className="card-head">
        <div>
          <h3>{title}</h3>
          <div className="card-subtitle">{drill ? 'By holding' : 'Click a category to see its holdings'}</div>
        </div>
        {drill && (
          <button className="btn btn-sm btn-outline" onClick={() => setDrill(null)}>
            <ArrowLeft size={14} /> Back
          </button>
        )}
      </div>

      {total <= 0 ? (
        <ChartEmpty icon={ChartPie} title="No investments yet" height={260}>Add holdings to see how your money is spread.</ChartEmpty>
      ) : (
        <>
          <div className="donut-wrap">
            <ResponsiveContainer width="100%" height={210}>
              <PieChart>
                <Pie
                  data={slices}
                  dataKey="value"
                  nameKey="name"
                  innerRadius={66}
                  outerRadius={96}
                  paddingAngle={slices.length > 1 ? 2 : 0}
                  cornerRadius={4}
                  stroke="none"
                  isAnimationActive={false}
                  onClick={(_, index) => { const k = slices[index]?.drillKey; if (!drill && k) setDrill(k); }}
                >
                  {slices.map(s => <Cell key={s.name} fill={s.color} style={{ cursor: !drill && s.drillKey ? 'pointer' : 'default', outline: 'none' }} />)}
                </Pie>
                <Tooltip content={<ChartTooltip />} />
              </PieChart>
            </ResponsiveContainer>
            <div className="donut-center">
              <span>{drill ? categoryMeta(drill).label : 'Total'}</span>
              <strong>{formatCompactINR(total)}</strong>
            </div>
          </div>
          <ul className="legend-list">
            {slices.map(s => {
              const clickable = !drill && !!s.drillKey;
              return (
                <li key={s.name}>
                  <button type="button" disabled={!clickable} onClick={() => clickable && setDrill(s.drillKey!)} title={`${s.name}: ${formatCurrency(s.value)}`}>
                    <i style={{ background: s.color }} />
                    <span className="legend-name">{truncate(s.name, 30)}</span>
                    <span className="legend-pct">{((s.value / total) * 100).toFixed(1)}%</span>
                    <span className="legend-amt">{formatCompactINR(s.value)}</span>
                  </button>
                </li>
              );
            })}
          </ul>
        </>
      )}
    </section>
  );
}
