<script lang="ts">
  import { onMount } from 'svelte';
  import { goto, getPath } from '$app/router';
  import { toast } from '$shared/stores/toast.svelte';
  import { formatDateInJakarta } from '$shared/utils/jakartaTime';
  import { getHistory, getImportDetail, getImportRows } from '$shared/services/import-export-service';
  import type { ImportProgress, ImportDetail, ImportRowWithErrors } from '$shared/types/import-export';
  import { Button, PageHeader, Skeleton } from '$shared/ui';
  import { ArrowLeft, CheckCircle2, AlertCircle, Ban, Loader2, History, Clock, Eye, EyeOff, ChevronRight } from 'lucide-svelte';

  const pathToModule: Record<string, string> = {
    '/categories/import-history': 'categories',
    '/brands/import-history': 'brands',
    '/units-of-measure/import-history': 'uoms',
    '/customers/import-history': 'customers',
    '/products/import-history': 'products',
  };

  const moduleDisplay: Record<string, string> = {
    categories: 'Categories',
    brands: 'Brands',
    uoms: 'Units of Measure',
    customers: 'Customers',
    products: 'Products',
  };

  const moduleParent: Record<string, string> = {
    categories: '/categories',
    brands: '/brands',
    uoms: '/units-of-measure',
    customers: '/customers',
    products: '/inventory/products',
  };

  let currentPath = $state(getPath());
  let module = $derived(pathToModule[currentPath] || 'categories');
  let displayName = $derived(moduleDisplay[module] || module);
  let parentPath = $derived(moduleParent[module] || '/');

  let loading = $state(true);
  let jobs = $state<ImportProgress[]>([]);
  let selectedJob = $state<ImportProgress | null>(null);
  let loadingDetail = $state(false);
  let detail = $state<ImportDetail | null>(null);
  let rows = $state<ImportRowWithErrors[]>([]);
  let showPreview = $state(false);

  onMount(async () => {
    await fetchJobs();
  });

  async function fetchJobs() {
    try {
      loading = true;
      jobs = await getHistory(module);
    } catch (err: any) {
      toast.error(err?.response?.data?.error || 'Failed to load import history');
    } finally {
      loading = false;
    }
  }

  async function selectJob(job: ImportProgress) {
    selectedJob = job;
    loadingDetail = true;
    showPreview = false;
    detail = null;
    rows = [];
    try {
      const [d, r] = await Promise.all([
        getImportDetail(module, String(job.job_id)),
        getImportRows(module, String(job.job_id)),
      ]);
      detail = d;
      rows = r;
    } catch (err: any) {
      toast.error(err?.response?.data?.error || 'Failed to load job detail');
    } finally {
      loadingDetail = false;
    }
  }

  function backToList() {
    selectedJob = null;
    detail = null;
    rows = [];
  }

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
    if (!iso) return '—';
    return formatDateInJakarta(iso);
  }

  function formatDuration(ms: number) {
    if (!ms || ms <= 0) return '—';
    if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
    return `${ms}ms`;
  }

  function snapshotColumns(): string[] {
    if (!detail?.snapshot?.rows_data?.length) return [];
    return Object.keys(detail.snapshot.rows_data[0]);
  }

  function rowStatusBadge(status: string) {
    if (status === 'insert') return 'bg-emerald-500/20 text-emerald-400';
    if (status === 'update') return 'bg-amber-500/20 text-amber-400';
    if (status === 'error') return 'bg-rose-500/20 text-rose-400';
    return 'bg-surface-default text-text-muted';
  }
</script>

