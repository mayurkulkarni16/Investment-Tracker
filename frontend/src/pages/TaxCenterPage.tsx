import { useEffect, useState } from 'react';
import { getTaxSummary, getCapitalGains } from '../api/tax';
import type { TaxSummary, CapitalGainsSummary } from '../types';
import { formatCurrency } from '../utils/format';
import { useAuth } from '../context/AuthContext';
import { Tabs, PageSkeleton } from '../components/ui';

export default function TaxCenterPage() {
  const { viewAsUserId } = useAuth();
  const [summary, setSummary] = useState<TaxSummary | null>(null);
  const [cg, setCG] = useState<CapitalGainsSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<'deductions' | 'gains'>('deductions');

  useEffect(() => {
    Promise.all([
      getTaxSummary().then(r => setSummary(r.data)),
      getCapitalGains().then(r => setCG(r.data)),
    ]).catch(() => {}).finally(() => setLoading(false));
  }, [viewAsUserId]);

  if (loading) return <PageSkeleton label="Loading" />;

  return (
    <div>
      <div className="page-header">
        <h1>Tax Center</h1>
      </div>

      <Tabs
        ariaLabel="Tax views"
        value={tab}
        onChange={setTab}
        tabs={[{ value: 'deductions', label: 'Tax deductions' }, { value: 'gains', label: 'Capital gains' }]}
      />

      {tab === 'deductions' && summary && (
        <div>
          <div className="stat-card mb-4" style={{ background: 'var(--primary-soft)' }}>
            <div className="label">Total Deductions (FY {summary.fy || 'Current'})</div>
            <div className="value" style={{ color: '#4f46e5' }}>{formatCurrency(summary.total_deductions)}</div>
          </div>

          <div className="card mb-4">
            <h3 className="text-lg fw-semibold mb-3">Section 80C <span className="text-muted text-md fw-normal">(Limit: {formatCurrency(summary.section_80c.limit)})</span></h3>
            <div className="payout-list">
              {summary.section_80c.epf_contribution > 0 && <div className="payout-item"><span>EPF Contribution</span><span>{formatCurrency(summary.section_80c.epf_contribution)}</span></div>}
              {summary.section_80c.ppf_contribution > 0 && <div className="payout-item"><span>PPF Contribution</span><span>{formatCurrency(summary.section_80c.ppf_contribution)}</span></div>}
              {summary.section_80c.elss_investment > 0 && <div className="payout-item"><span>ELSS Investment</span><span>{formatCurrency(summary.section_80c.elss_investment)}</span></div>}
              {summary.section_80c.home_loan_principal > 0 && <div className="payout-item"><span>Home Loan Principal</span><span>{formatCurrency(summary.section_80c.home_loan_principal)}</span></div>}
              {summary.section_80c.life_insurance > 0 && <div className="payout-item"><span>Life Insurance</span><span>{formatCurrency(summary.section_80c.life_insurance)}</span></div>}
              <div className="payout-item fw-semibold" style={{ borderTop: '2px solid var(--border)', paddingTop: 8 }}>
                <span>Total / Deduction</span>
                <span>{formatCurrency(summary.section_80c.total)} / {formatCurrency(summary.section_80c.deduction)}</span>
              </div>
            </div>
          </div>

          <div className="card mb-4">
            <h3 className="text-lg fw-semibold mb-3">Section 80CCD (NPS)</h3>
            <div className="payout-list">
              <div className="payout-item"><span>80CCD(1) - Employee</span><span>{formatCurrency(summary.section_80ccd.nps_80ccd1)}</span></div>
              <div className="payout-item"><span>80CCD(1B) - Additional ₹50K</span><span>{formatCurrency(summary.section_80ccd.nps_80ccd1b)}</span></div>
              <div className="payout-item"><span>80CCD(2) - Employer</span><span>{formatCurrency(summary.section_80ccd.nps_80ccd2)}</span></div>
            </div>
          </div>

          <div className="card mb-4">
            <h3 className="text-lg fw-semibold mb-3">Section 24(b) <span className="text-muted text-md fw-normal">(Limit: {formatCurrency(summary.section_24b.limit)})</span></h3>
            <div className="payout-list">
              {summary.section_24b.home_loan_interest > 0 && <div className="payout-item"><span>Home Loan Interest</span><span>{formatCurrency(summary.section_24b.home_loan_interest)}</span></div>}
              {summary.section_24b.pre_emi_interest > 0 && <div className="payout-item"><span>Pre-EMI Interest</span><span>{formatCurrency(summary.section_24b.pre_emi_interest)}</span></div>}
              <div className="payout-item fw-semibold" style={{ borderTop: '2px solid var(--border)', paddingTop: 8 }}>
                <span>Total / Deduction</span>
                <span>{formatCurrency(summary.section_24b.total)} / {formatCurrency(summary.section_24b.deduction)}</span>
              </div>
            </div>
          </div>

          {(summary.interest_income > 0 || summary.dividend_income > 0) && (
            <div className="card">
              <h3 className="text-lg fw-semibold mb-3">Other Income</h3>
              <div className="payout-list">
                {summary.interest_income > 0 && <div className="payout-item"><span>Interest Income</span><span>{formatCurrency(summary.interest_income)}</span></div>}
                {summary.dividend_income > 0 && <div className="payout-item"><span>Dividend Income</span><span>{formatCurrency(summary.dividend_income)}</span></div>}
              </div>
            </div>
          )}
        </div>
      )}

      {tab === 'gains' && cg && (
        <div>
          <div className="card-grid stat-grid mb-4">
            <div className="stat-card">
              <div className="label">LTCG</div>
              <div className="value">{formatCurrency(cg.ltcg)}</div>
              <div className="text-muted text-sm">Tax: {formatCurrency(cg.ltcg_tax)} (Exempt: {formatCurrency(cg.ltcg_exemption)})</div>
            </div>
            <div className="stat-card">
              <div className="label">STCG</div>
              <div className="value">{formatCurrency(cg.stcg)}</div>
              <div className="text-muted text-sm">Tax: {formatCurrency(cg.stcg_tax)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Total Tax</div>
              <div className="value negative">{formatCurrency(cg.total_tax)}</div>
            </div>
          </div>

          {cg.harvesting_tips && cg.harvesting_tips.length > 0 && (
            <div className="card mb-4" style={{ background: 'var(--success-soft)', border: '1px solid var(--success)' }}>
              <h3 className="fw-semibold mb-2" style={{ fontSize: 15, color: 'var(--success-text)' }}>Tax Harvesting Tips</h3>
              <ul className="m-0 text-md" style={{ paddingLeft: 18 }}>
                {cg.harvesting_tips.map((tip, i) => (
                  <li className="mb-1" key={i}>{tip}</li>
                ))}
              </ul>
            </div>
          )}

          {cg.entries?.length > 0 && (
            <div className="table-container">
              <table>
                <thead>
                  <tr>
                    <th>Investment</th>
                    <th>Buy Date</th>
                    <th className="text-right">Buy</th>
                    <th className="text-right">Sell</th>
                    <th className="text-right">Gain</th>
                    <th>Holding</th>
                    <th>Type</th>
                    <th className="text-right">Tax</th>
                  </tr>
                </thead>
                <tbody>
                  {cg.entries.map((e, i) => (
                    <tr key={i}>
                      <td>
                        {e.investment_name} <span className="text-muted text-xs">({e.investment_type})</span>
                        {e.grandfathered && <span className="text-warning ml-1" style={{ fontSize: 10, background: 'var(--warning-soft)', borderRadius: 4, padding: '1px 5px' }}>GF</span>}
                      </td>
                      <td className="text-muted text-sm">{e.buy_date}</td>
                      <td className="text-right">{formatCurrency(e.buy_amount)}</td>
                      <td className="text-right">{formatCurrency(e.sell_amount)}</td>
                      <td className={`text-right ${e.gain >= 0 ? 'text-success' : 'text-danger'}`}>{formatCurrency(e.gain)}</td>
                      <td className="text-muted text-sm">{e.holding_days > 365 ? `${Math.floor(e.holding_days / 365)}y ${Math.floor((e.holding_days % 365) / 30)}m` : `${e.holding_days}d`}</td>
                      <td>{e.is_long_term ? 'LTCG' : 'STCG'} <span className="text-muted text-xs">@{e.tax_rate}%</span></td>
                      <td className="text-right">{formatCurrency(e.tax_liability)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
