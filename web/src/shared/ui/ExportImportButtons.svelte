<script lang="ts">
  import { Button } from '$shared/ui';
  import { Download, Upload } from 'lucide-svelte';

  let {
    canExportImport = false,
    onExport = (_format: 'csv' | 'xlsx') => {},
    onImport = () => {},
  }: {
    canExportImport?: boolean;
    onExport?: (format: 'csv' | 'xlsx') => void;
    onImport?: () => void;
  } = $props();

  let showExportDropdown = $state(false);

  function handleExport(format: 'csv' | 'xlsx') {
    showExportDropdown = false;
    onExport(format);
  }
</script>

{#if canExportImport}
  <div class="flex items-center gap-2">
    <div class="relative export-dropdown-container">
      <Button variant="secondary" class="shrink-0 px-3 text-xs" onclick={() => showExportDropdown = !showExportDropdown}>
        <Download size={14} />
        Export
      </Button>
      {#if showExportDropdown}
        <div class="absolute right-0 top-full mt-1 w-32 bg-surface border border-border rounded-lg shadow-lg z-50 overflow-hidden" onclick={() => showExportDropdown = false}>
          <button type="button" class="w-full px-4 py-2 text-left text-sm hover:bg-surface-hover transition-colors" onclick={() => handleExport('csv')}>Export CSV</button>
          <button type="button" class="w-full px-4 py-2 text-left text-sm hover:bg-surface-hover transition-colors" onclick={() => handleExport('xlsx')}>Export XLSX</button>
        </div>
      {/if}
    </div>
    <Button variant="secondary" class="shrink-0 px-3 text-xs" onclick={onImport}>
      <Upload size={14} />
      Import
    </Button>
  </div>
{/if}
