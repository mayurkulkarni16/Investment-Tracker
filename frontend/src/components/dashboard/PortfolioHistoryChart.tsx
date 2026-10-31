import { useMemo, useState } from 'react';
import { Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { ChartLine } from 'lucide-react';
import type { NetWorthSnapshot } from '../../types';
import { formatCompactINR } from '../../utils/format';
import { ChartEmpty, ChartTooltip, SegmentedControl } from '../charts';
import { categoryMeta } from './categories';
import { monthLabel, SNAPSHOT_INVESTMENT_KEYS, snapshotValue } from './history';

type Range = '6m' | '1y' | '3y' | 'all';
const RANGE_MONTHS: Record<Range, number> = { '6m': 6, '1y': 12, '3y': 36, all: Infinity };

export default function PortfolioHistoryChart({ history }: { history: NetWorthSnapshot[] }) {
  const [range, setRange] = useState<Range>('1y');

  const { rows, keys } = useMemo(() => {
    const months = RANGE_MONTHS[range];
    const slice = Number.isFinite(months) ? history.slice(-months) : history;
    const activeKeys = SNAPSHOT_INVESTMENT_KEYS.filter(k => slice.some(s => snapshotValue(s, k) > 0));
    return {
      keys: activeKeys,
      rows: slice.map(s => {
        const row: Record<string, number | string> = { month: s.month };
        for (const k of activeKeys) row[k] = snapshotValue(s, k);
        return row;
      }),
    };
  }, [history, range]);

  const first = rows[0];
  const last = rows[rows.length - 1];
  const sum = (r?: Record<string, number | string>) => (r ? keys.reduce((t, k) => t + Number(r[k] || 0), 0) : 0);
  const growth = rows.length >= 2 && sum(first) > 0 ? ((sum(last) - sum(first)) / sum(first)) * 100 : null;

  return (
    <section className="card chart-card" aria-label="Portfolio value over time">
      <div className="card-head">
        <div>
          <h3>Portfolio value over time</h3>
          <div className="card-subtitle">
            Monthly snapshots of your investments
            {growth !== null && <> · <span className={growth >= 0 ? 'positive' : 'negative'}>{growth >= 0 ? '+' : ''}{growth.toFixed(1)}%</span> in this period</>}
          </div>
        </div>
        <SegmentedControl
          ariaLabel="Time range"
          value={range}
          onChange={setRange}
          options={[{ value: '6m', label: '6M' }, { value: '1y', label: '1Y' }, { value: '3y', label: '3Y' }, { value: 'all', label: 'All' }]}
        />
      </div>

      {rows.length < 2 || keys.length === 0 ? (
        <ChartEmpty icon={ChartLine} title="Your history is building up" height={280}>
          A snapshot of your portfolio is saved automatically each month. The trend appears once there are two months of data.
        </ChartEmpty>
      ) : (
        <div className="chart-fill">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={rows} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
            <defs>
              {keys.map(k => (
                <linearGradient key={k} id={`ph-${k}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={categoryMeta(k).color} stopOpacity={0.45} />
                  <stop offset="100%" stopColor={categoryMeta(k).color} stopOpacity={0.08} />
                </linearGradient>
              ))}
            </defs>
            <CartesianGrid vertical={false} />
            <XAxis dataKey="month" tickFormatter={m => monthLabel(String(m), true)} minTickGap={24} fontSize={11} />
            <YAxis tickFormatter={v => formatCompactINR(Number(v))} width={64} fontSize={11} />
            <Tooltip content={<ChartTooltip labelFormatter={l => monthLabel(String(l))} totalLabel="Total" />} />
            {keys.map(k => (
              <Area
                key={k}
                type="monotone"
                dataKey={k}
                name={categoryMeta(k).label}
                stackId="portfolio"
                stroke={categoryMeta(k).color}
                strokeWidth={1.5}
                fill={`url(#ph-${k})`}
              />
            ))}
          </AreaChart>
        </ResponsiveContainer>
        </div>
      )}
    </section>
  );
}
