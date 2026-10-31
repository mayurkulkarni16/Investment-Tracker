import { useEffect } from 'react';

const FOCUSABLE_FIELD = 'input:not([type="hidden"]):not([disabled]):not([readonly]), select:not([disabled]), textarea:not([disabled])';

function topOverlay(): HTMLElement | undefined {
  const overlays = document.querySelectorAll<HTMLElement>('.modal-overlay');
  return overlays[overlays.length - 1];
}

// App-wide modal behaviour. Every page renders its own `.modal-overlay` whose
// click handler closes it, so Escape simply clicks the top-most overlay. New
// modals get focus on their first field (or first button for confirmations).
export function useModalKeyboard() {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      const overlay = topOverlay();
      if (!overlay) return;
      e.preventDefault();
      overlay.click();
    };

    const observer = new MutationObserver(mutations => {
      for (const m of mutations) {
        for (const node of m.addedNodes) {
          if (!(node instanceof HTMLElement)) continue;
          const overlay = node.matches('.modal-overlay') ? node : node.querySelector<HTMLElement>('.modal-overlay');
          if (!overlay || overlay.contains(document.activeElement)) continue;
          const target = overlay.querySelector<HTMLElement>(FOCUSABLE_FIELD) || overlay.querySelector<HTMLElement>('button');
          target?.focus({ preventScroll: true });
        }
      }
    });

    window.addEventListener('keydown', onKey);
    observer.observe(document.body, { childList: true, subtree: true });
    return () => {
      window.removeEventListener('keydown', onKey);
      observer.disconnect();
    };
  }, []);
}
