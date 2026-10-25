import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "CashMovementModal.svelte"),
    "utf-8",
  );
}

describe("CashMovementModal.svelte source-structure guards", () => {
  const src = getSource();

  it("surfaces the backend's message on a failed movement", () => {
    // The inline `err.response?.data?.error || ...` fed the canonical object
    // body to alert(), which printed "[object Object]".
    expect(src).toContain(
      'import { getApiErrorMessage } from "$shared/utils/error-utils"',
    );
    expect(src).toContain(
      "alert(getApiErrorMessage(e, labels.cashMovementFailed));",
    );
    expect(src).not.toContain("err.response?.data?.error");
  });

  it("keeps the modal open and re-enables the form on failure", () => {
    const fnIdx = src.indexOf("async function handleSubmit()");
    expect(fnIdx).toBeGreaterThan(-1);
    const catchIdx = src.indexOf("} catch (e: unknown) {", fnIdx);
    const finallyIdx = src.indexOf("} finally {", catchIdx);
    expect(catchIdx).toBeGreaterThan(fnIdx);
    expect(finallyIdx).toBeGreaterThan(catchIdx);
    // The close lives in the success branch only…
    const tryBlock = src.slice(fnIdx, catchIdx);
    expect(tryBlock).toContain("open = false;");
    // …the failure branch clears neither the modal nor the submitting latch.
    const catchBlock = src.slice(catchIdx, finallyIdx);
    expect(catchBlock).not.toContain("open = false;");
    expect(src).toContain("isSubmitting = false;");
  });
});
