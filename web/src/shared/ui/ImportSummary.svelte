<script lang="ts">
  import { CheckCircle2, AlertCircle } from 'lucide-svelte';
  import type { ImportProgress } from '$shared/types/import-export';

  let {
    progress,
    error_report = '',
  }: {
    progress: ImportProgress | null;
    error_report?: string;
  } = $props();
</script>

{#if progress}
  <div class="space-y-4 text-center">
    <div class="w-14 h-14 rounded-full bg-emerald-400/10 flex items-center justify-center mx-auto">
      {#if progress.status === 'completed'}
        <CheckCircle2 size={28} class="text-emerald-400" />
      {:else if progress.status === 'failed'}
        <AlertCircle size={28} class="text-danger" />
      {:else}
        <CheckCircle2 size={28} class="text-emerald-400" />
      {/if}
    </div>

    <p class="text-text-primary font-semibold">
      {#if progress.status === 'completed'}
        Import Completed
      {:else if progress.status === 'failed'}
        Import Failed
      {:else}
        Import Cancelled
      {/if}
    </p>

    <div class="grid grid-cols-3 gap-3">
      <div class="p-3 bg-surface-subtle rounded-lg">
        <p class="text-lg font-bold text-emerald-400">{progress.inserted ?? 0}</p>
        <p class="text-xs text-text-muted">Inserted</p>
      </div>
      <div class="p-3 bg-surface-subtle rounded-lg">
        <p class="text-lg font-bold text-amber-400">{progress.updated ?? 0}</p>
        <p class="text-xs text-text-muted">Updated</p>
      </div>
      <div class="p-3 bg-surface-subtle rounded-lg">
        <p class="text-lg font-bold text-rose-400">{progress.errors}</p>
        <p class="text-xs text-text-muted">Errors</p>
      </div>
    </div>

    {#if progress.duration_ms > 0}
      <p class="text-xs text-text-muted">
        Duration: {progress.duration_ms >= 1000
          ? `${(progress.duration_ms / 1000).toFixed(1)}s`
          : `${progress.duration_ms}ms`}
      </p>
    {/if}

    {#if error_report}
      <div class="p-3 bg-danger-subtle/10 rounded-lg max-h-32 overflow-y-auto">
        <p class="text-xs text-danger flex items-start gap-1.5 py-0.5">
          <AlertCircle size={10} class="mt-0.5 shrink-0" />
          {error_report}
        </p>
      </div>
    {/if}
  </div>
{/if}
