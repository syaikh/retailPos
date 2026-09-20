import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "TermsEditor.svelte"),
    "utf-8",
  );
}

describe("TermsEditor.svelte source-structure guards", () => {
  const src = getSource();

  it("uses addTerm and removeTerm from consignment-service", () => {
    expect(src).toContain("addTerm");
    expect(src).toContain("removeTerm");
    expect(src).toContain('from "../services/consignment-service"');
  });

  it("uses searchAvailableProducts for search-based product assignment", () => {
    expect(src).toContain("searchAvailableProducts");
  });

  it("does not import Modal from shared/ui", () => {
    const sharedUiImport = src.match(
      /import\s*\{[^}]*\}\s*from\s*"\$shared\/ui"/,
    );
    if (sharedUiImport) {
      expect(sharedUiImport[0]).not.toContain("Modal");
    }
  });

  it("uses Trash2 icon for delete button", () => {
    expect(src).toContain("Trash2");
  });

  it("uses Plus icon for add button", () => {
    expect(src).toContain("Plus");
  });

  it("has a search input for product assignment", () => {
    expect(src).toContain("searchQuery");
    expect(src).toContain("handleSearchInput");
    expect(src).toContain("searchResults");
  });

  it("does not use productDropdownOpen or loadProducts", () => {
    expect(src).not.toContain("productDropdownOpen");
    expect(src).not.toContain("loadProducts()");
  });

  it("imports Copy and Check icons for SKU copy feature", () => {
    expect(src).toContain("Copy");
    expect(src).toContain("Check");
  });

  it("has copySku function for copying term SKUs to clipboard", () => {
    expect(src).toContain("function copySku");
    expect(src).toContain("navigator.clipboard.writeText");
  });

  it("has showCopied state for tracking copied SKUs", () => {
    expect(src).toContain("showCopied");
  });
});
