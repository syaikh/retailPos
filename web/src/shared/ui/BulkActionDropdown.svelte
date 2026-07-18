<script lang="ts">
  import { Button, Dropdown } from '$shared/ui';
  import { Download, Upload, FileDown, FileSpreadsheet, FileText, History, ChevronDown } from 'lucide-svelte';
  import { downloadExport, downloadTemplate } from '$shared/services/import-export-service';
  import { goto } from '$app/router';
  import type { ExportFormat } from '$shared/types/import-export';

  const historyRoutes: Record<string, string> = {
    categories: '/categories/import-history',
    brands: '/brands/import-history',
    uoms: '/units-of-measure/import-history',
    customers: '/customers/import-history',
    products: '/products/import-history',
    customer_groups: '/customer-groups/import-history',
  };

  let {
    module = '',
    canExport = false,
    canImport = false,
    canTemplate = true,
    onImport = () => {},
  }: {
    module?: string;
    canExport?: boolean;
    canImport?: boolean;
    canTemplate?: boolean;
    onImport?: () => void;
  } = $props();

  function handleHistory() {
    const route = historyRoutes[module];
    if (route) goto(route);
  }
  function handleExport(format: ExportFormat) {
    downloadExport(module, format);
  }

  function handleTemplate() {
    downloadTemplate(module);
  }
</script>

{#if canExport || canImport}
  <Dropdown
    items={[
      { label: 'Export CSV', icon: FileText, onclick: () => handleExport('csv') },
      { label: 'Export XLSX', icon: FileSpreadsheet, onclick: () => handleExport('xlsx') },
      ...(canTemplate
        ? [{ divider: true }, { label: 'Download Template', icon: FileDown, onclick: () => handleTemplate() }]
        : []),
      ...(canImport
        ? [{ divider: true }, { label: 'Import Data', icon: Upload, onclick: () => onImport() }]
        : []),
      ...(canImport
        ? [{ divider: true }, { label: 'Import History', icon: History, onclick: () => handleHistory() }]
        : []),
    ]}
    placement="bottom-end"
  >
    {#snippet trigger({ toggle })}
      <Button variant="secondary" class="shrink-0 px-3" onclick={toggle}>
        <Download size={14} />
        Bulk Actions
        <ChevronDown size={14} />
      </Button>
    {/snippet}
  </Dropdown>
{/if}
