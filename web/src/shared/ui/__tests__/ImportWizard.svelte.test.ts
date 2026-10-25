import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "ImportWizard.svelte"),
    "utf-8",
  );
}

describe("ImportWizard.svelte source-structure guards", () => {
  const src = getSource();

  it("schedules polls with a self-rearming setTimeout, not setInterval", () => {
    // The interval's catch only cleared the timer: one transient failure used
    // to stop polling forever, freezing the wizard on a step with no footer
    // buttons (onComplete never fired while the job kept running server-side).
    expect(src).toContain(
      "let pollTimer: ReturnType<typeof setTimeout> | null = null;",
    );
    expect(src).toContain("function stopPolling()");
    expect(src).toContain("clearTimeout(pollTimer);");
    expect(src).toContain(
      "pollTimer = setTimeout(() => void pollOnce(jobId), delayMs);",
    );
    expect(src).not.toContain("setInterval");
    expect(src).not.toContain("clearInterval");
  });

  it("keeps polling after a failure instead of dropping the timer", () => {
    const fnIdx = src.indexOf("async function pollOnce(jobId: number) {");
    expect(fnIdx).toBeGreaterThan(-1);
    const catchIdx = src.indexOf("} catch (err) {", fnIdx);
    expect(catchIdx).toBeGreaterThan(fnIdx);
    const catchBlock = src.slice(
      catchIdx,
      src.indexOf("  $effect(() => {", catchIdx),
    );
    expect(catchBlock).toContain("pollFailures += 1;");
    expect(catchBlock).toContain("schedulePoll(jobId, POLL_RETRY_DELAY_MS);");
    // The poll is re-armed unconditionally on every path, success or failure —
    // the failure branch must not clear it.
    expect(catchBlock).not.toContain("stopPolling();");
  });

  it("backs off after repeated failures but never stops the job poll", () => {
    expect(src).toContain("const MAX_POLL_FAILURES = 5;");
    expect(src).toContain("const POLL_INTERVAL_MS = 1000;");
    expect(src).toContain("const POLL_RETRY_DELAY_MS = 5000;");
    expect(src).toContain("if (pollFailures === MAX_POLL_FAILURES) {");
  });

  it("reports an unknown status as an error, not as a completion", () => {
    // Claiming completion would make the parent show a success toast for a job
    // whose status is unknown; the banner stays until a poll succeeds.
    expect(src).toContain(
      "error = getApiErrorMessage(err, labels.failedToLoad);",
    );
    // The failure branch stays on the progress step: it must not hand the
    // user a terminal-looking summary for a job whose state is unknown.
    const pollIdx = src.indexOf("async function pollOnce(jobId: number) {");
    const catchBlock = src.slice(
      src.indexOf("} catch (err) {", pollIdx),
      src.indexOf("  $effect(() => {", pollIdx),
    );
    expect(catchBlock).not.toContain('step = "summary"');
    expect(catchBlock).not.toContain("onComplete(");
  });

  it("clears any earlier error unconditionally once a poll succeeds", () => {
    // Unconditional: a banner left by a failed cancel must not survive the
    // next successful poll and reach the summary step next to a finished job.
    expect(src).toContain('pollFailures = 0;\n      error = "";');
    expect(src).not.toContain("pollErrorShown");
  });

  it("renders the error banner on every step, not only the upload step", () => {
    // The progress step had no error surface at all, so a failure was silent.
    expect(src).toContain("{#if error}");
    expect(src.split("{#if error}").length - 1).toBe(1);
    const bannerIdx = src.indexOf("{#if error}");
    const uploadIdx = src.indexOf('{#if step === "upload"}');
    expect(uploadIdx).toBeGreaterThan(bannerIdx);
  });

  it("resets the failure streak from the polling effect, not the timer", () => {
    expect(src).toContain("progress !== null");
    expect(src).toContain("schedulePoll(progress.job_id, POLL_INTERVAL_MS);");
    expect(src).toContain("pollFailures = 0;\n      schedulePoll(");
    expect(src).toContain("pollActive = false;\n      stopPolling();\n    };");
  });

  it("cannot re-arm the poll after the effect is torn down", () => {
    // Navigating away (or logging out) while a progress request is in flight
    // used to let the response schedule a timer no cleanup would ever clear,
    // polling the API forever from a destroyed component — and a late response
    // from an earlier import could clobber a newly started one on reopen.
    expect(src).toContain("let pollActive = false;");
    expect(src).toContain("pollActive = true;");
    expect(
      src.split("if (!pollActive || progress?.job_id !== jobId) return;")
        .length - 1,
    ).toBe(2);
    expect(src).toContain("pollActive = false;\n      stopPolling();");
  });

  it("leaves polling running when a cancel request fails", () => {
    expect(src).toContain(
      'onProgressDone({ ...progress, status: "cancelled" });',
    );
    expect(src).toContain(
      "error = getApiErrorMessage(err, labels.errorOccurred);",
    );
    // A failed cancel must NOT jump to the summary step (which would render a
    // false "Cancelled" terminal state while the job still runs).
    expect(src).not.toContain("catch {\n      // ignore");
  });

  it("stops the poll when the dialog closes", () => {
    expect(src).toContain("function handleClose()");
    expect(src).toContain("stopPolling();\n    open = false;");
  });
});
