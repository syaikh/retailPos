import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(component: string): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", component),
    "utf-8",
  );
}

describe("Consignment components error-handling pattern", () => {
  it("ArrangementsPage uses getApiErrorMessage", () => {
    const src = getSource("ArrangementsPage.svelte");
    expect(src).toContain("getApiErrorMessage");
  });

  it("PendingReturnPage uses getApiErrorMessage", () => {
    const src = getSource("PendingReturnPage.svelte");
    expect(src).toContain("getApiErrorMessage");
  });

  it("ReceiptEntry uses getApiErrorMessage", () => {
    const src = getSource("ReceiptEntry.svelte");
    expect(src).toContain("getApiErrorMessage");
  });

  it("ReturnPage uses getApiErrorMessage", () => {
    const src = getSource("ReturnPage.svelte");
    expect(src).toContain("getApiErrorMessage");
  });

  it("TermsEditor uses getApiErrorMessage", () => {
    const src = getSource("TermsEditor.svelte");
    expect(src).toContain("getApiErrorMessage");
  });
});
