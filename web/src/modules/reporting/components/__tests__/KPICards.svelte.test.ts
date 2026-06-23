import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'KPICards.svelte'), 'utf-8');
}

describe('KPICards.svelte source-structure guards', () => {
  const src = getSource();

  it('uses $props()', () => {
    expect(src).toContain('$props()');
  });

  it('imports Skeleton from $shared/ui', () => {
    expect(src).toContain("import { Skeleton } from '$shared/ui'");
  });

  it('imports TrendingUp and TrendingDown from lucide-svelte', () => {
    expect(src).toContain("TrendingUp");
    expect(src).toContain("TrendingDown");
  });

  it('imports formatCurrencyShort and formatLargeNumber', () => {
    expect(src).toContain("formatCurrencyShort");
    expect(src).toContain("formatLargeNumber");
  });

  it('has loading skeleton with 5 cards', () => {
    expect(src).toContain("length: 5");
    expect(src).toContain("Skeleton");
  });

  it('has Total Revenue card', () => {
    expect(src).toContain("Total Revenue");
  });

  it('has Total Orders card', () => {
    expect(src).toContain("Total Orders");
  });

  it('has Avg Order Value card', () => {
    expect(src).toContain("Avg Order Value");
  });

  it('has Peak Revenue / Avg Revenue per Day card (card4)', () => {
    expect(src).toContain("statCardLabels.card4");
  });

  it('has comparison / change card (card5)', () => {
    expect(src).toContain("statCardLabels.comparisonLabel");
  });
});
