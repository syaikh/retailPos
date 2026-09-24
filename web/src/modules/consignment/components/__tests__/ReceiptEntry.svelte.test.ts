import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "ReceiptEntry.svelte"),
    "utf-8",
  );
}

describe("ReceiptEntry.svelte source-structure guards", () => {
  const src = getSource();

  it("uses table layout for receipt entry lines", () => {
    expect(src).toContain("<table");
    expect(src).toContain("<thead");
    expect(src).toContain("<tbody");
  });

  it("does not have notes field in Line interface", () => {
    const lineInterface = src.slice(
      src.indexOf("interface Line"),
      src.indexOf("interface Line") + 100,
    );
    expect(lineInterface).not.toContain("notes:");
  });

  it("has optionsForLine derived for product deduplication", () => {
    expect(src).toContain("optionsForLine");
    expect(src).toContain("$derived");
  });

  it("uses getApiErrorMessage for error extraction", () => {
    expect(src).toContain("getApiErrorMessage");
  });

  it("imports Copy and Check icons for SKU copy feature", () => {
    expect(src).toContain("Copy");
    expect(src).toContain("Check");
  });

  it("has copySku function for copying product SKUs to clipboard", () => {
    expect(src).toContain("function copySku");
    expect(src).toContain("navigator.clipboard.writeText");
  });

  it("has showCopied state for tracking copied SKUs", () => {
    expect(src).toContain("showCopied");
  });

  it("uses 2xl modal size for receipt entry", () => {
    expect(src).toContain('size="2xl"');
  });

  it("has filterProductSearch state for product filtering", () => {
    expect(src).toContain("filterProductSearch");
    expect(src).toContain("$state");
  });

  it("has filteredReceipts derived that filters by product name or SKU", () => {
    expect(src).toContain("filteredReceipts");
    expect(src).toContain("$derived");
    expect(src).toContain("product_name");
    expect(src).toContain("product_sku");
  });

  it("uses SearchBar component for filter input", () => {
    expect(src).toContain("SearchBar");
    expect(src).toContain("consignmentFilterByProduct");
  });

  it("shows different empty state messages when filter is active", () => {
    expect(src).toContain("consignmentNoMatchingReceipts");
    expect(src).toContain("consignmentNoMatchingReceiptsSubtitle");
  });

  it("resets pagination when filter changes", () => {
    expect(src).toContain('pageOffset = 0');
  });
});
