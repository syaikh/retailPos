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

  it("delegates payout to PayoutModal component", () => {
    expect(src).toContain("PayoutModal");
    expect(src).toContain("bind:show={showPayoutModal}");
  });

  it("uses getApiErrorMessage for error extraction", () => {
    expect(src).toContain("getApiErrorMessage");
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

  it("has historyTab state for filtering by status", () => {
    expect(src).toContain("historyTab");
    expect(src).toContain('"all" | "pending" | "paid"');
  });

  it("sends tab and search filters to the server", () => {
    expect(src).toContain('return "pending_payment"');
    expect(src).toContain('return "paid"');
    expect(src).toContain("tabStatus()");
    expect(src).toContain("historySearch.trim() || undefined");
    expect(src).toContain("historyTab");
    expect(src).toContain('"all" | "pending" | "paid"');
    expect(src).toContain("$derived");
    expect(src).toContain("{total}");
    expect(src).not.toContain("slice(pageOffset");
  });

  it("has filteredPreviewItems derived for search in preview", () => {
    expect(src).toContain("filteredPreviewItems");
    expect(src).toContain("previewSearch");
  });

  it("uses SearchBar for settlement history filter", () => {
    expect(src).toContain("SearchBar");
  });

  it("accepts initialTab prop", () => {
    expect(src).toContain("initialTab");
  });
});
