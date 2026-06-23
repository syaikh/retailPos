import type { StockThreshold } from '../types';

let warningThreshold = $state(10);
let criticalThreshold = $state(5);
let initialized = false;

export function useInventoryStore() {
  function setThresholds(t: StockThreshold) {
    warningThreshold = t.warning;
    criticalThreshold = t.critical;
  }

  if (!initialized) {
    initialized = true;
  }

  return {
    get warningThreshold() { return warningThreshold; },
    get criticalThreshold() { return criticalThreshold; },
    setThresholds,
  };
}
