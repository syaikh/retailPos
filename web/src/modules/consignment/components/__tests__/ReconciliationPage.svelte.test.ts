import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "ReconciliationPage.svelte"),
    "utf-8",
  );
}

describe("ReconciliationPage.svelte source-structure guards", () => {
  const src = getSource();

  it("loads the count sheet for the active arrangement", () => {
    expect(src).toContain("getReconciliationCountSheet(arrangement.id)");
  });

  it("starts a count against the arrangement, not the supplier", () => {
    expect(src).toContain("createReconciliation({");
    expect(src).toContain("arrangement_id: arrangement.id");
  });

  it("saves physical counts through the count endpoint payload", () => {
    expect(src).toContain("saveReconciliationCount(editorRecord.id, {");
    expect(src).toContain("physical_available: i.entry.physical");
  });

  it("omits the reason when the count matches the system", () => {
    expect(src).toContain(
      "reason: i.variance !== 0 ? i.entry.reason : undefined",
    );
  });

  it("requires a reason whenever the variance is non-zero", () => {
    expect(src).toContain(
      "editorItems.some((i) => i.variance !== 0 && !i.entry.reason)",
    );
  });

  it("previews a store liability only for lost or stolen shortfalls", () => {
    expect(src).toContain("reconcileReasonCreatesLiability(item.entry.reason)");
    expect(src).toContain("item.variance >= 0) return 0");
  });

  it("mirrors the server entitlement formula for the liability preview", () => {
    expect(src).toContain("storeShare(item.row, qty)");
    expect(src).toContain("entitlement - storeShare(item.row, qty)");
  });

  it("only offers post on counted drafts and edit on drafts", () => {
    expect(src).toContain("rec.status === RECONCILE_STATUS_DRAFT");
    expect(src).toContain("rec.status === RECONCILE_STATUS_COUNTED");
    expect(src).toContain("postReconciliation(editorRecord.id)");
  });

  it("renders a liability column and totals", () => {
    expect(src).toContain("labels.reconcileLiability");
    expect(src).toContain("labels.consignmentLiability");
    expect(src).toContain("labels.reconcileTotalShortfall");
    expect(src).toContain("labels.reconcileTotalSurplus");
  });

  it("gates new/count actions behind the reconcile permission", () => {
    expect(src).toContain("canReconcile");
  });
});
