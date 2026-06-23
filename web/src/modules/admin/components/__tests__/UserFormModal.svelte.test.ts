import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'UserFormModal.svelte'), 'utf-8');
}

describe('UserFormModal.svelte source-structure guards', () => {
  const src = getSource();

  it('uses $bindable for open and form', () => {
    expect(src).toContain('$bindable(');
  });

  it('imports Button, Input, Modal from shared/ui', () => {
    expect(src).toContain("import { Button, Input, Modal } from '$shared/ui'");
  });

  it('renders Modal with dynamic title', () => {
    expect(src).toContain('title={modalMode ===');
  });

  it('has form fields for username, email, password', () => {
    expect(src).toContain('id="usr-username"');
    expect(src).toContain('id="usr-email"');
    expect(src).toContain('id="usr-password"');
  });

  it('has role dropdown', () => {
    expect(src).toContain('form-role-dropdown-container');
  });
});
