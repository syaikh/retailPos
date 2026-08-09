import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'DeactivateCustomerModal.svelte'), 'utf-8');
}

describe('DeactivateCustomerModal.svelte source-structure guards', () => {
  const src = getSource();

  it('uses $bindable for open', () => {
    expect(src).toContain('$bindable(');
  });

  it('imports Button, Modal from shared/ui', () => {
    expect(src).toContain("import { Button, Modal } from '$shared/ui'");
  });

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels } from '$shared/i18n'");
  });

  it('renders Modal with "Deactivate Customer" title', () => {
    expect(src).toContain('title={labels.deactivateCustomer}');
  });

  it('shows target name in confirmation', () => {
    expect(src).toContain('{targetName}');
  });
});
