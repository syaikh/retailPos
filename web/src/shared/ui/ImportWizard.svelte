<script lang="ts">
  import {
    Button,
    Modal,
    DropZone,
    PreviewTable,
    ValidationSummary,
    ProgressDialog,
    ImportSummary,
  } from "$shared/ui";
  import {
    Loader2,
    AlertCircle,
    Upload,
    CheckCircle2,
    ArrowLeft,
    Download,
    FileSpreadsheet,
  } from "lucide-svelte";
  import {
    uploadPreview,
    confirmImport,
    getProgress,
    cancelImport,
    downloadTemplate,
  } from "$shared/services/import-export-service";
  import type {
    PreviewResult,
    ImportProgress,
  } from "$shared/types/import-export";
  import { labels, t } from "$shared/i18n";
  import { getApiErrorMessage } from "$shared/utils/error-utils";

  let {
    open = $bindable(false),
    module = "",
    displayName = "",
    onComplete = () => {},
  }: {
    open?: boolean;
    module?: string;
    displayName?: string;
    onComplete?: () => void;
  } = $props();

  type Step = "upload" | "preview" | "progress" | "summary";

  let step = $state<Step>("upload");
  let file = $state<File | null>(null);
  let preview = $state<PreviewResult | null>(null);
  let progress = $state<ImportProgress | null>(null);
  let loading = $state(false);
  let error = $state("");

  let errorReport = $state("");

  let columns = $derived<string[]>([]);
  let validationErrors = $derived<
    import("$shared/types/import-export").ValidationError[]
  >([]);

  let pollTimer: ReturnType<typeof setTimeout> | null = null;
  // Consecutive `getProgress` failures. Deliberately NOT reactive: it is only
  // mutated from the timer callback, so a failed attempt never re-runs (and
  // therefore never restarts) the polling effect.
  let pollFailures = 0;
  // True only while the polling effect wants a timer armed. Checked after
  // every await: the wizard can be torn down (navigation, logout) while a
  // request is in flight, and without this guard the response would re-arm a
  // timer that no effect cleanup will ever clear — polling forever against
  // the API from a destroyed component.
  let pollActive = false;
  const MAX_POLL_FAILURES = 5;
  const POLL_INTERVAL_MS = 1000;
  // After repeated failures, back off instead of hammering a struggling API —
  // but keep polling: the import job is still running server-side.
  const POLL_RETRY_DELAY_MS = 5000;

  $effect(() => {
    if (open) {
      step = "upload";
      file = null;
      preview = null;
      progress = null;
      error = "";
      errorReport = "";
    }
  });

  $effect(() => {
    if (preview) {
      columns =
        preview.rows.length > 0
          ? Object.keys(
              (preview.rows[0] as unknown as Record<string, unknown>)
                .new_values ||
                (preview.rows[0] as unknown as Record<string, unknown>)
                  .newValues ||
                {},
            )
          : [];
      validationErrors = preview.rows
        .filter((r) => r.status === "error")
        .flatMap((r) => r.errors);
    }
  });

  async function handleUpload() {
    if (!file || !module) return;
    loading = true;
    error = "";
    try {
      preview = await uploadPreview(module, file);
      step = "preview";
    } catch (err: unknown) {
      error = getApiErrorMessage(err, labels.previewFailed);
    } finally {
      loading = false;
    }
  }

  async function handleConfirm() {
    if (!module || !preview?.token) return;
    loading = true;
    error = "";
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
      step = "progress";
    } catch (err: unknown) {
      error = getApiErrorMessage(err, labels.confirmFailed);
    } finally {
      loading = false;
    }
  }

  function onProgressDone(p: ImportProgress) {
    progress = p;
    errorReport = p.error_report || "";
    if (p.status === "completed") onComplete();
    stopPolling();
    step = "summary";
  }

  function stopPolling() {
    if (pollTimer) {
      clearTimeout(pollTimer);
      pollTimer = null;
    }
  }

  // Always clears the previous timer first, so the effect re-run triggered by a
  // successful `progress = p` write and the timer the poll callback schedules
  // itself can never both stay armed (which would double the request rate).
  function schedulePoll(jobId: number, delayMs: number) {
    stopPolling();
    pollTimer = setTimeout(() => void pollOnce(jobId), delayMs);
  }

  async function pollOnce(jobId: number) {
    try {
      const p = await getProgress(String(jobId));
      // The wizard may have been torn down while this request was in flight;
      // continuing would write to a dead instance and, worse, re-arm a timer
      // no effect cleanup can reach. The job-id check drops a late response
      // from a previous import that arrives after the wizard was reopened.
      if (!pollActive || progress?.job_id !== jobId) return;
      // A successful poll means the job's state is known again, so clear any
      // earlier banner — a poll-failure streak or a cancel that failed —
      // rather than letting a stale error reach the summary step.
      pollFailures = 0;
      error = "";
      if (["completed", "failed", "cancelled"].includes(p.status)) {
        onProgressDone(p);
      } else {
        progress = p;
        schedulePoll(jobId, POLL_INTERVAL_MS);
      }
    } catch (err) {
      if (!pollActive || progress?.job_id !== jobId) return;
      // A transient failure must NOT stop the poll: the import job keeps
      // running server-side, and dropping the timer here used to freeze the
      // wizard on the progress step forever (that step has no footer buttons,
      // so onComplete/the parent's refresh were unreachable).
      pollFailures += 1;
      if (pollFailures === MAX_POLL_FAILURES) {
        // Status is unknown, so do not claim completion (the parent would show
        // a success toast). Surface it in the banner shared by every step.
        error = getApiErrorMessage(err, labels.failedToLoad);
      }
      schedulePoll(jobId, POLL_RETRY_DELAY_MS);
    }
  }

  $effect(() => {
    if (
      step === "progress" &&
      progress !== null &&
      !["completed", "failed", "cancelled"].includes(progress.status)
    ) {
      pollActive = true;
      // The effect only re-runs on a successful `progress` write or a step
      // change, so this resets the failure streak exactly when polling works.
      pollFailures = 0;
      schedulePoll(progress.job_id, POLL_INTERVAL_MS);
    } else {
      stopPolling();
    }
    return () => {
      // Runs before every re-run — the body re-arms the flag when polling
      // should continue — and once on destroy, where it is what makes an
      // in-flight response a no-op instead of a leaked, self-perpetuating
      // timer.
      pollActive = false;
      stopPolling();
    };
  });

  async function handleCancel() {
    if (!progress) return;
    try {
      await cancelImport(String(progress.job_id));
      onProgressDone({ ...progress, status: "cancelled" });
    } catch (err) {
      // Stay on the progress step so polling continues — the job was NOT
      // cancelled — but tell the user instead of failing silently.
      error = getApiErrorMessage(err, labels.errorOccurred);
    }
  }

  function handleClose() {
    stopPolling();
    open = false;
  }

  function handleBack() {
    step = "upload";
    preview = null;
    error = "";
  }

  function handleDownloadTemplate() {
    if (module) downloadTemplate(module);
  }

  const canImport = $derived(
    preview && preview.error_count === 0 && preview.total_rows > 0,
  );
