import { useId } from 'react';

interface LogoMarkProps {
  size?: number;
}

export function LogoMark({ size = 32 }: LogoMarkProps) {
  // Each logo needs its own gradient id: if two share one and the first is inside a
  // hidden element (e.g. the sign-in brand panel on mobile), the visible one renders blank.
  const gradientId = `finboard-mark-${useId().replace(/:/g, '')}`;
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" aria-hidden="true">
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="32" y2="32" gradientUnits="userSpaceOnUse">
          <stop stopColor="#6366f1" />
          <stop offset="1" stopColor="#8b5cf6" />
        </linearGradient>
      </defs>
      <rect width="32" height="32" rx="9" fill={`url(#${gradientId})`} />
      <rect x="8" y="17" width="4" height="7" rx="1.5" fill="#fff" fillOpacity="0.65" />
      <rect x="14" y="12.5" width="4" height="11.5" rx="1.5" fill="#fff" fillOpacity="0.82" />
      <rect x="20" y="8" width="4" height="16" rx="1.5" fill="#fff" />
    </svg>
  );
}

export default function Logo({ size = 32 }: LogoMarkProps) {
  return (
    <span className="logo" aria-label="FinBoard">
      <LogoMark size={size} />
      <span className="brand-name" aria-hidden="true">Fin<span>Board</span></span>
    </span>
  );
}
