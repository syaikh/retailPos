import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'CustomerTable.svelte'), 'utf-8');
}

describe('CustomerTable.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Badge, Button, Skeleton from shared/ui', () => {
    expect(src).toContain("import { Badge, Button, Skeleton, SortableHeader } from '$shared/ui'");
  });

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels, t } from '$shared/i18n'");
  });

  it('uses $bindable for selectedIds, sortBy, sortDir', () => {
    expect(src).toContain('selectedIds = $bindable');
    expect(src).toContain('sortBy = $bindable');
    expect(src).toContain('sortDir = $bindable');
  });

  it('uses $derived for allSelected and someSelected', () => {
    expect(src).toContain('allSelected = $derived');
    expect(src).toContain('someSelected = $derived');
  });

  it('has event callbacks (onsort, onedit, ondeactivate)', () => {
    expect(src).toContain('onsort');
    expect(src).toContain('onedit');
    expect(src).toContain('ondeactivate');
  });

  it('renders table element', () => {
    expect(src).toContain('<table');
    expect(src).toContain('</table>');
  });

  it('handles loading state with Skeleton', () => {
    expect(src).toContain('{#if loading}');
  });

  it('handles empty state', () => {
    expect(src).toContain('labels.noCustomersYet');
  });

  it('does not have inline editing row', () => {
    expect(src).not.toContain('{#if editingId === c.id}');
  });
});
