import { labels } from "$shared/i18n";
import type { IneligibleReason, ReturnableItem } from "../types";

/** Whether the server will pay this line out (refund) or replace the goods.
 *
 * Mirrors ProcessReturn's proposeOutcomes: a line becomes a replacement
 * candidate whenever ANY sellable stock is on hand — the decision is not
 * scoped to the requested quantity, and it may still degrade to a refund
 * when a concurrent sale wins the stock during deduction. Anything the UI
 * derives from the outcome (the row badge, the refund estimate and the
 * approval warning) must use this rule rather than comparing stock against
 * the quantity, or it disagrees with the server. */
export function isRefundOutcome(item: ReturnableItem): boolean {
  return item.stock_available <= 0;
}

/** Maps the server's ineligible reason code (returnable.go) to its
 * user-facing text, so both the drawer's disabled button and the modal's
 * banner render the same explanation. */
export function ineligibleText(
  reason: IneligibleReason | undefined,
  windowDays: number,
): string {
  switch (reason) {
    case "not_completed":
      return labels.returnIneligibleNotCompleted;
    case "window_expired":
      return labels.returnIneligibleWindowExpired.replace(
        "{days}",
        String(windowDays),
      );
    case "shift_not_open":
      return labels.returnIneligibleShiftNotOpen;
    case "fully_returned":
      return labels.returnIneligibleFullyReturned;
    default:
      return "";
  }
}

/** Maps a return's wire status to its badge label. A rejected return flips
 * to 'cancelled' server-side (see presentReturn). */
export function returnStatusLabel(status: string): string {
  switch (status) {
    case "pending_approval":
      return labels.returnStatusPendingApproval;
    case "completed":
      return labels.statusCompleted;
    case "cancelled":
      return labels.statusCancelled;
    case "expired":
      return labels.returnStatusExpired;
    default:
      return status;
  }
}

export function returnStatusVariant(
  status: string,
): "success" | "warning" | "danger" | "muted" {
  switch (status) {
    case "completed":
      return "success";
    case "pending_approval":
      return "warning";
    case "cancelled":
      return "danger";
    case "expired":
      // Distinct from the default so an elapsed approval window reads as
      // "needs attention" rather than a generic grey chip, matching the
      // dedicated label returnStatusLabel already gives it.
      return "danger";
    default:
      return "muted";
  }
}

/** Maps the stored reason code (processReturn request wire value) to display
 * text. Receipt payloads carry translated text, never wire codes — neither the
 * print agent nor the paper translates. Unknown codes pass through verbatim so
 * a future reason renders as-is instead of blank. */
export function returnReasonLabel(code: string): string {
  switch (code) {
    case "damaged":
      return labels.returnReasonCodeDamaged;
    case "defective":
      return labels.returnReasonCodeDefective;
    case "wrong_item":
      return labels.returnReasonCodeWrongItem;
    case "other":
      return labels.returnReasonCodeOther;
    default:
      return code;
  }
}

/** Maps the stored condition code to display text; same contract as
 * returnReasonLabel. Phase 1 locks condition to 'damaged' server-side. */
export function returnConditionLabel(code: string): string {
  switch (code) {
    case "damaged":
      return labels.returnConditionDamaged;
    default:
      return code;
  }
}
