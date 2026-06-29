<script lang="ts">
  import { Button, Dropdown } from '$shared/ui';
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

  function handleExport(format: 'csv' | 'xlsx') {
    onExport(format);
  }
</script>

{#if canExportImport}
  <div class="flex items-center gap-2">
    <Dropdown items={[
      { label: 'Export CSV', onclick: () => handleExport('csv') },
      { label: 'Export XLSX', onclick: () => handleExport('xlsx') },
    ]}>
      {#snippet trigger({ toggle })}
        <Button variant="secondary" class="shrink-0 px-3 text-xs" onclick={toggle}>
          <Download size={14} />
          Export
        </Button>
      {/snippet}
    </Dropdown>
    <Button variant="secondary" class="shrink-0 px-3 text-xs" onclick={onImport}>
      <Upload size={14} />
      Import
    </Button>
  </div>
{/if}
