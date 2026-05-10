import { useEffect, useState } from 'react';
import { getTaxSummary, getCapitalGains } from '../api/tax';
import type { TaxSummary, CapitalGainsSummary } from '../types';
import { formatCurrency } from '../utils/format';
import { useAuth } from '../context/AuthContext';

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

  if (loading) return <div className="loading">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h1>Tax Center</h1>
      </div>

      <div style={{ display: 'flex', gap: 8, marginBottom: 24 }}>
        <button className={`btn ${tab === 'deductions' ? 'btn-primary' : 'btn-outline'}`} onClick={() => setTab('deductions')}>Tax Deductions</button>
        <button className={`btn ${tab === 'gains' ? 'btn-primary' : 'btn-outline'}`} onClick={() => setTab('gains')}>Capital Gains</button>
      </div>

      {tab === 'deductions' && summary && (
        <div>
          <div className="stat-card" style={{ marginBottom: 16, background: '#e8f0fe' }}>
            <div className="label">Total Deductions (FY {summary.fy || 'Current'})</div>
            <div className="value" style={{ color: '#1a73e8' }}>{formatCurrency(summary.total_deductions)}</div>
          </div>

          <div className="card" style={{ marginBottom: 16 }}>
            <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 12 }}>Section 80C <span className="text-muted" style={{ fontSize: 13, fontWeight: 400 }}>(Limit: {formatCurrency(summary.section_80c.limit)})</span></h3>
            <div className="payout-list">
              {summary.section_80c.epf_contribution > 0 && <div className="payout-item"><span>EPF Contribution</span><span>{formatCurrency(summary.section_80c.epf_contribution)}</span></div>}
              {summary.section_80c.ppf_contribution > 0 && <div className="payout-item"><span>PPF Contribution</span><span>{formatCurrency(summary.section_80c.ppf_contribution)}</span></div>}
              {summary.section_80c.elss_investment > 0 && <div className="payout-item"><span>ELSS Investment</span><span>{formatCurrency(summary.section_80c.elss_investment)}</span></div>}
              {summary.section_80c.home_loan_principal > 0 && <div className="payout-item"><span>Home Loan Principal</span><span>{formatCurrency(summary.section_80c.home_loan_principal)}</span></div>}
              {summary.section_80c.life_insurance > 0 && <div className="payout-item"><span>Life Insurance</span><span>{formatCurrency(summary.section_80c.life_insurance)}</span></div>}
              <div className="payout-item" style={{ fontWeight: 600, borderTop: '2px solid var(--border)', paddingTop: 8 }}>
                <span>Total / Deduction</span>
                <span>{formatCurrency(summary.section_80c.total)} / {formatCurrency(summary.section_80c.deduction)}</span>
              </div>
            </div>
          </div>

          <div className="card" style={{ marginBottom: 16 }}>
            <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 12 }}>Section 80CCD (NPS)</h3>
            <div className="payout-list">
              <div className="payout-item"><span>80CCD(1) - Employee</span><span>{formatCurrency(summary.section_80ccd.nps_80ccd1)}</span></div>
              <div className="payout-item"><span>80CCD(1B) - Additional ₹50K</span><span>{formatCurrency(summary.section_80ccd.nps_80ccd1b)}</span></div>
              <div className="payout-item"><span>80CCD(2) - Employer</span><span>{formatCurrency(summary.section_80ccd.nps_80ccd2)}</span></div>
            </div>
          </div>

          <div className="card" style={{ marginBottom: 16 }}>
            <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 12 }}>Section 24(b) <span className="text-muted" style={{ fontSize: 13, fontWeight: 400 }}>(Limit: {formatCurrency(summary.section_24b.limit)})</span></h3>
            <div className="payout-list">
              {summary.section_24b.home_loan_interest > 0 && <div className="payout-item"><span>Home Loan Interest</span><span>{formatCurrency(summary.section_24b.home_loan_interest)}</span></div>}
              {summary.section_24b.pre_emi_interest > 0 && <div className="payout-item"><span>Pre-EMI Interest</span><span>{formatCurrency(summary.section_24b.pre_emi_interest)}</span></div>}
              <div className="payout-item" style={{ fontWeight: 600, borderTop: '2px solid var(--border)', paddingTop: 8 }}>
                <span>Total / Deduction</span>
                <span>{formatCurrency(summary.section_24b.total)} / {formatCurrency(summary.section_24b.deduction)}</span>
              </div>
            </div>
          </div>

          {(summary.interest_income > 0 || summary.dividend_income > 0) && (
            <div className="card">
              <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 12 }}>Other Income</h3>
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
          <div className="card-grid" style={{ gridTemplateColumns: 'repeat(3, 1fr)', marginBottom: 16 }}>
            <div className="stat-card">
              <div className="label">LTCG</div>
              <div className="value">{formatCurrency(cg.ltcg)}</div>
              <div className="text-muted" style={{ fontSize: 12 }}>Tax: {formatCurrency(cg.ltcg_tax)} (Exempt: {formatCurrency(cg.ltcg_exemption)})</div>
            </div>
            <div className="stat-card">
              <div className="label">STCG</div>
              <div className="value">{formatCurrency(cg.stcg)}</div>
              <div className="text-muted" style={{ fontSize: 12 }}>Tax: {formatCurrency(cg.stcg_tax)}</div>
            </div>
            <div className="stat-card">
              <div className="label">Total Tax</div>
              <div className="value negative">{formatCurrency(cg.total_tax)}</div>
            </div>
          </div>

          {cg.harvesting_tips && cg.harvesting_tips.length > 0 && (
            <div className="card" style={{ marginBottom: 16, background: '#e8f5e9', border: '1px solid #4caf50' }}>
              <h3 style={{ fontSize: 15, fontWeight: 600, marginBottom: 8, color: '#2e7d32' }}>Tax Harvesting Tips</h3>
              <ul style={{ margin: 0, paddingLeft: 18, fontSize: 13 }}>
                {cg.harvesting_tips.map((tip, i) => (
                  <li key={i} style={{ marginBottom: 4 }}>{tip}</li>
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
                        {e.investment_name} <span className="text-muted" style={{ fontSize: 11 }}>({e.investment_type})</span>
                        {e.grandfathered && <span style={{ fontSize: 10, background: '#fff3e0', color: '#e65100', borderRadius: 4, padding: '1px 5px', marginLeft: 4 }}>GF</span>}
                      </td>
                      <td className="text-muted" style={{ fontSize: 12 }}>{e.buy_date}</td>
                      <td className="text-right">{formatCurrency(e.buy_amount)}</td>
                      <td className="text-right">{formatCurrency(e.sell_amount)}</td>
                      <td className={`text-right ${e.gain >= 0 ? 'text-success' : 'text-danger'}`}>{formatCurrency(e.gain)}</td>
                      <td className="text-muted" style={{ fontSize: 12 }}>{e.holding_days > 365 ? `${Math.floor(e.holding_days / 365)}y ${Math.floor((e.holding_days % 365) / 30)}m` : `${e.holding_days}d`}</td>
                      <td>{e.is_long_term ? 'LTCG' : 'STCG'} <span className="text-muted" style={{ fontSize: 11 }}>@{e.tax_rate}%</span></td>
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
