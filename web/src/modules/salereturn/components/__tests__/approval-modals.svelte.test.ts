import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(file: string): string {
  return readFileSync(path.join(path.dirname(__filename), "..", file), "utf-8");
}

describe("PendingReturnsModal.svelte source-structure guards", () => {
  const src = getSource("PendingReturnsModal.svelte");

  it("imports the queue from listPendingApprovals", () => {
    expect(src).toContain("listPendingApprovals");
    expect(src).toContain("salereturn-service");
  });

  it("re-fetches the queue on every open", () => {
    expect(src).toContain("if (show) load()");
  });

  it("delegates a row to the ApprovalModal review and refreshes after a decision", () => {
    expect(src).toContain('import ApprovalModal from "./ApprovalModal.svelte"');
    expect(src).toContain("bind:show={showReview}");
    expect(src).toContain("ondecided");
    expect(src).toContain("load();");
  });

  it("stays persistent while the review modal is stacked on top", () => {
    // Both are Modal instances binding Escape on window; without this one
    // Escape would close the review AND the queue behind it.
    expect(src).toContain("persistent={showReview}");
  });

  it("renders an empty state for an empty queue", () => {
    expect(src).toContain("labels.returnApprovalsEmpty");
  });

  it("renders each pending return's number, refund and expiry", () => {
    expect(src).toContain("ret.return_number");
    expect(src).toContain("formatCurrency(ret.total_refund_amount)");
    expect(src).toContain("ret.expires_at");
  });
});

describe("ApprovalModal.svelte source-structure guards", () => {
  const src = getSource("ApprovalModal.svelte");

  it("loads detail via getReturnById and decides via approve/reject", () => {
    expect(src).toContain("getReturnById");
    expect(src).toContain("approveReturn");
    expect(src).toContain("rejectReturn");
    expect(src).toContain("salereturn-service");
  });

  it("requires a rejection reason before calling rejectReturn", () => {
    expect(src).toContain("!rejectReason.trim()");
    expect(src).toContain("labels.returnRejectReasonRequired");
  });

  it("reveals the rejection reason field before submitting the rejection", () => {
    // The first click must not call decide() with an empty reason and toast
    // an error before the reviewer has had a chance to type.
    expect(src).toContain('if (mode !== "reject")');
    expect(src).toContain("labels.returnConfirmReject");
  });

  it("sends a blank approval note as undefined", () => {
    expect(src).toContain("note.trim() || undefined");
  });

  it("closes through the bound show prop and notifies via ondecided", () => {
    expect(src).toContain("show = $bindable(false)");
    expect(src).toContain("ondecided");
  });

  it("renders the return's items and total refund", () => {
    expect(src).toContain("detail.items");
    expect(src).toContain("formatCurrency(detail.total_refund_amount)");
  });
});

describe("ReturnItemRow.svelte source-structure guards", () => {
  const src = getSource("ReturnItemRow.svelte");

  it("blocks consignment and exhausted lines from selection", () => {
    expect(src).toContain("item.consignment");
    expect(src).toContain("item.returnable_quantity > 0");
    expect(src).toContain("disabled={!selectable}");
  });

  it("clamps the quantity stepper to the returnable quantity", () => {
    expect(src).toContain("item.returnable_quantity");
    expect(src).toContain("changeQuantity");
  });

  it("predicts replacement vs refund from the server's outcome rule", () => {
    // Any on-hand stock makes the line a replacement candidate server-side
    // (proposeOutcomes), regardless of the requested quantity — comparing
    // stock against quantity disagreed with what the server pays out.
    expect(src).toContain("isRefundOutcome(item)");
    expect(src).not.toContain("item.stock_available >= quantity");
    expect(src).toContain("labels.returnReplacement");
    expect(src).toContain("labels.returnRefund");
  });

  it("shows sold, already-returned and returnable quantities", () => {
    expect(src).toContain("labels.returnSoldQty");
    expect(src).toContain("labels.returnAlreadyReturned");
    expect(src).toContain("labels.returnAvailableQty");
  });
});

describe("ReturnSummary.svelte source-structure guards", () => {
  const src = getSource("ReturnSummary.svelte");

  it("counts only refund-outcome lines toward the refund estimate", () => {
    // Mirrors totalRefundOf: a replacement line pays nothing out, so
    // including it could raise the approval warning for a return the server
    // completes without approval.
    expect(src).toContain("isRefundOutcome");
    expect(src).toContain(".filter(isRefundOutcome)");
  });

  it("compares the estimate against the server-provided threshold", () => {
    expect(src).toContain("estimatedRefund > threshold");
    expect(src).toContain("labels.returnApprovalNeeded");
  });
});

describe("labels.ts source-structure guards", () => {
  const src = getSource("../lib/labels.ts");

  it("maps every ineligible reason code to a label", () => {
    expect(src).toContain('"not_completed"');
    expect(src).toContain('"window_expired"');
    expect(src).toContain('"shift_not_open"');
    expect(src).toContain('"fully_returned"');
  });

  it("interpolates the window days into the expired message", () => {
    expect(src).toContain('"{days}"');
    expect(src).toContain("String(windowDays)");
  });

  it("covers the return statuses surfaced by the approval queue", () => {
    expect(src).toContain('"pending_approval"');
    expect(src).toContain('"completed"');
    expect(src).toContain('"cancelled"');
    expect(src).toContain('"expired"');
  });

  it("gives 'expired' a distinct badge variant from the default", () => {
    // Without an explicit case it fell through to 'muted' while the label
    // already treated it as its own status.
    expect(src).toContain('case "expired":');
    expect(src).not.toMatch(/"expired":\s*return "muted"/);
  });

  it("exposes one shared refund-outcome rule for the row and the summary", () => {
    expect(src).toContain("export function isRefundOutcome");
    expect(src).toContain("stock_available <= 0");
  });
});
