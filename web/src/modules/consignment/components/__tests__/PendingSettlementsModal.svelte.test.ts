import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "PendingSettlementsModal.svelte"),
    "utf-8",
  );
}

describe("PendingSettlementsModal.svelte source-structure guards", () => {
  const src = getSource();

  it("imports Modal and EmptyState from $shared/ui", () => {
    expect(src).toContain(
      'import { Button, Modal, EmptyState } from "$shared/ui"',
    );
  });

  it("imports listSettlements from consignment-service", () => {
    expect(src).toContain(
      'import { listSettlements } from "../services/consignment-service"',
    );
  });

  it("imports PayoutModal component", () => {
    expect(src).toContain('import PayoutModal from "./PayoutModal.svelte"');
  });

  it("exports a show prop (bindable)", () => {
    expect(src).toContain("show = $bindable()");
    expect(src).toContain("show: boolean");
  });

  it("uses listSettlements with pending_payment filter", () => {
    expect(src).toContain('listSettlements(undefined, "pending_payment")');
  });

  it("has a settlements state array", () => {
    expect(src).toContain("settlements = $state<Settlement[]>([])");
  });

  it("has a selectedSettlement state for payout flow", () => {
    expect(src).toContain(
      "selectedSettlement = $state<Settlement | null>(null)",
    );
  });

  it("has a showPayoutModal state", () => {
    expect(src).toContain("showPayoutModal = $state(false)");
  });

  it("delegates payout to PayoutModal component", () => {
    expect(src).toContain("PayoutModal");
    expect(src).toContain("bind:show={showPayoutModal}");
    expect(src).toContain("settlement={selectedSettlement}");
  });
});
