<script lang="ts">
  import { Button, Dropdown } from '$shared/ui';
  import { Download, Upload, FileDown, FileSpreadsheet, FileText } from 'lucide-svelte';
  import { downloadExport, downloadTemplate } from '$shared/services/import-export-service';
  import type { ExportFormat } from '$shared/types/import-export';

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

  function handleExport(format: ExportFormat) {
    downloadExport(module, format);
  }

  function handleTemplate() {
    downloadTemplate(module);
  }
</script>

{#if canExport || canImport}
  <div class="flex items-center gap-2">
    {#if canExport}
      <Dropdown items={[
        { label: 'Export CSV', icon: FileText, onclick: () => handleExport('csv') },
        { label: 'Export XLSX', icon: FileSpreadsheet, onclick: () => handleExport('xlsx') },
        ...(canTemplate
          ? [{ separator: 'Template' } as const,
             { label: 'Download Template', icon: FileDown, onclick: () => handleTemplate() }]
          : []),
      ]}>
        {#snippet trigger({ toggle })}
          <Button variant="secondary" class="shrink-0 px-3" onclick={toggle}>
            <Download size={14} />
            Export
          </Button>
        {/snippet}
      </Dropdown>
    {/if}

    {#if canImport}
      <Button variant="secondary" class="shrink-0 px-3" onclick={onImport}>
        <Upload size={14} />
        Import
      </Button>
    {/if}
  </div>
{/if}
