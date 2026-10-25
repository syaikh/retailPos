import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "GoodsReceiptModal.svelte"),
    "utf-8",
  );
}

describe("GoodsReceiptModal.svelte source-structure guards", () => {
  const src = getSource();

  it("enables submit as soon as ANY quantity is entered, good or damaged", () => {
    // A shipment can arrive entirely damaged: the PO line still counts
    // good+damaged as received, so gating on the good total made a legitimate
    // receipt impossible to record.
    expect(src).toContain("function hasEnteredQuantities()");
    expect(src).toContain(
      "return items.some((item) => item.qty_good > 0 || item.qty_damaged > 0);",
    );
    expect(src).toContain("disabled={saving || !hasEnteredQuantities()}");
    expect(src).not.toContain("disabled={saving || getTotalGood() === 0}");
  });

  it("submits exactly the rows the gate lets through", () => {
    // The gate and handleSubmit's validItems filter must select the same rows,
    // otherwise the button can be enabled for a payload that would be empty.
    expect(src).toContain(
      "const validItems = items.filter(\n        (item) => item.qty_good > 0 || item.qty_damaged > 0,\n      );",
    );
  });

  it("still toasts when nothing at all was entered", () => {
    expect(src).toContain("if (validItems.length === 0) {");
    expect(src).toContain("labels.pleaseEnterReceivingQuantities");
  });

  it("caps each entry at the remaining quantity so a damaged line fits", () => {
    expect(src).toContain(
      "if (item.qty_damaged > rem) item.qty_damaged = rem;",
    );
    expect(src).toContain("if (item.qty_good > rem) item.qty_good = rem;");
  });
});
