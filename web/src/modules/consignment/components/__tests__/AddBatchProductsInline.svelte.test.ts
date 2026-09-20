import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "AddBatchProductsInline.svelte"),
    "utf-8",
  );
}

describe("AddBatchProductsInline.svelte source-structure guards", () => {
  const src = getSource();

  it("includes all required ProductFormData fields in createProduct call", () => {
    expect(src).toContain("barcode:");
    expect(src).toContain("category:");
    expect(src).toContain("brand_id:");
    expect(src).toContain("unit_of_measure_id:");
    expect(src).toContain("tax_class_id:");
    expect(src).toContain("weight_grams:");
    expect(src).toContain("description:");
  });

  it("uses getApiErrorMessage for error extraction", () => {
    expect(src).toContain("getApiErrorMessage");
  });

  it("does not use old e instanceof Error pattern", () => {
    expect(src).not.toContain("e instanceof Error");
  });

  it("sets status to active for created products", () => {
    expect(src).toContain('status: "active"');
  });

  it("initializes cost and stock to zero", () => {
    expect(src).toContain("cost: 0");
    expect(src).toContain("stock: 0");
  });
});
