<script lang="ts">
  import { Button } from '$shared/ui';
  import { Loader2, Ban } from 'lucide-svelte';
  import type { ImportProgress } from '$shared/types/import-export';

  let {
    progress = $bindable(null as ImportProgress | null),
    onCancel = () => {},
    onClose = () => {},
  }: {
    progress?: ImportProgress | null;
    onCancel?: () => void;
    onClose?: () => void;
  } = $props();

  let cancelling = $state(false);

  function handleCancel() {
    cancelling = true;
    onCancel();
  }

  let statusInfo = $derived.by(() => {
    if (!progress) return { label: 'Starting...', spin: true };
    switch (progress.status) {
      case 'queued':
        return { label: 'Queued', spin: true };
      case 'parsing':
        return { label: 'Parsing file...', spin: true };
      case 'validating':
        return { label: 'Validating data...', spin: true };
      case 'preview_ready':
        return { label: 'Preview ready', spin: false };
      case 'confirmed':
        return { label: 'Importing...', spin: true };
      case 'importing':
        return { label: 'Importing...', spin: true };
      case 'completed':
        return { label: 'Completed', spin: false };
      case 'failed':
        return { label: 'Failed', spin: false };
      case 'cancelled':
        return { label: 'Cancelled', spin: false };
    }
  });

  let isImporting = $derived(progress?.status === 'importing' || progress?.status === 'confirmed');
</script>

{#if progress}
  <div class="space-y-4">
    <div class="text-center py-4">
      <div class="w-14 h-14 rounded-full bg-surface-subtle flex items-center justify-center mx-auto mb-3">
        <Loader2 size={28} class="text-primary-light {statusInfo.spin ? 'animate-spin' : ''}" />
      </div>
      <p class="text-text-primary font-semibold">{statusInfo.label}</p>

      {#if isImporting}
        <div class="mt-4 max-w-xs mx-auto">
          <div class="flex items-center justify-between text-xs text-text-muted mb-1.5">
            <span>{progress.processed} / {progress.total_rows} rows</span>
            <span>{progress.progress_pct}%</span>
          </div>
          <div class="h-2 bg-surface-default rounded-full overflow-hidden border border-border">
            <div
              class="h-full bg-primary-default rounded-full transition-all duration-500"
              style="width: {progress.progress_pct}%"
            ></div>
          </div>
        </div>
      {/if}
    </div>

    <div class="flex items-center justify-center gap-3 pt-2">
      {#if isImporting && !cancelling}
        <Button variant="danger" onclick={handleCancel}>
          <Ban size={14} /> Cancel
        </Button>
      {:else if cancelling}
        <Button variant="danger" disabled>
          <Loader2 size={14} class="animate-spin" /> Cancelling...
        </Button>
      {/if}
    </div>
  </div>
{/if}
