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

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels } from '$shared/i18n'");
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
    expect(src).toContain('{labels.noProductsFound}');
  });

  it('renders table headers', () => {
    expect(src).toContain('{labels.productName}');
    expect(src).toContain('{labels.stock}');
    expect(src).toContain('{labels.price}');
  });

  it('renders stock badges with variants', () => {
    expect(src).toContain('variant="danger"');
    expect(src).toContain('variant="warning"');
    expect(src).toContain('variant="success"');
  });

  it('renders Pagination component', () => {
    expect(src).toContain('<Pagination');
  });

  it('has selectedIndex bindable prop for keyboard navigation', () => {
    expect(src).toContain('selectedIndex = $bindable');
    expect(src).toContain('selectedIndex?: number');
  });

  it('has element bindable prop for scroll into view', () => {
    expect(src).toContain('element = $bindable');
    expect(src).toContain('element?: HTMLElement');
    expect(src).toContain('bind:this={element}');
  });

  it('highlights selected product row', () => {
    expect(src).toContain("idx === selectedIndex ? 'bg-primary/10");
    expect(src).toContain('onclick={() => selectedIndex = idx}');
    expect(src).toContain('ondblclick');
  });
});
