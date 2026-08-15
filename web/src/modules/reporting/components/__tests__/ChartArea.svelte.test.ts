import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'ChartArea.svelte'), 'utf-8');
}

describe('ChartArea.svelte source-structure guards', () => {
  const src = getSource();

  it('uses $props()', () => {
    expect(src).toContain('$props()');
  });

  it('imports chart action from $shared/actions/chart', () => {
    expect(src).toContain("import { chart } from '$shared/actions/chart'");
  });

  it('has loading shimmer state', () => {
    expect(src).toContain('loading');
    expect(src).toContain('animate-shimmer');
  });

  it('has empty state message', () => {
    expect(src).toContain('labels.noDataAvailable');
  });

  it('has canvas with bind:this and use:chart', () => {
    expect(src).toContain('bind:this={chartCanvas}');
    expect(src).toContain('use:chart={chartConfig}');
  });

  it('has chartCanvas as bindable', () => {
    expect(src).toContain('chartCanvas = $bindable()');
  });
});
