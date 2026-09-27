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

  it("sends the stock filter to the server as a search param", () => {
    expect(src).toContain("listStock(arrangement.supplier_id, {");
    expect(src).toContain("limit: pageLimit");
    expect(src).toContain("offset: pageOffset");
    expect(src).toContain("search: filterQuery.trim() || undefined");
  });

  it("uses SearchBar component for the stock filter input", () => {
    expect(src).toContain("SearchBar");
    expect(src).toContain("placeholder={labels.consignmentFilterByProduct}");
    expect(src).toContain("bind:value={filterQuery}");
    expect(src).toContain("oninput={handleFilterInput}");
    // Stays mounted while a filter is typed so input focus survives refetch.
    expect(src).toContain('{#if rows.length > 0 || filterQuery !== ""}');
  });

  it("resets pagination to the first page when the filter input changes", () => {
    expect(src).toContain("function handleFilterInput() {");
    expect(src).toContain("pageOffset = 0");
  });

  it("paginates server-side and binds the server total", () => {
    expect(src).toContain("{total}");
    expect(src).toContain("void load()");
    expect(src).not.toContain("slice(pageOffset");
  });

  it("shows a no-results empty state when the filter matches nothing", () => {
    expect(src).toContain("{#if filterQuery.trim()}");
    expect(src).toContain("icon={Search}");
    expect(src).toContain("labels.noResultsFor");
    expect(src).toContain("filterQuery.trim()");
  });
});
