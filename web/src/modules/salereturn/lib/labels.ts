import { labels } from "$shared/i18n";
import type { IneligibleReason } from "../types";

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
    default:
      return "muted";
  }
}
