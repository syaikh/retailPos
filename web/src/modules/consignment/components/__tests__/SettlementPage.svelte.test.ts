import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "SettlementPage.svelte"),
    "utf-8",
  );
}

describe("SettlementPage.svelte source-structure guards", () => {
  const src = getSource();

  it("imports FormattedNumberInput instead of NumberInput for payout amount", () => {
    expect(src).toContain("FormattedNumberInput");
    expect(src).not.toMatch(
      /import.*\bNumberInput\b.*from.*["'].*\$shared\/ui["']/,
    );
  });

  it("uses getApiErrorMessage for error extraction", () => {
    expect(src).toContain("getApiErrorMessage");
  });

  it("uses FormattedNumberInput for payout amount field", () => {
    expect(src).toContain("<FormattedNumberInput");
    expect(src).toContain("bind:value={payoutForm.amount}");
  });

  it("imports getSettlement for detail modal", () => {
    expect(src).toContain("getSettlement");
  });

  it("has detail modal state variables", () => {
    expect(src).toContain("showDetailModal");
    expect(src).toContain("detailSettlement");
    expect(src).toContain("loadingDetail");
  });

  it("has openDetail function that fetches settlement details", () => {
    expect(src).toContain("async function openDetail");
    expect(src).toContain("getSettlement(settlementId)");
  });

  it("makes settlement rows clickable with keyboard support", () => {
    expect(src).toContain('role="button"');
    expect(src).toContain('tabindex="0"');
    expect(src).toContain("onkeydown");
  });
});
