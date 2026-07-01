<script lang="ts">
  import { Upload, FileText } from 'lucide-svelte';
  import { cn } from '$shared/utils/cn';

  let {
    file = $bindable(null),
    accept = '.csv,.xlsx',
    disabled = false,
  }: {
    file?: File | null;
    accept?: string;
    disabled?: boolean;
  } = $props();

  let dragOver = $state(false);
  let inputId = $state(`dropzone-${Math.random().toString(36).slice(2, 9)}`);

  function handleDrop(e: DragEvent) {
    dragOver = false;
    if (disabled) return;
    const f = e.dataTransfer?.files?.[0];
    if (f) file = f;
  }

  function handleSelect(e: Event) {
    const input = e.target as HTMLInputElement;
    const f = input.files?.[0];
    if (f) file = f;
    input.value = '';
  }

  function reset() {
    file = null;
  }
</script>

{#if file}
  <div class="flex items-center gap-3 p-3 bg-surface-subtle rounded-lg">
    <FileText size={20} class="text-primary-light shrink-0" />
    <div class="flex-1 min-w-0">
      <p class="text-sm font-medium text-text-primary truncate">{file.name}</p>
      <p class="text-xs text-text-muted">{(file.size / 1024).toFixed(1)} KB</p>
    </div>
    <button
      type="button"
      class="text-xs text-text-muted hover:text-text-primary transition-colors"
      onclick={reset}
      disabled={disabled}
    >
      Change
    </button>
  </div>
{:else}
  <div
    class={cn(
      'border-2 border-dashed rounded-xl p-8 text-center transition-all cursor-pointer',
      dragOver ? 'border-primary/50 bg-primary/5' : 'border-border hover:border-border-strong',
      disabled && 'opacity-40 pointer-events-none',
    )}
    ondragover={(e) => { e.preventDefault(); dragOver = true; }}
    ondragleave={() => dragOver = false}
    ondrop={(e) => { e.preventDefault(); handleDrop(e); }}
    onclick={() => document.getElementById(inputId)?.click()}
    role="button"
    tabindex="0"
    onkeydown={(e) => { if (e.key === 'Enter') document.getElementById(inputId)?.click(); }}
  >
    <Upload size={36} class="mx-auto mb-3 text-text-muted" />
    <p class="text-text-primary font-semibold">Drop file here or click to browse</p>
    <p class="text-text-muted text-sm mt-1">Supports CSV and XLSX files</p>
    <input
      id={inputId}
      type="file"
      {accept}
      class="hidden"
      onchange={handleSelect}
    />
  </div>
{/if}
