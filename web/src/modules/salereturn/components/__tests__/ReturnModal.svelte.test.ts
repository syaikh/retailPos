import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "ReturnModal.svelte"),
    "utf-8",
  );
}

describe("ReturnModal.svelte source-structure guards", () => {
  const src = getSource();

  it("imports Modal, Button, Input from $shared/ui", () => {
    expect(src).toContain("Modal");
    expect(src).toContain("Button");
    expect(src).toContain("Input");
    expect(src).toContain("$shared/ui");
  });

  it("composes the form from ReturnItemRow and ReturnSummary", () => {
    expect(src).toContain('import ReturnItemRow from "./ReturnItemRow.svelte"');
    expect(src).toContain('import ReturnSummary from "./ReturnSummary.svelte"');
  });

  it("loads the snapshot via getReturnableSale and submits via processReturn", () => {
    expect(src).toContain("getReturnableSale");
    expect(src).toContain("processReturn");
    expect(src).toContain("salereturn-service");
  });

  it("exports props: saleId, show (bindable), onreturned", () => {
    expect(src).toContain("saleId = null");
    expect(src).toContain("show = $bindable(false)");
    expect(src).toContain("onreturned");
  });

  it("re-fetches the snapshot every time the modal opens", () => {
    expect(src).toContain("if (show && saleId)");
    expect(src).toContain("load(saleId)");
  });

  it("locks the condition to damaged in the request body", () => {
    expect(src).toContain('condition: "damaged"');
  });

  it("offers the four PRD reason codes", () => {
    expect(src).toContain('value="damaged"');
    expect(src).toContain('value="defective"');
    expect(src).toContain('value="wrong_item"');
    expect(src).toContain('value="other"');
  });

  it("explains ineligibility instead of rendering the form", () => {
    expect(src).toContain("ineligibleText");
    expect(src).toContain("data.ineligible_reason");
  });

  it("distinguishes the pending-approval outcome from the completed one", () => {
    expect(src).toContain('ret.status === "pending_approval"');
    expect(src).toContain("labels.returnSuccessPending");
    expect(src).toContain("labels.returnSuccess");
  });

  it("disables submit until a line is selected and eligibility is confirmed", () => {
    expect(src).toContain("data?.eligible === true");
    expect(src).toContain("selectedItems.length > 0");
  });

  it("shows the approval-threshold notice through ReturnSummary", () => {
    expect(src).toContain("threshold={data.approval_threshold}");
  });

  it("explains a sale whose every line is unselectable", () => {
    // A fully-consignment sale is eligible by design yet has no selectable
    // line; without this the modal opens on a disabled submit with no reason.
    expect(src).toContain("hasSelectable");
    expect(src).toContain("labels.returnNoSelectableItems");
  });

  it("keeps a completed return open on a result view instead of closing (Phase 4.4)", () => {
    expect(src).toContain("completed = ret");
    expect(src).toContain('ret.status === "completed"');
    // The pending path still closes: there is nothing to print yet.
    expect(src).toContain("show = false");
  });

  it("offers printing the just-completed return receipt", () => {
    expect(src).toContain("buildReturnReceipt");
    expect(src).toContain("printReceiptWithToast");
    expect(src).toContain("labels.returnPrintReceipt");
    // The return number becomes the receipt's own document number, and the
    // sale's invoice from the snapshot is the "Inv asal" link.
    expect(src).toContain("buildReturnReceipt(completed, data.invoice_number)");
  });

  it("clears the result state on close and on every reopen", () => {
    // Without this a second return in the same session would open straight
    // onto the previous return's receipt screen.
    expect(src).toContain("completed = null");
  });
});
