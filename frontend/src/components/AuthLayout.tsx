import type { ReactNode } from 'react';
import { Bell, ChartPie, Target } from 'lucide-react';
import Logo from './Logo';

const FEATURES = [
  { icon: ChartPie, title: 'One view of everything', text: 'Mutual funds, stocks, FDs, PF, NPS, gold, loans and cards — with real XIRR and allocation.' },
  { icon: Bell, title: 'Never miss a date', text: 'Reminders for EMIs, SIPs, FD maturities, coupons and insurance premiums.' },
  { icon: Target, title: 'Plan with confidence', text: 'Goals, FIRE planning, projections and tax-saving insights.' },
];

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="auth-page">
      <aside className="auth-showcase" aria-hidden="true">
        <Logo size={36} />
        <div className="auth-showcase-body">
          <h1>Every rupee,<br />one clear board.</h1>
          <p>Track your investments, loans and spending in one place.</p>
          <ul>
            {FEATURES.map(({ icon: Icon, title, text }) => (
              <li key={title}>
                <span><Icon size={18} /></span>
                <div><strong>{title}</strong>{text}</div>
              </li>
            ))}
          </ul>
        </div>
        <svg className="auth-showcase-chart" viewBox="0 0 400 120" preserveAspectRatio="none">
          <path d="M0 100 C 60 90, 90 70, 140 74 S 220 40, 270 46 S 350 14, 400 8 L 400 120 L 0 120 Z" />
          <path d="M0 100 C 60 90, 90 70, 140 74 S 220 40, 270 46 S 350 14, 400 8" />
        </svg>
      </aside>
      <main className="auth-main">
        <div className="auth-card">
          <div className="auth-brand"><Logo size={40} /></div>
          {children}
        </div>
      </main>
    </div>
  );
}
