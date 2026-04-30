// Client-side router implementation

let listeners: Array<(path: string) => void> = [];

export function getPath(): string {
  return window.location.pathname || '/';
}

export function goto(path: string): void {
  if (path === getPath()) {
    // If same path, still notify listeners (for auth guard re-evaluation)
    listeners.forEach(listener => listener(path));
    return;
  }

  window.history.pushState({}, '', path);
  listeners.forEach(listener => listener(path));
}

export function subscribe(listener: (path: string) => void): () => void {
  listeners.push(listener);

  // Return unsubscribe function
  return () => {
    listeners = listeners.filter(l => l !== listener);
  };
}

// Handle browser back/forward buttons
window.addEventListener('popstate', () => {
  listeners.forEach(listener => listener(getPath()));
});
