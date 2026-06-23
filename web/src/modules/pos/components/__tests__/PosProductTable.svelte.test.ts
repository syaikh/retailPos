import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'PosProductTable.svelte'), 'utf-8');
}

describe('PosProductTable.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Badge, Button, Pagination, Skeleton from shared/ui', () => {
    expect(src).toContain("import { Badge, Button, Pagination, Skeleton } from '$shared/ui'");
  });

  it('imports Plus, Copy, Package from lucide-svelte', () => {
    expect(src).toContain("import { Plus, Copy, Package } from 'lucide-svelte'");
  });

  it('uses $bindable for showCopySuccess', () => {
    expect(src).toContain('showCopySuccess = $bindable(');
  });

  it('has event callbacks (onaddtocart, oncopy, onpagechange)', () => {
    expect(src).toContain('onaddtocart');
    expect(src).toContain('oncopy');
    expect(src).toContain('onpagechange');
  });

  it('handles loading state with Skeleton', () => {
    expect(src).toContain('{#if loading}');
    expect(src).toContain('<Skeleton');
  });

  it('handles empty state', () => {
    expect(src).toContain('No products found');
  });

  it('renders table headers', () => {
    expect(src).toContain('PRODUCT NAME');
    expect(src).toContain('Stock');
    expect(src).toContain('Price');
  });

  it('renders stock badges with variants', () => {
    expect(src).toContain('variant="destructive"');
    expect(src).toContain('variant="warning"');
    expect(src).toContain('variant="success"');
  });

  it('renders Pagination component', () => {
    expect(src).toContain('<Pagination');
  });
});
