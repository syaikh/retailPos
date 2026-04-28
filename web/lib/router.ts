// Minimal client-side router for SPA (no SvelteKit needed)

let currentPath = window.location.pathname;
let listeners: (() => void)[] = [];

function notify() {
  listeners.forEach(fn => fn());
}

export function subscribe(fn: () => void): () => void {
  listeners.push(fn);
  return () => {
    listeners = listeners.filter(l => l !== fn);
  };
}

export function getPath(): string {
  return currentPath;
}

export function goto(path: string, replace?: boolean): void {
  if (replace) {
    window.history.replaceState({}, '', path);
  } else {
    window.history.pushState({}, '', path);
  }
  currentPath = path;
  notify();
}

// Handle browser back/forward
window.addEventListener('popstate', () => {
  currentPath = window.location.pathname;
  notify();
});

// Handle navigation from anchor clicks (optional)
window.addEventListener('click', (e) => {
  const target = e.target as HTMLElement;
  const anchor = target.closest('a[href]') as HTMLAnchorElement | null;
  if (!anchor) return;
  const href = anchor.getAttribute('href');
  if (!href || href.startsWith('http') || href.startsWith('#')) return;
  // Only intercept same-origin, no-target links
  e.preventDefault();
  goto(href);
});

export default {
  subscribe,
  getPath,
  goto
};
