// Shared state: tracks whether any SelectSearch/Dropdown is currently open.
// Modal checks this before closing on Escape to avoid closing both.
let count = 0;
let listeners: Array<() => void> = [];

export function setDropdownOpen(open: boolean) {
  count += open ? 1 : -1;
  if (count < 0) count = 0;
  listeners.forEach((fn) => fn());
}

export function isAnyDropdownOpen(): boolean {
  return count > 0;
}
