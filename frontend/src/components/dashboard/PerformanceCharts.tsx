import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { Bar, BarChart, CartesianGrid, Cell, Legend, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { ChartColumn, ChartPie } from 'lucide-react';
import type { DashboardSummary, InvestmentSummary, MutualFund } from '../../types';
import { formatCompactINR, formatPercent } from '../../utils/format';
import { ChartEmpty, ChartTooltip, SegmentedControl } from '../charts';
import { truncate } from '../chartUtils';

export function ContributionChart({ data }: { data: DashboardSummary }) {
  const gainOf = (s?: InvestmentSummary) => (s ? s.current_value + (s.income_received || 0) - s.total_invested : 0);
  const rows = [
    { name: 'Mutual Funds', s: data.mutual_fund_summary },
    { name: 'Stocks', s: data.stock_summary },
    { name: 'FDs', s: data.fixed_deposit_summary },
    { name: 'Bonds', s: data.corporate_bond_summary },
    { name: 'PF', s: data.provident_fund_summary },
    { name: 'NPS', s: data.nps_summary },
    { name: 'Rentals', s: data.vehicle_rental_summary },
  ]
    .map(({ name, s }) => ({ name, invested: s?.total_invested || 0, gain: gainOf(s) }))
    .filter(d => d.invested > 0 || d.gain !== 0);

  return (
    <section className="card chart-card" aria-label="Invested vs gains">
      <div className="card-head">
        <div>
          <h3>Invested vs gains</h3>
          <div className="card-subtitle">Gains include interest, maturities and payouts already received</div>
        </div>
      </div>
      {rows.length === 0 ? (
        <ChartEmpty icon={ChartColumn} title="Nothing invested yet" height={280} />
      ) : (
        <ResponsiveContainer width="100%" height={280}>
          <BarChart data={rows} margin={{ top: 8, right: 8, bottom: 0, left: 0 }} barCategoryGap="28%">
            <CartesianGrid vertical={false} />
            <XAxis dataKey="name" fontSize={11} interval={0} />
            <YAxis tickFormatter={v => formatCompactINR(Number(v))} width={64} fontSize={11} />
            <Tooltip content={<ChartTooltip totalLabel="Total value" />} />
            <Legend iconType="circle" iconSize={8} />
            <ReferenceLine y={0} className="zero-line" />
            <Bar dataKey="invested" name="Invested" stackId="a" fill="#6366f1" />
            <Bar dataKey="gain" name="Gains" stackId="a" fill="#10b981" radius={[4, 4, 0, 0]}>
              {rows.map(r => <Cell key={r.name} fill={r.gain >= 0 ? '#10b981' : '#ef4444'} />)}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      )}
    </section>
  );
}

type FundView = 'value' | 'returns';
const MAX_FUNDS = 8;

// Single-line axis label; Recharts' default tick wraps long names onto two lines.
function FundTick({ x, y, payload }: { x?: number | string; y?: number | string; payload?: { value: string } }) {
  return (
    <text x={Number(x)} y={Number(y)} dy={4} textAnchor="end" fontSize={11.5} className="recharts-cartesian-axis-tick-value">
      {payload?.value}
    </text>
  );
}

export function FundPerformanceChart({ funds }: { funds: MutualFund[] }) {
  const [view, setView] = useState<FundView>('value');
  const invested = funds.filter(f => f.total_invested > 0);

  const rows = useMemo(() => {
    const mapped = invested.map(f => ({
      name: truncate(f.fund_name, 24),
      fullName: f.fund_name,
      invested: f.total_invested,
      current: f.current_value,
      returnPct: f.gain_loss_percent,
    }));
    return view === 'value'
      ? mapped.sort((a, b) => b.current - a.current).slice(0, MAX_FUNDS)
      : mapped.sort((a, b) => b.returnPct - a.returnPct).slice(0, MAX_FUNDS);
  }, [invested, view]);

  const fullName = (_: unknown, payload: ReadonlyArray<{ payload?: Record<string, unknown> }>) => String(payload[0]?.payload?.fullName ?? '');
  const height = Math.max(240, rows.length * (view === 'value' ? 46 : 36) + 40);

  return (
    <section className="card chart-card" aria-label="Mutual fund performance">
      <div className="card-head">
        <div>
          <h3>Mutual fund performance</h3>
          <div className="card-subtitle">
            {invested.length > MAX_FUNDS ? <>Top {MAX_FUNDS} of {invested.length} funds · <Link to="/mutual-funds">View all</Link></> : `${invested.length} fund${invested.length === 1 ? '' : 's'}`}
          </div>
        </div>
        <SegmentedControl
          ariaLabel="Chart view"
          value={view}
          onChange={setView}
          options={[{ value: 'value', label: 'Value' }, { value: 'returns', label: 'Returns' }]}
        />
      </div>

      {rows.length === 0 ? (
        <ChartEmpty icon={ChartPie} title="No mutual funds yet" height={240}>
          <Link to="/mutual-funds">Add a fund</Link> or import your CAS statement.
        </ChartEmpty>
      ) : (
        <ResponsiveContainer width="100%" height={height}>
          {view === 'value' ? (
            <BarChart data={rows} layout="vertical" margin={{ top: 0, right: 12, bottom: 0, left: 0 }} barGap={2} barCategoryGap="22%">
              <CartesianGrid horizontal={false} />
              <XAxis type="number" tickFormatter={v => formatCompactINR(Number(v))} fontSize={11} />
              <YAxis type="category" dataKey="name" width={170} tick={<FundTick />} />
              <Tooltip content={<ChartTooltip labelFormatter={fullName} />} />
              <Legend iconType="circle" iconSize={8} />
              <Bar dataKey="invested" name="Invested" fill="#c7d2fe" radius={[0, 4, 4, 0]} />
              <Bar dataKey="current" name="Current value" fill="#6366f1" radius={[0, 4, 4, 0]} />
            </BarChart>
          ) : (
            <BarChart data={rows} layout="vertical" margin={{ top: 0, right: 12, bottom: 0, left: 0 }} barCategoryGap="28%">
              <CartesianGrid horizontal={false} />
              <XAxis type="number" tickFormatter={v => `${Number(v).toFixed(0)}%`} fontSize={11} />
              <YAxis type="category" dataKey="name" width={170} tick={<FundTick />} />
              <Tooltip content={<ChartTooltip labelFormatter={fullName} valueFormatter={formatPercent} />} />
              <ReferenceLine x={0} className="zero-line" />
              <Bar dataKey="returnPct" name="Absolute return" radius={[0, 4, 4, 0]}>
                {rows.map(r => <Cell key={r.fullName} fill={r.returnPct >= 0 ? '#10b981' : '#ef4444'} />)}
              </Bar>
            </BarChart>
          )}
        </ResponsiveContainer>
      )}
    </section>
  );
}
