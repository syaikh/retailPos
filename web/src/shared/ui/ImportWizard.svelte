<script lang="ts">
  import { Button, Modal, DropZone, PreviewTable, ValidationSummary, ProgressDialog, ImportSummary } from '$shared/ui';
  import { Loader2, AlertCircle, Upload, CheckCircle2, ArrowLeft, ArrowRight, Download, FileSpreadsheet } from 'lucide-svelte';
  import { uploadPreview, confirmImport, getProgress, cancelImport, downloadTemplate } from '$shared/services/import-export-service';
  import type { PreviewResult, ImportProgress } from '$shared/types/import-export';
  import { labels, t } from '$shared/i18n';

  let {
    open = $bindable(false),
    module = '',
    displayName = '',
    onComplete = () => {},
  }: {
    open?: boolean;
    module?: string;
    displayName?: string;
    onComplete?: () => void;
  } = $props();

  type Step = 'upload' | 'preview' | 'progress' | 'summary';

  let step = $state<Step>('upload');
  let file = $state<File | null>(null);
  let preview = $state<PreviewResult | null>(null);
  let progress = $state<ImportProgress | null>(null);
  let loading = $state(false);
  let error = $state('');

  let errorReport = $state('');

  let columns = $derived<string[]>([]);
  let validationErrors = $derived<import('$shared/types/import-export').ValidationError[]>([]);

  let pollTimer: ReturnType<typeof setInterval> | null = null;

  $effect(() => {
    if (open) {
      step = 'upload';
      file = null;
      preview = null;
      progress = null;
      error = '';
      errorReport = '';
    }
  });

  $effect(() => {
    if (preview) {
      columns = preview.rows.length > 0
        ? Object.keys((preview.rows[0] as unknown as Record<string, unknown>).new_values || (preview.rows[0] as unknown as Record<string, unknown>).newValues || {})
        : [];
      validationErrors = preview.rows
        .filter(r => r.status === 'error')
        .flatMap(r => r.errors);
    }
  });

  async function handleUpload() {
    if (!file || !module) return;
    loading = true;
    error = '';
    try {
      preview = await uploadPreview(module, file);
      step = 'preview';
    } catch (err: any) {
      error = err?.response?.data?.error || err?.message || labels.previewFailed;
    } finally {
      loading = false;
    }
  }

  async function handleConfirm() {
    if (!module || !preview?.token) return;
    loading = true;
    error = '';
    try {
      const result = await confirmImport(module, preview.token);
      progress = {
        job_id: result.job_id,
        status: result.status,
        progress_pct: 0,
        total_rows: preview.total_rows,
        processed: 0,
        inserted: 0,
        updated: 0,
        errors: preview.error_count,
        started_at: new Date().toISOString(),
        duration_ms: 0,
      };
      step = 'progress';
    } catch (err: any) {
      error = err?.response?.data?.error || err?.message || labels.confirmFailed;
    } finally {
      loading = false;
    }
  }

  function onProgressDone(p: ImportProgress) {
    progress = p;
    errorReport = p.error_report || '';
    if (p.status === 'completed') onComplete();
    if (pollTimer) clearInterval(pollTimer);
    step = 'summary';
  }

  $effect(() => {
    if (step === 'progress' && progress && !['completed', 'failed', 'cancelled'].includes(progress.status)) {
      pollTimer = setInterval(async () => {
        try {
          const p = await getProgress(String(progress!.job_id));
          progress = p;
          if (['completed', 'failed', 'cancelled'].includes(p.status)) {
            onProgressDone(p);
          }
        } catch {
          if (pollTimer) clearInterval(pollTimer);
        }
      }, 1000);
    }
    return () => {
      if (pollTimer) clearInterval(pollTimer);
    };
  });

  async function handleCancel() {
    if (!progress) return;
    try {
      await cancelImport(String(progress.job_id));
    } catch {
      // ignore
    }
  }

  function handleClose() {
    if (pollTimer) clearInterval(pollTimer);
    open = false;
  }

  function handleBack() {
    step = 'upload';
    preview = null;
    error = '';
  }

  function handleDownloadTemplate() {
    if (module) downloadTemplate(module);
  }

  let canImport = $derived(
    preview && preview.error_count === 0 && preview.total_rows > 0
  );
