<script lang="ts">
  import { Button } from '$shared/ui';
  import { Loader2, XCircle, CheckCircle2, AlertCircle, Ban } from 'lucide-svelte';
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

  let isDone = $derived(
    progress?.status === 'completed' || progress?.status === 'failed' || progress?.status === 'cancelled'
  );

  let statusInfo = $derived.by(() => {
    if (!progress) return { icon: Loader2, color: 'text-primary-light', label: 'Starting...', spin: true };
    switch (progress.status) {
      case 'queued':
        return { icon: Loader2, color: 'text-text-muted', label: 'Queued', spin: true };
      case 'parsing':
        return { icon: Loader2, color: 'text-primary-light', label: 'Parsing file...', spin: true };
      case 'validating':
        return { icon: Loader2, color: 'text-primary-light', label: 'Validating data...', spin: true };
      case 'preview_ready':
        return { icon: CheckCircle2, color: 'text-emerald-400', label: 'Preview ready', spin: false };
      case 'confirmed':
        return { icon: Loader2, color: 'text-amber-400', label: 'Importing...', spin: true };
      case 'importing':
        return { icon: Loader2, color: 'text-amber-400', label: 'Importing...', spin: true };
      case 'completed':
        return { icon: CheckCircle2, color: 'text-emerald-400', label: 'Completed', spin: false };
      case 'failed':
        return { icon: AlertCircle, color: 'text-danger', label: 'Failed', spin: false };
      case 'cancelled':
        return { icon: Ban, color: 'text-text-muted', label: 'Cancelled', spin: false };
    }
  });
</script>

{#if progress}
  <div class="space-y-4">
    <div class="text-center py-4">
      <div class="w-14 h-14 rounded-full bg-surface-subtle flex items-center justify-center mx-auto mb-3">
        <statusInfo.icon size={28} class="{statusInfo.color} {statusInfo.spin ? 'animate-spin' : ''}" />
      </div>
      <p class="text-text-primary font-semibold">{statusInfo.label}</p>

      {#if progress.status === 'importing' || progress.status === 'confirmed'}
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

    {#if isDone}
      <div class="grid grid-cols-3 gap-3 text-center">
        <div class="p-2 bg-surface-subtle rounded-lg">
          <p class="text-lg font-bold text-emerald-400">{progress.inserted}</p>
          <p class="text-xs text-text-muted">Inserted</p>
        </div>
        <div class="p-2 bg-surface-subtle rounded-lg">
          <p class="text-lg font-bold text-amber-400">{progress.updated}</p>
          <p class="text-xs text-text-muted">Updated</p>
        </div>
        <div class="p-2 bg-surface-subtle rounded-lg">
          <p class="text-lg font-bold text-rose-400">{progress.errors}</p>
          <p class="text-xs text-text-muted">Errors</p>
        </div>
      </div>

      {#if progress.error_report}
        <div class="p-3 bg-danger-subtle/10 rounded-lg max-h-32 overflow-y-auto">
          <p class="text-xs text-danger flex items-start gap-1.5 py-0.5">
            <AlertCircle size={10} class="mt-0.5 shrink-0" />
            {progress.error_report}
          </p>
        </div>
      {/if}
    {/if}

    <div class="flex items-center justify-center gap-3 pt-2">
      {#if isDone}
        <Button variant="primary" onclick={onClose}>Close</Button>
      {:else if progress.status !== 'cancelled' && !cancelling}
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
