import type { ReturnReceiptData } from "$shared/stores/printReceipt.svelte";
import type { SaleReturn } from "../types";
import { returnConditionLabel, returnReasonLabel } from "./labels";

/** Maps a processed return (getReturn / processReturn detail shape) onto the
 * return-receipt payload the print pipeline carries (Phase 4.4).
 *
 * The original invoice number is a parameter rather than a presenter field:
 * every current caller already knows it — the transaction drawer displays it,
 * the return modal reads it from the returnable snapshot — so the backend
 * stays untouched.
 *
 * Numbers are copied verbatim: the print agent and the preview overlay render
 * precomputed display data and must never recompute totals.
 */
export function buildReturnReceipt(
  ret: SaleReturn,
  originalInvoiceNumber: string,
): ReturnReceiptData {
  // A replacement hands over the same product that came back (ProcessReturn
  // rejects mixed-product replacements), so the name lives on the returned
  // line, not on the replacement record.
  const replacements = ret.items.flatMap((it) =>
    it.replacements.map((rep) => ({
      name: it.product_name,
      quantity: rep.quantity,
    })),
  );

  return {
    document_type: "return",
    invoice_number: ret.return_number,
    created_at: ret.created_at,
    return_type: ret.return_type === "replacement" ? "replacement" : "refund",
    original_invoice_number: originalInvoiceNumber,
    refund_amount: ret.total_refund_amount,
    payments: ret.payments.map((p) => ({
      method: p.payment_method_code,
      amount: p.amount,
    })),
    reason: returnReasonLabel(ret.reason),
    condition: returnConditionLabel(ret.condition),
    ...(ret.notes ? { notes: ret.notes } : {}),
    items: ret.items.map((it) => ({
      name: it.product_name,
      quantity: it.quantity,
      unit_price: it.unit_price,
    })),
    ...(replacements.length > 0 ? { replacements } : {}),
  };
}
