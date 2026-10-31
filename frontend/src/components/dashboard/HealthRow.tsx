import { Gauge, Layers, ReceiptText } from 'lucide-react';
import type { DashboardSummary } from '../../types';
import { formatCurrency } from '../../utils/format';
import InfoTooltip from '../InfoTooltip';

function riskLabel(score: number) {
  if (score <= 3) return { text: 'Conservative', tone: 'var(--success-text)' };
  if (score <= 6) return { text: 'Moderate', tone: 'var(--warning-text)' };
  return { text: 'Aggressive', tone: 'var(--danger-text)' };
}

export default function HealthRow({ data }: { data: DashboardSummary }) {
  const risk = data.risk_metrics?.risk_score ?? 0;
  const diversification = Math.max(0, Math.min(100, data.risk_metrics?.diversification ?? 0));
  const r = riskLabel(risk);
  const tax = data.tax_80c;
  const taxUsed = tax ? Math.min(tax.used, tax.limit) : 0;
  const taxPct = tax && tax.limit > 0 ? (taxUsed / tax.limit) * 100 : 0;

  return (
    <div className="kpi-grid">
      <section className="kpi-card" aria-label="Risk score">
        <div className="kpi-head">
          <span className="kpi-title"><Gauge size={15} aria-hidden="true" /> Risk score<InfoTooltip tip="1-10 scale based on equity/debt ratio and asset concentration. Higher = more risk" /></span>
          <span className="kpi-tag" style={{ color: r.tone }}>{r.text}</span>
        </div>
        <div className="kpi-value">{risk || '–'}<small>/ 10</small></div>
        <div className="risk-meter" role="meter" aria-valuemin={1} aria-valuemax={10} aria-valuenow={risk} aria-label="Risk score">
          <i style={{ left: `${Math.max(0, Math.min(100, ((risk - 1) / 9) * 100))}%` }} />
        </div>
        <div className="kpi-foot">
          {data.risk_metrics?.concentration_risk
            ? <>Concentration risk: <strong className="capitalize">{data.risk_metrics.concentration_risk}</strong></>
            : 'Based on your current mix of assets'}
        </div>
      </section>

      <section className="kpi-card" aria-label="Section 80C">
        <div className="kpi-head">
          <span className="kpi-title"><ReceiptText size={15} aria-hidden="true" /> Section 80C{tax ? ` · FY ${tax.fy}` : ''}<InfoTooltip tip="Deductions used this financial year: EPF, ELSS (incl. SIPs) and home loan principal. Applies under the old tax regime only." /></span>
          {tax && <span className="kpi-tag">{taxPct.toFixed(0)}%</span>}
        </div>
        {tax ? (
          <>
            <div className="kpi-value">{formatCurrency(taxUsed)}<small>of {formatCurrency(tax.limit)}</small></div>
            <div className="meter" role="meter" aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(taxPct)} aria-label="80C limit used">
              <span style={{ width: `${taxPct}%`, background: taxPct >= 100 ? 'var(--success)' : 'var(--primary)' }} />
            </div>
            <div className="kpi-foot">
              {tax.used >= tax.limit ? 'Limit fully used' : <><strong>{formatCurrency(tax.limit - tax.used)}</strong> room left</>}
              {tax.elss > 0 && ` · ELSS ${formatCurrency(tax.elss)}`}
            </div>
          </>
        ) : (
          <>
            <div className="kpi-value">{formatCurrency(data.elss_tax_saving)}</div>
            <div className="kpi-foot">ELSS tax saving</div>
          </>
        )}
      </section>

      <section className="kpi-card" aria-label="Diversification">
        <div className="kpi-head">
          <span className="kpi-title"><Layers size={15} aria-hidden="true" /> Diversification<InfoTooltip tip="Percentage of your portfolio spread across different asset classes. Higher = more diversified" /></span>
          <span className="kpi-tag">{diversification >= 70 ? 'Well spread' : diversification >= 40 ? 'Fair' : 'Concentrated'}</span>
        </div>
        <div className="kpi-value">{diversification.toFixed(0)}<small>%</small></div>
        <div className="meter" role="meter" aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(diversification)} aria-label="Diversification">
          <span style={{ width: `${diversification}%`, background: diversification >= 70 ? 'var(--success)' : diversification >= 40 ? 'var(--warning)' : 'var(--danger)' }} />
        </div>
        <div className="kpi-foot">Spread across {Object.values(data.asset_allocation || {}).filter(v => v > 0).length} asset classes</div>
      </section>
    </div>
  );
}
