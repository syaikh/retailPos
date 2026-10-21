import { writable } from "svelte/store";

export interface ReceiptPayment {
  method: string;
  amount: number;
  reference_number?: string;
}

/** The subset a sale receipt and a return receipt genuinely share: the
 * document number, when it was written, the line list, and nothing else.
 * Everything else is document-specific and lives on the two payloads below —
 * forcing both into one interface would leave a dozen fields meaningless in
 * half the states (change due on a replacement? savings on a return?). */
interface ReceiptBase {
  /** Document number: the sale's invoice, or the return's return_number. */
  invoice_number: string;
  created_at: string;
  items: Array<{
    name: string;
    quantity: number;
    unit_price: number;
    original_price?: number;
    pricing_rule_name?: string;
    pricing_type?: string;
  }>;
  customer_name?: string;
}

export interface ReceiptData extends ReceiptBase {
  /** Absent on payloads built before the return-reprint feature (Phase 4.4);
   * an absent tag means the original sale receipt. */
  document_type?: "sale";
  total_amount: number;
  subtotal_dpp?: number;
  tax?: number;
  paymentMethod: string;
  payments?: ReceiptPayment[];
  cashReceived: number;
  changeDue: number;
  total_savings?: number;
}

/** Receipt for a processed sales return (Phase 4.4). Numbers are display data
 * computed server-side; the print agent must never recompute them. */
export interface ReturnReceiptData extends ReceiptBase {
  document_type: "return";
  return_type: "replacement" | "refund";
  /** The invoice of the sale this return belongs to ("Inv asal"). */
  original_invoice_number: string;
  /** Tax-inclusive money handed back; 0 on a replacement (goods only). */
  refund_amount: number;
  /** Payout tenders for a refund; empty/absent on a replacement. */
  payments?: ReceiptPayment[];
  /** Translated display text, NOT the wire codes (damaged/…) — the agent and
   * the overlay render whatever they are given. */
  reason: string;
  condition: string;
  notes?: string;
  /** Same-product units handed over on a replacement. */
  replacements?: Array<{ name: string; quantity: number }>;
}

/** Anything the print pipeline can carry: a sale receipt or a return receipt,
 * told apart by document_type (which the agent echoes as its `type` field). */
export type PrintDocument = ReceiptData | ReturnReceiptData;

export const printReceipt = writable<PrintDocument | null>(null);
