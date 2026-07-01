<script lang="ts">
  import { Button, Modal } from '$shared/ui';
  import { Loader2, AlertCircle, CheckCircle2, Ban, History, Clock } from 'lucide-svelte';
  import { getHistory } from '$shared/services/import-export-service';
  import type { ImportProgress } from '$shared/types/import-export';

  let {
    open = $bindable(false),
    module = '',
    displayName = '',
  }: {
    open?: boolean;
    module?: string;
    displayName?: string;
  } = $props();

  let jobs = $state<ImportProgress[]>([]);
  let loading = $state(false);
  let error = $state('');

  $effect(() => {
    if (open && module) {
      loading = true;
      error = '';
      getHistory(module)
        .then((data) => {
          jobs = data;
        })
        .catch((err) => {
          error = err?.response?.data?.error || err?.message || 'Failed to load history';
        })
        .finally(() => {
          loading = false;
        });
    }
  });

  function statusIcon(status: string) {
    if (status === 'completed') return CheckCircle2;
    if (status === 'failed') return AlertCircle;
    if (status === 'cancelled') return Ban;
    return Loader2;
  }

  function statusColor(status: string) {
    if (status === 'completed') return 'text-emerald-400';
    if (status === 'failed') return 'text-danger';
    if (status === 'cancelled') return 'text-text-muted';
    return 'text-primary-light';
  }

  function formatDate(iso: string) {
    const d = new Date(iso);
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
</script>

<Modal bind:open={open} title={displayName ? `Import History — ${displayName}` : 'Import History'} size="lg">
  <div class="space-y-4 min-h-[150px]">
    {#if loading}
      <div class="flex items-center justify-center py-8">
        <Loader2 size={24} class="animate-spin text-primary-light" />
      </div>
    {:else if error}
      <div class="flex items-start gap-2 p-3 bg-danger-subtle/10 rounded-lg">
        <AlertCircle size={16} class="text-danger shrink-0 mt-0.5" />
        <p class="text-sm text-danger">{error}</p>
      </div>
    {:else if jobs.length === 0}
      <div class="text-center py-8 text-text-muted">
        <History size={32} class="mx-auto mb-2 opacity-50" />
        <p class="text-sm">No import history yet</p>
      </div>
    {:else}
      <div class="space-y-2 max-h-[400px] overflow-y-auto">
        {#each jobs as job}
          <div class="flex items-center justify-between p-3 bg-surface-subtle rounded-lg">
            <div class="flex items-center gap-3">
              <svelte:component this={statusIcon(job.status)} size={18} class={statusColor(job.status)} />
              <div>
                <p class="text-sm text-text-primary font-medium capitalize">{job.status}</p>
                <p class="text-xs text-text-muted flex items-center gap-1">
                  <Clock size={10} />
                  {formatDate(job.started_at)}
                  {#if job.duration_ms > 0}
                    &middot; {job.duration_ms >= 1000
                      ? `${(job.duration_ms / 1000).toFixed(1)}s`
                      : `${job.duration_ms}ms`}
                  {/if}
                </p>
              </div>
            </div>
            <div class="flex items-center gap-4 text-xs text-text-muted">
              <span>{job.total_rows} rows</span>
              <span class="text-emerald-400">{job.inserted ?? 0} ins</span>
              <span class="text-amber-400">{job.updated ?? 0} upd</span>
              <span class="text-rose-400">{job.errors} err</span>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  {#snippet footer()}
    <Button variant="primary" onclick={() => open = false}>Close</Button>
  {/snippet}
</Modal>
