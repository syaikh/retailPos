import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'AuditLogsFilterToolbar.svelte'), 'utf-8');
}

describe('AuditLogsFilterToolbar.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Button, Input, SearchBar, Dropdown from shared/ui', () => {
    expect(src).toContain("import { Button, Input, SearchBar, Dropdown, FilterChipBar } from '$shared/ui'");
  });

  it('uses $bindable for filter props', () => {
    expect(src).toContain('searchQuery = $bindable');
    expect(src).toContain('selectedAction = $bindable');
    expect(src).toContain('selectedResource = $bindable');
    expect(src).toContain('selectedDateRange = $bindable');
  });

  it('uses $props', () => {
    expect(src).toContain('= $props()');
  });

  it('has date picker and resource/action dropdowns via Dropdown', () => {
    expect(src).toContain('date-picker-container');
    expect(src).toContain('<Dropdown');
  });

  it('has export with CSV and Excel options via Dropdown', () => {
    expect(src).toContain('Export to CSV');
    expect(src).toContain('Export to Excel');
  });

  it('has filter chips section', () => {
    expect(src).toContain('activeFilters = $derived');
    expect(src).toContain('Clear all');
  });

  it('has date range presets', () => {
    expect(src).toContain('datePresets');
    expect(src).toContain('applyDatePreset');
  });

  it('has export utility functions', () => {
    expect(src).toContain('function buildExportUrl');
    expect(src).toContain('function downloadExport');
    expect(src).toContain('function exportToCsv');
    expect(src).toContain('function exportToExcel');
  });

  it('imports getAuthToken for export auth', () => {
    expect(src).toContain("import { getAuthToken } from '$modules/auth'");
  });

  it('imports toast for export notifications', () => {
    expect(src).toContain("import { toast } from '$shared/stores/toast.svelte'");
  });
});