</script>

<Modal bind:open={open} title={displayName ? t('importWithName', { name: displayName }) : labels.import} size="xl">
  <div class="space-y-4 min-h-[200px]">
    {#if step === 'upload'}
      <div class="space-y-4">
        <DropZone bind:file accept=".csv,.xlsx" disabled={loading} />

        <div class="flex items-center justify-between">
          <button
            type="button"
            class="text-xs text-primary-light hover:underline inline-flex items-center gap-1"
            onclick={handleDownloadTemplate}
          >
            <Download size={12} />
            {labels.downloadTemplate}
          </button>
          {#if file}
            <span class="text-xs text-text-muted">
              <FileSpreadsheet size={12} class="inline mr-1" />
              {t('fileSelected', { format: file.name.endsWith('.csv') ? 'CSV' : 'XLSX' })}
            </span>
          {/if}
        </div>

        {#if error}
          <div class="flex items-start gap-2 p-3 bg-danger-subtle/10 rounded-lg">
            <AlertCircle size={16} class="text-danger shrink-0 mt-0.5" />
            <p class="text-sm text-danger">{error}</p>
          </div>
        {/if}
      </div>

    {:else if step === 'preview' && preview}
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3 text-sm">
            <span class="text-text-primary font-medium">{labels.preview}</span>
            <span class="text-text-muted">{preview.total_rows} {labels.rows}</span>
            <span class="inline-flex items-center gap-1 text-emerald-400">
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
              {preview.insert_count} {labels.insert}
            </span>
            <span class="inline-flex items-center gap-1 text-amber-400">
              <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
              {preview.update_count} {labels.update}
            </span>
            {#if preview.error_count > 0}
              <span class="inline-flex items-center gap-1 text-rose-400">
                <span class="w-1.5 h-1.5 rounded-full bg-rose-400"></span>
                {preview.error_count} {labels.error}
              </span>
            {/if}
          </div>
        </div>

        <ValidationSummary errors={validationErrors} />

        <PreviewTable rows={preview.rows} {columns} />

        {#if !canImport && preview.error_count > 0}
          <div class="p-3 bg-danger-subtle/10 rounded-lg flex items-start gap-2">
            <AlertCircle size={16} class="text-danger shrink-0 mt-0.5" />
            <p class="text-sm text-danger">
              {labels.fixErrorsBeforeImport}
            </p>
          </div>
        {/if}
      </div>

    {:else if step === 'progress'}
      <ProgressDialog
        bind:progress
        onCancel={handleCancel}
        onClose={handleClose}
      />

    {:else if step === 'summary'}
      <ImportSummary progress={progress} error_report={errorReport} />
    {/if}
  </div>

  {#snippet footer()}
    {#if step === 'upload'}
      <Button variant="secondary" onclick={handleClose}>{labels.cancel}</Button>
      <Button variant="primary" disabled={!file || loading} onclick={handleUpload}>
        {#if loading}
          <Loader2 size={16} class="animate-spin" /> {labels.parsing}
        {:else}
          <Upload size={14} /> {labels.preview}
        {/if}
      </Button>

    {:else if step === 'preview'}
      <div class="flex items-center gap-3 w-full">
        <Button variant="secondary" onclick={handleBack}>
          <ArrowLeft size={14} /> {labels.back}
        </Button>
        <div class="flex-1"></div>
        <Button variant="secondary" onclick={handleClose}>{labels.cancel}</Button>
          <Button variant="primary" disabled={!canImport || loading} onclick={handleConfirm}>
            {#if loading}
              <Loader2 size={16} class="animate-spin" /> {labels.confirming}
            {:else}
              <CheckCircle2 size={14} /> {t('importRows', { count: (preview?.insert_count ?? 0) + (preview?.update_count ?? 0) })}
            {/if}
          </Button>
      </div>

    {:else if step === 'summary'}
      <Button variant="primary" onclick={handleClose}>{labels.close}</Button>
    {/if}
  {/snippet}
</Modal>
