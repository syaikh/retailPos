import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'CreateCustomerModal.svelte'), 'utf-8');
}

describe('CreateCustomerModal.svelte source-structure guards', () => {
  const src = getSource();

  it('uses $bindable for form fields', () => {
    expect(src).toContain('$bindable(');
  });

  it('imports Button, Input, Modal from shared/ui', () => {
    expect(src).toContain("import { Button, Input, Modal } from '$shared/ui'");
  });

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels } from '$shared/i18n'");
  });

  it('renders Modal with "Add Customer" title', () => {
    expect(src).toContain('title={labels.addCustomer}');
  });

  it('has form fields for name, phone, email, note', () => {
    expect(src).toContain('id="customer-name"');
    expect(src).toContain('id="customer-phone"');
    expect(src).toContain('id="customer-email"');
    expect(src).toContain('id="customer-note"');
  });
});
