import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'BulkStatusModal.svelte'), 'utf-8');
}

describe('BulkStatusModal.svelte source-structure guards', () => {
  const src = getSource();

  it('uses $bindable for open', () => {
    expect(src).toContain('$bindable(');
  });

  it('imports Button, Modal from shared/ui', () => {
    expect(src).toContain("import { Button, Modal } from '$shared/ui'");
  });

  it('renders Modal with "Bulk Update Status" title', () => {
    expect(src).toContain('title="Bulk Update Status"');
  });

  it('has Activate/Deactivate toggle buttons', () => {
    expect(src).toContain('Activate');
    expect(src).toContain('Deactivate');
  });
});
