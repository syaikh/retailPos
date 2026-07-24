import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'CustomerToolbar.svelte'), 'utf-8');
}

describe('CustomerToolbar.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Button, SearchBar, BulkActionDropdown from shared/ui', () => {
    expect(src).toContain("import { Button, SearchBar, BulkActionDropdown, Dropdown } from '$shared/ui'");
  });

  it('uses $bindable for searchQuery and statusFilter', () => {
    expect(src).toContain('searchQuery = $bindable');
    expect(src).toContain('statusFilter = $bindable');
  });

  it('uses $props', () => {
    expect(src).toContain('= $props()');
  });

  it('has canCreate guard and event callbacks', () => {
    expect(src).toContain('canCreate');
    expect(src).toContain('onsearch');
    expect(src).toContain('onstatuschange');
    expect(src).toContain('oncreate');
  });

  it('renders SearchBar with bind:value', () => {
    expect(src).toContain('<SearchBar');
    expect(src).toContain('bind:value={searchQuery}');
  });

  it('shows Add Customer button only when canCreate', () => {
    expect(src).toContain('{#if canCreate}');
  });
});
