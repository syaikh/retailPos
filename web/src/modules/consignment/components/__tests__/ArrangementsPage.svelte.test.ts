import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "ArrangementsPage.svelte"),
    "utf-8",
  );
}

describe("ArrangementsPage.svelte source-structure guards", () => {
  const src = getSource();

  it("gates the force-return action behind consignment.force_return", () => {
    expect(src).toContain(
      'userPermissions.includes("consignment.force_return")',
    );
    expect(src).toContain("{#if canForceReturn}");
  });

  it("falls back to the ordinary bulk return without the force grant", () => {
    expect(src).toContain("{:else}");
    expect(src).toContain("bulkReturnAllStock");
    expect(src).toContain("showBulkReturnModal = true");
  });

  it("releases ownership through the force-return endpoint", () => {
    expect(src).toContain("forceReturnAllStock(activeArrangement.id)");
    expect(src).toContain("showForceReturnModal = true");
    expect(src).toContain("bind:open={showForceReturnModal}");
    expect(src).toContain(
      "{forceReturning ? labels.saving : labels.consignmentForceReturn}",
    );
  });

  it("keeps the legacy bulk-return path when the grant is absent", () => {
    expect(src).toContain("showBulkReturnModal");
    expect(src).toContain("bulkReturning");
  });

  it("adds a reconcile tab gated by consignment.reconcile", () => {
    expect(src).toContain('userPermissions.includes("consignment.reconcile")');
    expect(src).toContain("disabled={!canReconcile}");
    expect(src).toContain('activeTab = "reconcile"');
    expect(src).toContain("labels.consignmentTabReconcile");
  });

  it("mounts ReconciliationPage for the reconcile tab", () => {
    expect(src).toContain(
      'import ReconciliationPage from "./ReconciliationPage.svelte"',
    );
    expect(src).toContain('{:else if activeTab === "reconcile"}');
    expect(src).toContain("<ReconciliationPage");
    expect(src).toContain("{canReconcile}");
    expect(src).toContain("oncreated={refreshArrangement}");
  });

  it("refreshes the arrangement after a forced return", () => {
    expect(src).toContain(
      "toast.success(labels.consignmentForceReturnSuccess)",
    );
    expect(src).toContain("returnRefreshKey++");
    expect(src).toContain("await refreshArrangement()");
  });
});
