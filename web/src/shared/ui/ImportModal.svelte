<script lang="ts">
  import { Button, Modal } from '$shared/ui';
  import { Upload, FileText, Loader2, AlertCircle, CheckCircle2, XCircle } from 'lucide-svelte';

  interface ImportResult {
    inserted: number;
    updated: number;
    errors?: string[];
  }

  let {
    show = $bindable(false),
    title = 'Import Data',
    accept = '.csv',
    templateHeaders = [] as string[],
    onImport = async (_file: File): Promise<ImportResult> => ({ inserted: 0, updated: 0 }),
  }: {
    show?: boolean;
    title?: string;
    accept?: string;
    templateHeaders?: string[];
    onImport?: (file: File) => Promise<ImportResult>;
  } = $props();

  let file = $state<File | null>(null);
  let previewRows = $state<string[][]>([]);
  let importing = $state(false);
  let result = $state<ImportResult | null>(null);
  let dragOver = $state(false);
  let error = $state('');

  function handleFileDrop(e: DragEvent) {
    dragOver = false;
    const f = e.dataTransfer?.files?.[0];
    if (f) loadFile(f);
  }

  function handleFileSelect(e: Event) {
    const input = e.target as HTMLInputElement;
    const f = input.files?.[0];
    if (f) loadFile(f);
  }

  function loadFile(f: File) {
    error = '';
    result = null;
    if (!f.name.endsWith('.csv')) {
      error = 'Only CSV files are supported';
      return;
    }
    file = f;
    const reader = new FileReader();
    reader.onload = () => {
      const text = reader.result as string;
      const lines = text.split('\n').filter(l => l.trim());
      if (lines.length < 2) {
        error = 'File must have a header row and at least one data row';
        return;
      }
      previewRows = lines.slice(1, 11).map(l => l.split(',').map(c => c.trim()));
    };
    reader.readAsText(f);
  }

  async function handleImport() {
    if (!file) return;
    importing = true;
    error = '';
    try {
      result = await onImport(file);
      if (result.errors?.length) {
        error = result.errors.slice(0, 5).join(', ');
        if (result.errors.length > 5) {
          error += ` and ${result.errors.length - 5} more errors`;
        }
      }
    } catch (err: any) {
      error = err?.response?.data?.error || err?.message || 'Import failed';
    } finally {
      importing = false;
    }
  }

  function reset() {
    file = null;
    previewRows = [];
    result = null;
    error = '';
  }

  function handleClose() {
    reset();
    show = false;
  }

  function downloadTemplate() {
    if (templateHeaders.length === 0) return;
    const csv = templateHeaders.join(',') + '\n';
    const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${title.toLowerCase().replace(/\s+/g, '-')}-template.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }
</script>

<Modal bind:open={show} {title} size="lg">
  {#if result}
    <div class="text-center py-6">
      <div class="w-16 h-16 rounded-full bg-primary-subtle flex items-center justify-center mx-auto mb-4">
        <CheckCircle2 size={32} class="text-primary-light" />
      </div>
      <p class="text-lg font-semibold text-text-primary mb-2">Import Complete</p>
      <div class="flex items-center justify-center gap-6 text-sm">
        <div class="text-center">
          <p class="text-2xl font-bold text-primary-light">{result.inserted}</p>
          <p class="text-text-muted">Inserted</p>
        </div>
        <div class="text-center">
          <p class="text-2xl font-bold text-amber-400">{result.updated}</p>
          <p class="text-text-muted">Updated</p>
        </div>
      </div>
      {#if result.errors?.length}
        <div class="mt-4 p-3 bg-danger-subtle/10 rounded-lg text-left max-h-32 overflow-y-auto">
          {#each result.errors as err}
            <p class="text-xs text-danger flex items-start gap-1.5 py-0.5">
              <AlertCircle size={12} class="mt-0.5 shrink-0" />
              {err}
            </p>
          {/each}
        </div>
      {/if}
    </div>
    {#snippet footer()}
      <Button variant="secondary" onclick={handleClose}>Close</Button>
    {/snippet}

  {:else}
    <div class="space-y-4">
      {#if !file}
        <div
          class="border-2 border-dashed rounded-xl p-8 text-center transition-colors cursor-pointer {dragOver ? 'border-primary/50 bg-primary/5' : 'border-border hover:border-border-strong'}"
          ondragover={(e) => { e.preventDefault(); dragOver = true; }}
          ondragleave={() => dragOver = false}
          ondrop={(e) => { e.preventDefault(); handleFileDrop(e); }}
          onclick={() => document.getElementById('import-file-input')?.click()}
          role="button"
          tabindex="0"
          onkeydown={(e) => { if (e.key === 'Enter') document.getElementById('import-file-input')?.click(); }}
        >
          <Upload size={40} class="mx-auto mb-3 text-text-muted" />
          <p class="text-text-primary font-semibold">Drop CSV file here or click to browse</p>
          <p class="text-text-muted text-sm mt-1">Only .csv files are accepted</p>
          <input
            id="import-file-input"
            type="file"
            accept={accept}
            class="hidden"
            onchange={handleFileSelect}
          />
        </div>
      {:else}
        <div class="flex items-center gap-3 p-3 bg-surface-subtle rounded-lg">
          <FileText size={20} class="text-primary-light shrink-0" />
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium text-text-primary truncate">{file.name}</p>
            <p class="text-xs text-text-muted">{(file.size / 1024).toFixed(1)} KB</p>
          </div>
          <button type="button" class="text-xs text-text-muted hover:text-text-primary transition-colors" onclick={() => { file = null; previewRows = []; }}>Change</button>
        </div>

        {#if previewRows.length > 0}
          <div>
            <p class="text-sm font-medium text-text-secondary mb-2">Preview (first {previewRows.length} rows)</p>
            <div class="overflow-x-auto border border-border rounded-lg max-h-48 overflow-y-auto">
              <table class="w-full text-xs">
                <thead class="bg-muted/50 sticky top-0">
                  <tr>
                    {#each templateHeaders as header}
                      <th class="px-3 py-2 text-left font-semibold text-text-secondary whitespace-nowrap">{header}</th>
                    {/each}
                  </tr>
                </thead>
                <tbody>
                  {#each previewRows as row}
                    <tr class="border-t border-border">
                      {#each row as cell}
                        <td class="px-3 py-1.5 text-text-primary truncate max-w-40">{cell}</td>
                      {/each}
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>
        {/if}
      {/if}

      {#if templateHeaders.length > 0 && !file}
        <div class="text-center">
          <button type="button" class="text-xs text-primary-light hover:underline" onclick={downloadTemplate}>
            Download Template CSV
          </button>
        </div>
      {/if}

      {#if error}
        <div class="flex items-start gap-2 p-3 bg-danger-subtle/10 rounded-lg">
          <XCircle size={16} class="text-danger shrink-0 mt-0.5" />
          <p class="text-sm text-danger">{error}</p>
        </div>
      {/if}
    </div>

    {#snippet footer()}
      <Button variant="secondary" onclick={handleClose}>Cancel</Button>
      <Button variant="primary" disabled={!file || importing} onclick={handleImport}>
        {#if importing}
          <Loader2 size={16} class="animate-spin" /> Importing...
        {:else}
          Import
        {/if}
      </Button>
    {/snippet}
  {/if}
</Modal>
