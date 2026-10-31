import { useEffect, useState } from 'react';
import { Menu, Moon, Search, Sun } from 'lucide-react';
import NotificationBell from './NotificationBell';
import UserMenu from './UserMenu';
import CommandPalette from './CommandPalette';

const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.userAgent);

export default function TopBar({ onOpenMenu }: { onOpenMenu: () => void }) {
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [dark, setDark] = useState(() => localStorage.getItem('theme') === 'dark');

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark);
    localStorage.setItem('theme', dark ? 'dark' : 'light');
  }, [dark]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setPaletteOpen(o => !o);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  return (
    <header className="topbar">
      <button className="icon-btn topbar-menu-btn" onClick={onOpenMenu} aria-label="Open menu">
        <Menu size={20} />
      </button>
      <button className="topbar-search" onClick={() => setPaletteOpen(true)} aria-label="Search pages">
        <Search size={16} aria-hidden="true" />
        <span>Search pages…</span>
        <kbd>{isMac ? '⌘' : 'Ctrl'} K</kbd>
      </button>
      <div className="topbar-actions">
        <button
          className="icon-btn"
          onClick={() => setDark(d => !d)}
          title={dark ? 'Switch to light mode' : 'Switch to dark mode'}
          aria-label={dark ? 'Switch to light mode' : 'Switch to dark mode'}
        >
          {dark ? <Sun size={19} strokeWidth={1.9} /> : <Moon size={19} strokeWidth={1.9} />}
        </button>
        <NotificationBell />
        <UserMenu />
      </div>
      {paletteOpen && <CommandPalette onClose={() => setPaletteOpen(false)} />}
    </header>
  );
}
