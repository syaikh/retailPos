<script lang="ts">
  import { onMount } from 'svelte';
  import { goto, getPath } from '$app/router';
  import { toast } from '$shared/stores/toast.svelte';
  import { formatDateInJakarta } from '$shared/utils/jakartaTime';
  import { getHistory, getImportDetail, getImportRows } from '$shared/services/import-export-service';
  import type { ImportProgress, ImportDetail, ImportRowWithErrors } from '$shared/types/import-export';
  import { Button, Skeleton, Badge } from '$shared/ui';
  import { ArrowLeft, History, ChevronRight, ChevronDown, Clock, CheckCircle2, AlertCircle, Ban, Loader2, Database } from 'lucide-svelte';

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

  function badgeVariant(status: string) {
    if (status === 'completed') return 'success' as const;
    if (status === 'failed') return 'danger' as const;
    if (status === 'cancelled') return 'muted' as const;
    return 'primary' as const;
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

  const PREVIEW_LIMIT = 20;

  function snapshotColumns(): string[] {
    if (!detail?.snapshot?.rows_data?.length) return [];
    return Object.keys(detail.snapshot.rows_data[0]);
  }

  let previewRows = $derived(detail?.snapshot?.rows_data?.slice(0, PREVIEW_LIMIT) ?? []);
  let previewTotal = $derived(detail?.snapshot?.rows_data?.length ?? 0);

  function previewColumnLabel(key: string): string {
    const labels: Record<string, string> = {
      _row: 'No',
      IsActive: 'Active',
      UnitOfMeasure: 'Unit of Measure',
      WeightGrams: 'Weight (g)',
    };
    if (labels[key]) return labels[key];
    const candidate = key.replace(/([A-Z])/g, ' $1').replace(/^./, (s) => s.toUpperCase()).trim();
    if (candidate === candidate.toUpperCase()) return key;
    return candidate;
  }

  function statusIcon(status: string) {
    if (status === 'completed') return CheckCircle2;
    if (status === 'failed') return AlertCircle;
    if (status === 'cancelled') return Ban;
    return Loader2;
  }

  function isObject(v: unknown): v is Record<string, unknown> {
    return typeof v === 'object' && v !== null && !Array.isArray(v);
  }

  function prettyEntries(obj: Record<string, unknown>): [string, string][] {
    return Object.entries(obj).map(([k, v]) => [
      k,
      v === null || v === undefined ? '—' : String(v),
    ]);
  }
</script>

<div class="space-y-5">
  {#if selectedJob}
    <div class="flex items-center justify-between">
      <Button variant="ghost" onclick={backToList} class="gap-1.5">
        <ArrowLeft size={14} />
        Back to Import History
      </Button>
      <span class="text-xs text-text-muted font-mono">Job #{selectedJob.job_id}</span>
    </div>

    {#if loadingDetail}
      <div class="card p-8 flex items-center justify-center">
        <Loader2 size={24} class="animate-spin text-primary-light" />
      </div>
    {:else if detail}
      <div class="card p-4 sm:p-5 space-y-4">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <Badge variant={badgeVariant(detail.progress.status)}>
              {detail.progress.status}
            </Badge>
            <div class="flex items-center gap-1.5 text-xs text-text-muted">
              <Clock size={12} />
              {formatDate(detail.progress.started_at)}
              {#if detail.progress.duration_ms > 0}
                <span>&middot; {formatDuration(detail.progress.duration_ms)}</span>
              {/if}
            </div>
          </div>
          <span class="text-xs text-text-muted">{detail.progress.total_rows} rows</span>
        </div>

        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <div class="bg-surface-subtle rounded-lg p-3 border border-border/50">
            <p class="text-sm font-bold text-text-primary">{detail.progress.total_rows}</p>
            <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mt-0.5">Total rows</p>
          </div>
          <div class="bg-surface-subtle rounded-lg p-3 border border-border/50">
            <p class="text-sm font-bold text-emerald-400">{detail.progress.inserted ?? 0}</p>
            <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mt-0.5">Inserted</p>
          </div>
          <div class="bg-surface-subtle rounded-lg p-3 border border-border/50">
            <p class="text-sm font-bold text-amber-400">{detail.progress.updated ?? 0}</p>
            <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mt-0.5">Updated</p>
          </div>
          <div class="bg-surface-subtle rounded-lg p-3 border border-border/50">
            <p class="text-sm font-bold text-rose-400">{detail.progress.errors}</p>
            <p class="text-[10px] uppercase tracking-wider text-text-muted font-semibold mt-0.5">Errors</p>
          </div>
        </div>
      </div>

      {#if detail.snapshot?.rows_data?.length}
        <div class="border border-border/50 rounded-xl bg-surface-subtle/40 overflow-hidden">
          <button
            class="w-full flex items-center justify-between px-4 sm:px-5 py-3 hover:bg-surface-hover/30 transition-colors text-left"
            onclick={() => showPreview = !showPreview}
            aria-expanded={showPreview}
          >
            <div class="flex items-center gap-2.5 min-w-0">
              {#if showPreview}
                <ChevronDown size={14} class="shrink-0 text-text-muted" />
              {:else}
                <ChevronRight size={14} class="shrink-0 text-text-muted" />
              {/if}
              <span class="text-sm font-medium text-text-primary">Preview Imported Data</span>
              <span class="text-[11px] font-semibold text-text-muted bg-surface-default px-2 py-0.5 rounded-md border border-border/50">{previewTotal}</span>
            </div>
            <span class="text-xs text-text-muted">{showPreview ? 'Hide' : 'Show'}</span>
          </button>

          {#if showPreview}
            <div class="animate-in slide-in-from-top-1 fade-in duration-200 border-t border-border/40">
              <!-- Desktop table -->
              <div class="hidden sm:block overflow-x-auto max-h-80 overflow-y-auto">
                <table class="w-full text-sm">
                  <thead class="sticky top-0 bg-surface-subtle z-10 border-b border-border/40">
                    <tr>
                      {#each snapshotColumns() as col}
                        <th class="text-left px-4 py-2.5 text-[11px] font-semibold uppercase tracking-wider text-text-muted whitespace-nowrap">{previewColumnLabel(col)}</th>
                      {/each}
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-border/30">
                    {#each previewRows as row}
                      <tr class="hover:bg-surface-hover/20 transition-colors">
                        {#each snapshotColumns() as col}
                          <td class="px-4 py-2.5 text-sm text-text-primary max-w-[200px] truncate" title={row[col] ?? ''}>{row[col] ?? ''}</td>
                        {/each}
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>

              <!-- Mobile card layout -->
              <div class="sm:hidden divide-y divide-border/30">
                {#each previewRows as row, ri}
                  <div class="px-4 py-3 space-y-1.5 {ri % 2 === 0 ? 'bg-surface-subtle/30' : ''}">
                    {#each snapshotColumns() as col}
                      <div class="flex items-start gap-2">
                        <span class="text-[11px] font-semibold uppercase tracking-wider text-text-muted shrink-0 w-24 truncate">{previewColumnLabel(col)}</span>
                        <span class="text-sm text-text-primary break-words min-w-0 line-clamp-2">{row[col] ?? ''}</span>
                      </div>
                    {/each}
                  </div>
                {/each}
              </div>

              {#if previewTotal > PREVIEW_LIMIT}
                <div class="px-4 sm:px-5 py-2.5 border-t border-border/30 text-center">
                  <span class="text-xs text-text-muted">Showing first {PREVIEW_LIMIT} of {previewTotal} imported rows</span>
                </div>
              {/if}
            </div>
          {/if}
        </div>
      {:else}
        <div class="border border-border/50 rounded-xl bg-surface-subtle/40 p-6 text-center">
          <Database size={20} class="mx-auto text-text-muted" />
          <p class="text-sm text-text-muted mt-2">No preview data available</p>
        </div>
      {/if}

      <div class="card overflow-hidden">
        <div class="px-4 sm:px-5 py-3 border-b border-border flex items-center justify-between">
          <h3 class="text-sm font-semibold text-text-primary">Row Results</h3>
          <span class="text-xs text-text-muted">{rows.length} {rows.length === 1 ? 'row' : 'rows'}</span>
        </div>
        {#if rows.length === 0}
          <div class="px-4 py-12 text-center text-text-muted text-sm">No row data available</div>
        {:else}
          <div class="overflow-x-auto max-h-96 overflow-y-auto">
            <table class="w-full text-sm">
              <thead class="sticky top-0 bg-surface-subtle z-10 border-b border-border">
                <tr>
                  <th class="text-left px-4 py-3 text-xs font-semibold uppercase tracking-wider text-text-muted w-14">#</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold uppercase tracking-wider text-text-muted w-24">Status</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold uppercase tracking-wider text-text-muted">New Values</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold uppercase tracking-wider text-text-muted">Errors</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border/60">
                {#each rows as row}
                  <tr class="hover:bg-surface-default/60 transition-colors">
                    <td class="px-4 py-3 text-sm text-text-muted font-mono">{row.row_number}</td>
                    <td class="px-4 py-3">
                      <Badge variant={row.status === 'error' ? 'danger' : row.status === 'insert' ? 'success' : 'warning'} size="sm">
                        {row.status}
                      </Badge>
                    </td>
                    <td class="px-4 py-3">
                      {#if row.new_values && isObject(row.new_values)}
                        <div class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
                          {#each prettyEntries(row.new_values) as [key, val]}
                            <span class="text-text-muted font-medium whitespace-nowrap">{key}</span>
                            <span class="text-text-primary break-words min-w-0">{val}</span>
                          {/each}
                        </div>
                      {:else if row.new_values}
                        <span class="text-xs text-text-muted">{JSON.stringify(row.new_values)}</span>
                      {:else}
                        <span class="text-xs text-text-muted">—</span>
                      {/if}
                    </td>
                    <td class="px-4 py-3">
                      {#if row.errors?.length}
                        <div class="space-y-1.5">
                          {#each row.errors as err}
                            <div class="flex items-start gap-2">
                              <span class="text-[10px] uppercase tracking-wider font-semibold text-rose-400 shrink-0 mt-0.5">{err.field}</span>
                              <span class="text-xs text-text-secondary">{err.reason}</span>
                            </div>
                          {/each}
                        </div>
                      {:else}
                        <span class="text-xs text-text-muted">—</span>
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
    <div class="flex items-center justify-between">
      <div>
        <p class="text-xs text-text-muted font-medium uppercase tracking-wider">{displayName}</p>
        <h2 class="text-xl font-bold text-white mt-0.5">Import History</h2>
      </div>
      <Button variant="secondary" onclick={() => goto(parentPath)} size="sm" class="gap-1.5">
        <ArrowLeft size={14} />
        Back
      </Button>
    </div>

    <div class="card overflow-hidden">
      {#if loading}
        <div class="p-4 space-y-2">
          {#each Array(4) as _}
            <Skeleton class="h-14 w-full rounded-xl" />
          {/each}
        </div>
      {:else if jobs.length === 0}
        <div class="px-4 py-12 text-center">
          <div class="w-16 h-16 mx-auto flex items-center justify-center rounded-full bg-surface-subtle">
            <History size={28} class="text-text-muted" />
          </div>
          <p class="text-text-primary font-semibold mt-4">No import history</p>
          <p class="text-text-muted text-sm mt-1">Import data using Bulk Actions to see history here</p>
        </div>
      {:else}
        <div class="divide-y divide-border/60">
          {#each jobs as job (job.job_id)}
            {@const Icon = statusIcon(job.status)}
            <button
              class="w-full text-left px-4 sm:px-5 py-3.5 hover:bg-surface-hover/50 transition-colors flex items-center justify-between gap-4"
              onclick={() => selectJob(job)}
            >
              <div class="flex items-center gap-3 min-w-0">
                <Icon size={16} class="shrink-0 {job.status === 'completed' ? 'text-emerald-400' : job.status === 'failed' ? 'text-rose-400' : job.status === 'cancelled' ? 'text-text-muted' : 'text-primary-light'}" />
                <div class="min-w-0 flex items-baseline gap-2">
                  <Badge variant={badgeVariant(job.status)} size="sm">{job.status}</Badge>
                  <span class="text-xs text-text-muted whitespace-nowrap">
                    {formatDate(job.started_at)}
                    {#if job.duration_ms > 0}
                      &middot; {formatDuration(job.duration_ms)}
                    {/if}
                  </span>
                </div>
              </div>
              <div class="flex items-center gap-3 sm:gap-5 text-xs shrink-0">
                <span class="text-text-muted hidden sm:inline">{job.total_rows} rows</span>
                <span class="text-emerald-400">{job.inserted ?? 0}</span>
                <span class="text-amber-400">{job.updated ?? 0}</span>
                <span class="text-rose-400">{job.errors}</span>
                <ChevronRight size={14} class="text-text-muted" />
              </div>
            </button>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>