</script>

<Modal
  bind:open
  title={displayName
    ? t("importWithName", { name: displayName })
    : labels.import}
  size="xl"
>
  <div class="space-y-4 min-h-[200px]">
    {#if error}
      <!-- Rendered for every step: a progress-polling failure is shown while
           polling continues, and any failure must survive a step change. -->
      <div class="flex items-start gap-2 p-3 bg-danger-subtle/10 rounded-lg">
        <AlertCircle size={16} class="text-danger shrink-0 mt-0.5" />
        <p class="text-sm text-danger">{error}</p>
      </div>
    {/if}
    {#if step === "upload"}
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
              {t("fileSelected", {
                format: file.name.endsWith(".csv") ? "CSV" : "XLSX",
              })}
            </span>
          {/if}
        </div>
      </div>
    {:else if step === "preview" && preview}
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3 text-sm">
            <span class="text-text-primary font-medium">{labels.preview}</span>
            <span class="text-text-muted"
              >{preview.total_rows} {labels.rows}</span
            >
            <span class="inline-flex items-center gap-1 text-success-light">
              <span class="w-1.5 h-1.5 rounded-full bg-success"></span>
              {preview.insert_count}
              {labels.insert}
            </span>
            <span class="inline-flex items-center gap-1 text-warning-light">
              <span class="w-1.5 h-1.5 rounded-full bg-warning-default"></span>
              {preview.update_count}
              {labels.update}
            </span>
            {#if preview.error_count > 0}
              <span class="inline-flex items-center gap-1 text-danger-light">
                <span class="w-1.5 h-1.5 rounded-full bg-danger"></span>
                {preview.error_count}
                {labels.error}
              </span>
            {/if}
          </div>
        </div>

        <ValidationSummary errors={validationErrors} />

        <PreviewTable rows={preview.rows} {columns} />

        {#if !canImport && preview.error_count > 0}
          <div
            class="p-3 bg-danger-subtle/10 rounded-lg flex items-start gap-2"
          >
            <AlertCircle size={16} class="text-danger shrink-0 mt-0.5" />
            <p class="text-sm text-danger">
              {labels.fixErrorsBeforeImport}
            </p>
          </div>
        {/if}
      </div>
    {:else if step === "progress"}
      <ProgressDialog bind:progress onCancel={handleCancel} />
    {:else if step === "summary"}
      <ImportSummary {progress} error_report={errorReport} />
    {/if}
  </div>

  {#snippet footer()}
    {#if step === "upload"}
      <Button variant="secondary" onclick={handleClose}>{labels.cancel}</Button>
      <Button
        variant="primary"
        disabled={!file || loading}
        onclick={handleUpload}
      >
        {#if loading}
          <Loader2 size={16} class="animate-spin" /> {labels.parsing}
        {:else}
          <Upload size={14} /> {labels.preview}
        {/if}
      </Button>
    {:else if step === "preview"}
      <div class="flex items-center gap-3 w-full">
        <Button variant="secondary" onclick={handleBack}>
          <ArrowLeft size={14} />
          {labels.back}
        </Button>
        <div class="flex-1"></div>
        <Button variant="secondary" onclick={handleClose}
          >{labels.cancel}</Button
        >
        <Button
          variant="primary"
          disabled={!canImport || loading}
          onclick={handleConfirm}
        >
          {#if loading}
            <Loader2 size={16} class="animate-spin" /> {labels.confirming}
          {:else}
            <CheckCircle2 size={14} />
            {t("importRows", {
              count:
                (preview?.insert_count ?? 0) + (preview?.update_count ?? 0),
            })}
          {/if}
        </Button>
      </div>
    {:else if step === "summary"}
      <Button variant="primary" onclick={handleClose}>{labels.close}</Button>
    {/if}
  {/snippet}
</Modal>
