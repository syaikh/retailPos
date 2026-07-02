export const DEBOUNCE_DELAY = 300;

/** @type {ReturnType<typeof setTimeout>|null} */
let timer = null;

/**
 * @template T
 * @param {(...args: T[]) => void} fn
 * @param {number} [delay]
 * @returns {(...args: T[]) => void}
 */
export function debounce(fn, delay = DEBOUNCE_DELAY) {
  return (...args) => {
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => fn(...args), delay);
  };
}

/**
 * @template T
 * @param {(...args: T[]) => void} fn
 * @param {number} [delay]
 * @returns {(...args: T[]) => void}
 */
export function debounceWithTimer(fn, delay = DEBOUNCE_DELAY) {
  /** @type {ReturnType<typeof setTimeout>|null} */
  let localTimer = null;
  return (...args) => {
    if (localTimer) clearTimeout(localTimer);
    localTimer = setTimeout(() => fn(...args), delay);
  };
}

/**
 * @param {ReturnType<typeof setTimeout>|null} timerId
 */
export function cleanupTimer(timerId) {
  if (timerId) clearTimeout(timerId);
}
