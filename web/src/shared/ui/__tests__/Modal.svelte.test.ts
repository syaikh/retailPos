import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'Modal.svelte'), 'utf-8');
}

describe('Modal.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Button from shared/ui', () => {
    expect(src).toContain("import { Button } from '$shared/ui'");
  });

  it('uses $bindable for open prop', () => {
    expect(src).toContain('open = $bindable(false)');
  });

  it('has role="dialog" for accessibility', () => {
    expect(src).toContain('role="dialog"');
  });

  it('has aria-modal attribute', () => {
    expect(src).toContain('aria-modal');
  });

  it('renders close button with X icon', () => {
    expect(src).toContain('<X');
  });
});