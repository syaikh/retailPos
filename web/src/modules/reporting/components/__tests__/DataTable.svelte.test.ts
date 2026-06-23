import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'DataTable.svelte'), 'utf-8');
}

describe('DataTable.svelte source-structure guards', () => {
  const src = getSource();

  it('uses $props()', () => {
    expect(src).toContain('$props()');
  });

  it('imports fly from svelte/transition', () => {
    expect(src).toContain("import { fly } from 'svelte/transition'");
  });

  it('imports TrendingUp and TrendingDown from lucide-svelte', () => {
    expect(src).toContain("TrendingUp");
    expect(src).toContain("TrendingDown");
  });

  it('has table headers with sort buttons', () => {
    expect(src).toContain("tablePeriodHeading");
    expect(src).toContain("Revenue (Rp)");
    expect(src).toContain("Prev Period (Rp)");
    expect(src).toContain("Change");
  });

  it('has sortColumn and sortAsc as bindable', () => {
    expect(src).toContain("sortColumn = $bindable(");
    expect(src).toContain("sortAsc = $bindable(");
  });

  it('has tfoot for total row', () => {
    expect(src).toContain("tfoot");
    expect(src).toContain("Total");
  });

  it('has ontogglesort callback', () => {
    expect(src).toContain("ontogglesort");
  });
});
