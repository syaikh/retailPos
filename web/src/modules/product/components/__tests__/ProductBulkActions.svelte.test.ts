import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'ProductBulkActions.svelte'), 'utf-8');
}

describe('ProductBulkActions.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Button from shared/ui', () => {
    expect(src).toContain("import { Button } from '$shared/ui'");
  });

  it('uses $props', () => {
    expect(src).toContain('= $props()');
  });

  it('has event callbacks (onstatus, onclear)', () => {
    expect(src).toContain('onstatus');
    expect(src).toContain('onclear');
  });

  it('renders selected count', () => {
    expect(src).toContain('selectedCount} selected');
  });

  it('wraps content in conditional block', () => {
    expect(src).toContain('{#if selectedCount > 0}');
  });
});
