import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "StockPage.svelte"),
    "utf-8",
  );
}

describe("StockPage.svelte source-structure guards", () => {
  const src = getSource();

  it("filters the stock table by product name or SKU", () => {
    expect(src).toContain("const filteredRows = $derived(");
    expect(src).toContain("filterQuery.trim()");
    expect(src).toContain("r.product_name?.toLowerCase().includes(q)");
    expect(src).toContain("r.product_sku?.toLowerCase().includes(q)");
  });

  it("uses SearchBar component for the stock filter input", () => {
    expect(src).toContain("SearchBar");
    expect(src).toContain("placeholder={labels.consignmentFilterByProduct}");
    expect(src).toContain("bind:value={filterQuery}");
    expect(src).toContain("oninput={handleFilterInput}");
    // Only shown once rows are loaded and at least one row exists.
    expect(src).toContain("{#if !loading && rows.length > 0}");
  });

  it("resets pagination to the first page when the filter input changes", () => {
    expect(src).toContain("function handleFilterInput() {");
    expect(src).toContain("pageOffset = 0");
  });

  it("paginates over the filtered rows, not the raw list", () => {
    expect(src).toContain(
      "filteredRows.slice(pageOffset, pageOffset + pageLimit)",
    );
    expect(src).toContain("total={filteredRows.length}");
  });

  it("shows a no-results empty state when the filter matches nothing", () => {
    expect(src).toContain("filteredRows.length === 0");
    expect(src).toContain("icon={Search}");
    expect(src).toContain("labels.noResultsFor");
    expect(src).toContain("filterQuery.trim()");
  });
});
