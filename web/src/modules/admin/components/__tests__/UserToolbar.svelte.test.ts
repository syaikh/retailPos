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

  it('imports Button, SearchBar, Dropdown from shared/ui', () => {
    expect(src).toContain("import { Button, SearchBar, Dropdown, FilterChipBar } from '$shared/ui'");
  });

  it('uses $bindable for searchQuery, filterRole, filterStatus', () => {
    expect(src).toContain('searchQuery = $bindable');
    expect(src).toContain('filterRole = $bindable');
    expect(src).toContain('filterStatus = $bindable');
  });

  it('uses $props', () => {
    expect(src).toContain('= $props()');
  });

  it('has role and status dropdowns via Dropdown component', () => {
    expect(src).toContain('<Dropdown');
    expect(src).toContain('placement="bottom-start"');
  });

  it('renders filter chips section', () => {
    expect(src).toContain('activeChips = $derived');
    expect(src).toContain('Clear all');
  });

  it('shows Add User button only when canCreate', () => {
    expect(src).toContain('{#if canCreate}');
  });
});
