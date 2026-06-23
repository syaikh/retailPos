import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'BestWorstBadges.svelte'), 'utf-8');
}

describe('BestWorstBadges.svelte source-structure guards', () => {
  const src = getSource();

  it('uses $props()', () => {
    expect(src).toContain('$props()');
  });

  it('imports TrendingUp, TrendingDown, ChevronDown from lucide-svelte', () => {
    expect(src).toContain("TrendingUp");
    expect(src).toContain("TrendingDown");
    expect(src).toContain("ChevronDown");
  });

  it('imports formatCurrencyShort and getPeriodLabel', () => {
    expect(src).toContain("formatCurrencyShort");
    expect(src).toContain("getPeriodLabel");
  });

  it('has Best badge section', () => {
    expect(src).toContain("Best {bestWorstHeading}:");
  });

  it('has Worst badge section', () => {
    expect(src).toContain("Worst {bestWorstHeading}:");
  });

  it('has data table toggle button', () => {
    expect(src).toContain("showDataTable");
    expect(src).toContain("Show");
    expect(src).toContain("Hide");
  });
});
