import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'UserDeleteModal.svelte'), 'utf-8');
}

describe('UserDeleteModal.svelte source-structure guards', () => {
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

  it('renders Modal with "Delete User" title', () => {
    expect(src).toContain('title={labels.deleteUser}');
  });

  it('shows username in confirmation', () => {
    expect(src).toContain('{username}');
  });

  it('imports Users icon from lucide-svelte', () => {
    expect(src).toContain("Users } from 'lucide-svelte'");
  });

  it('has subordinateCount prop', () => {
    expect(src).toContain('subordinateCount');
  });

  it('shows subordinate warning when count > 0', () => {
    expect(src).toContain('{#if subordinateCount > 0}');
  });
});
