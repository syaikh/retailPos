import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'UserToolbar.svelte'), 'utf-8');
}

describe('UserToolbar.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Button, SearchBar from shared/ui', () => {
    expect(src).toContain("import { Button, SearchBar } from '$shared/ui'");
  });

  it('uses $bindable for searchQuery, filterRole, filterStatus', () => {
    expect(src).toContain('searchQuery = $bindable');
    expect(src).toContain('filterRole = $bindable');
    expect(src).toContain('filterStatus = $bindable');
  });

  it('uses $props', () => {
    expect(src).toContain('= $props()');
  });

  it('has role and status dropdowns', () => {
    expect(src).toContain('showRoleDropdown = $state');
    expect(src).toContain('showStatusDropdown = $state');
    expect(src).toContain('role-filter-container');
    expect(src).toContain('status-filter-container');
  });

  it('renders filter chips section', () => {
    expect(src).toContain('activeChips = $derived');
    expect(src).toContain('Clear all');
  });

  it('shows Add User button only when canCreate', () => {
    expect(src).toContain('{#if canCreate}');
  });
});