<div class="space-y-5">
  {#if selectedJob}
    <div class="flex items-center justify-between">
      <Button variant="ghost" onclick={backToList} class="gap-2">
        <ArrowLeft size={16} />
        Back to History
      </Button>
      <span class="text-sm text-text-muted">Job #{selectedJob.job_id}</span>
    </div>

    {#if loadingDetail}
      <div class="card p-8 flex items-center justify-center">
        <Loader2 size={24} class="animate-spin text-primary-light" />
      </div>
    {:else if detail}
      <div class="card p-4 space-y-4">
        <div class="flex items-center justify-between">
          {#each [statusIcon(detail.progress.status)] as Icon}
            <div class="flex items-center gap-3">
              <Icon size={20} class={statusColor(detail.progress.status)} />
              <div>
                <p class="text-sm font-medium capitalize">{detail.progress.status}</p>
                <p class="text-xs text-text-muted">
                  {formatDate(detail.progress.started_at)} &middot; {formatDuration(detail.progress.duration_ms)}
                </p>
              </div>
            </div>
          {/each}
          <div class="flex items-center gap-4 text-xs text-text-muted">
            <span>{detail.progress.total_rows} rows</span>
            <span class="text-emerald-400">{detail.progress.inserted ?? 0} ins</span>
            <span class="text-amber-400">{detail.progress.updated ?? 0} upd</span>
            <span class="text-rose-400">{detail.progress.errors} err</span>
          </div>
        </div>

        {#if detail.snapshot?.rows_data?.length}
          <button
            class="flex items-center gap-2 text-sm text-primary-light hover:underline"
            onclick={() => showPreview = !showPreview}
          >
            {#if showPreview}
              <EyeOff size={14} />
            {:else}
              <Eye size={14} />
            {/if}
            {showPreview ? 'Hide' : 'Show'} Preview Data ({detail.snapshot.rows_data.length} rows)
          </button>

          {#if showPreview}
            <div class="overflow-x-auto border border-border rounded-lg max-h-80 overflow-y-auto">
              <table class="w-full text-xs">
                <thead class="bg-muted/50 sticky top-0">
                  <tr>
                    {#each snapshotColumns() as col}
                      <th class="text-left p-2 font-semibold whitespace-nowrap">{col}</th>
                    {/each}
                  </tr>
                </thead>
                <tbody>
                  {#each detail.snapshot.rows_data as row, i}
                    <tr class="border-t border-border {i % 2 === 0 ? 'bg-surface-default' : 'bg-surface-subtle'}">
                      {#each snapshotColumns() as col}
                        <td class="p-2 whitespace-nowrap">{row[col] ?? ''}</td>
                      {/each}
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        {/if}
      </div>

      <div class="card overflow-hidden">
        <div class="px-4 py-3 border-b border-border">
          <h3 class="text-sm font-semibold">Row Results ({rows.length})</h3>
        </div>
        {#if rows.length === 0}
          <div class="p-8 text-center text-text-muted text-sm">No row data available</div>
        {:else}
          <div class="overflow-x-auto max-h-96 overflow-y-auto">
            <table class="w-full text-xs">
              <thead class="bg-muted/50 sticky top-0">
                <tr>
                  <th class="text-left p-2 font-semibold">#</th>
                  <th class="text-left p-2 font-semibold">Status</th>
                  <th class="text-left p-2 font-semibold">New Values</th>
                  <th class="text-left p-2 font-semibold">Errors</th>
                </tr>
              </thead>
              <tbody>
                {#each rows as row, i}
                  <tr class="border-t border-border {i % 2 === 0 ? 'bg-surface-default' : 'bg-surface-subtle'}">
                    <td class="p-2 font-mono">{row.row_number}</td>
                    <td class="p-2">
                      <span class="inline-block px-2 py-0.5 rounded text-xs font-medium {rowStatusBadge(row.status)}">
                        {row.status}
                      </span>
                    </td>
                    <td class="p-2 max-w-xs truncate" title={JSON.stringify(row.new_values)}>
                      {row.new_values ? JSON.stringify(row.new_values).slice(0, 80) : '—'}
                    </td>
                    <td class="p-2">
                      {#if row.errors?.length}
                        <div class="space-y-1">
                          {#each row.errors as err}
                            <div class="text-rose-400" title={err.reason}>
                              <span class="font-medium">{err.field}:</span> {err.reason}
                            </div>
                          {/each}
                        </div>
                      {:else}
                        <span class="text-text-muted">—</span>
                      {/if}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    {/if}
  {:else}
    <PageHeader title={`Import History — ${displayName}`} subtitle={`View past imports for ${displayName}`}>
      {#snippet actions()}
        <Button variant="secondary" onclick={() => goto(parentPath)} class="gap-2">
          <ArrowLeft size={14} />
          Back to {displayName}
        </Button>
      {/snippet}
    </PageHeader>

    <div class="card overflow-hidden">
      {#if loading}
        <div class="p-4 space-y-3">
          {#each Array(4) as _}
            <Skeleton class="h-14 w-full rounded-lg" />
          {/each}
        </div>
      {:else if jobs.length === 0}
        <div class="px-4 py-12 text-center">
          <div class="w-20 h-20 mx-auto flex items-center justify-center">
            <History size={32} class="text-text-muted" />
          </div>
          <p class="text-text-primary font-semibold mt-4">No import history</p>
          <p class="text-text-muted text-sm mt-1">Import data using the Bulk Actions menu to see history here</p>
        </div>
      {:else}
        <div class="divide-y divide-border">
          {#each jobs as job (job.job_id)}
            {@const Icon = statusIcon(job.status)}
            <button
              class="w-full text-left p-4 hover:bg-surface-hover/50 transition-colors flex items-center justify-between"
              onclick={() => selectJob(job)}
            >
              <div class="flex items-center gap-3 min-w-0">
                <Icon size={18} class={`shrink-0 ${statusColor(job.status)}`} />
                <div class="min-w-0">
                  <p class="text-sm text-text-primary font-medium capitalize truncate">{job.status}</p>
                  <p class="text-xs text-text-muted flex items-center gap-1">
                    <Clock size={10} />
                    {formatDate(job.started_at)}
                    {#if job.duration_ms > 0}
                      &middot; {formatDuration(job.duration_ms)}
                    {/if}
                  </p>
                </div>
              </div>
              <div class="flex items-center gap-4 text-xs text-text-muted shrink-0">
                <span>{job.total_rows} rows</span>
                <span class="text-emerald-400">{job.inserted ?? 0} ins</span>
                <span class="text-amber-400">{job.updated ?? 0} upd</span>
                <span class="text-rose-400">{job.errors} err</span>
                <ChevronRight size={14} />
              </div>
            </button>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>
