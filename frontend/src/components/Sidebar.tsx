import { NavLink } from 'react-router-dom';
import { PanelLeftClose, PanelLeftOpen } from 'lucide-react';
import Logo from './Logo';
import { NAV_SECTIONS } from './navigation';

interface SidebarProps {
  collapsed: boolean;
  mobileOpen: boolean;
  onCloseMobile: () => void;
  onToggleCollapsed: () => void;
}

export default function Sidebar({ collapsed, mobileOpen, onCloseMobile, onToggleCollapsed }: SidebarProps) {
  return (
    <>
      {mobileOpen && <div className="sidebar-overlay" onClick={onCloseMobile} />}
      <aside className={`sidebar${mobileOpen ? ' open' : ''}`}>
        <div className="sidebar-brand">
          <NavLink to="/" className="logo" onClick={onCloseMobile} aria-label="FinBoard home">
            <Logo />
          </NavLink>
        </div>
        <nav className="sidebar-nav" aria-label="Main navigation" onClick={onCloseMobile}>
          {NAV_SECTIONS.map(section => (
            <div key={section.title} className="nav-section">
              <div className="nav-section-title">{section.title}</div>
              {section.items.map(({ to, label, icon: Icon }) => (
                <NavLink key={to} to={to} end={to === '/'} aria-label={label} title={collapsed ? label : undefined}>
                  <Icon size={18} strokeWidth={1.8} aria-hidden="true" />
                  <span className="nav-label">{label}</span>
                </NavLink>
              ))}
            </div>
          ))}
        </nav>
        <div className="sidebar-footer">
          <button
            className="sidebar-toggle"
            onClick={onToggleCollapsed}
            title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {collapsed ? <PanelLeftOpen size={18} /> : <PanelLeftClose size={18} />}
            <span className="nav-label">Collapse</span>
          </button>
        </div>
      </aside>
    </>
  );
}
