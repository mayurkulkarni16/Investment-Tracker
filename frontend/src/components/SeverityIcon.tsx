import { CircleCheck, Info, OctagonAlert, TriangleAlert, type LucideIcon } from 'lucide-react';

const SEVERITY: Record<string, { icon: LucideIcon; color: string }> = {
  critical: { icon: OctagonAlert, color: 'var(--danger-text)' },
  warning: { icon: TriangleAlert, color: 'var(--warning-text)' },
  info: { icon: Info, color: 'var(--primary-text)' },
  positive: { icon: CircleCheck, color: 'var(--success-text)' },
};

export default function SeverityIcon({ severity, size = 18 }: { severity: string; size?: number }) {
  const { icon: Icon, color } = SEVERITY[severity] || SEVERITY.info;
  return (
    <span className="severity-icon" style={{ color }}>
      <Icon size={size} strokeWidth={2} aria-hidden="true" />
    </span>
  );
}
