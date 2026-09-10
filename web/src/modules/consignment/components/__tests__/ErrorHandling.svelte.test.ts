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
  it("ArrangementsPage uses instanceof Error error-handling pattern", () => {
    const src = getSource("ArrangementsPage.svelte");
    expect(src).toContain("e instanceof Error");
  });

  it("PendingReturnPage uses instanceof Error error-handling pattern", () => {
    const src = getSource("PendingReturnPage.svelte");
    expect(src).toContain("e instanceof Error");
  });

  it("ReceiptEntry uses instanceof Error error-handling pattern", () => {
    const src = getSource("ReceiptEntry.svelte");
    expect(src).toContain("e instanceof Error");
  });

  it("ReturnPage uses instanceof Error error-handling pattern", () => {
    const src = getSource("ReturnPage.svelte");
    expect(src).toContain("e instanceof Error");
  });

  it("TermsEditor uses instanceof Error error-handling pattern", () => {
    const src = getSource("TermsEditor.svelte");
    expect(src).toContain("e instanceof Error");
  });
});
