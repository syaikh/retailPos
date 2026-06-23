import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'StatCard.svelte'), 'utf-8');
}

describe('StatCard.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Badge and Skeleton from shared/ui', () => {
    expect(src).toContain("import { Badge, Skeleton } from '$shared/ui'");
  });

  it('uses $props for prop destructuring', () => {
    expect(src).toContain('$props()');
  });

  it('uses $derived for trend calculations', () => {
    expect(src).toContain('trendUp');
    expect(src).toContain('trendDown');
  });

  it('has loading state handling with Skeleton', () => {
    expect(src).toContain('{#if loading}');
    expect(src).toContain('<Skeleton');
  });

  it('renders Icon prop', () => {
    expect(src).toContain('Icon');
  });
});