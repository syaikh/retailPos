import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'BulkDeleteModal.svelte'), 'utf-8');
}

describe('BulkDeleteModal.svelte source-structure guards', () => {
  const src = getSource();

  it('uses $bindable for open', () => {
    expect(src).toContain('$bindable(');
  });

  it('imports Button, Modal from shared/ui', () => {
    expect(src).toContain("import { Button, Modal } from '$shared/ui'");
  });

  it('renders Modal with "Delete Customers" title', () => {
    expect(src).toContain('title="Delete Customers"');
  });

  it('shows delete count', () => {
    expect(src).toContain('{count}');
  });
});
