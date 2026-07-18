import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'CustomerGroupsPage.svelte'), 'utf-8');
}

describe('CustomerGroupsPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports bulk service functions', () => {
    expect(src).toContain('bulkUpdateCustomerGroups');
    expect(src).toContain('bulkDeleteCustomerGroups');
  });

  it('imports ImportWizard from shared/ui', () => {
    expect(src).toContain("import ImportWizard from '$shared/ui/ImportWizard.svelte'");
  });

  it('has showImportWizard state', () => {
    expect(src).toContain('let showImportWizard = $state(false)');
  });

  it('has handleBulkActivate function', () => {
    expect(src).toContain('async function handleBulkActivate(ids: number[])');
  });

  it('has handleBulkDeactivate function', () => {
    expect(src).toContain('async function handleBulkDeactivate(ids: number[])');
  });

  it('has handleBulkDelete function', () => {
    expect(src).toContain('async function handleBulkDelete(ids: number[])');
  });

  it('has hasCustomersFilter state', () => {
    expect(src).toContain("let hasCustomersFilter = $state('all')");
  });

  it('passes has_customers filter to getCustomerGroups', () => {
    expect(src).toContain("filters.has_customers = hasCustomersFilter === 'yes'");
  });

  it('wires hasCustomersFilter to toolbar', () => {
    expect(src).toContain('bind:hasCustomersFilter');
  });

  it('wires bulk action callbacks to table', () => {
    expect(src).toContain('onbulkactivate={handleBulkActivate}');
    expect(src).toContain('onbulkdeactivate={handleBulkDeactivate}');
    expect(src).toContain('onbulkdelete={handleBulkDelete}');
  });

  it('has ImportWizard component wired', () => {
    expect(src).toContain('bind:open={showImportWizard}');
    expect(src).toContain('module="customer_groups"');
    expect(src).toContain('displayName="Customer Groups"');
  });

  it('has handleImport that opens import wizard', () => {
    expect(src).toContain('showImportWizard = true');
  });
});
