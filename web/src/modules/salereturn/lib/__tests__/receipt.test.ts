import { describe, it, expect } from "vitest";
import { labels } from "$shared/i18n";
import { buildReturnReceipt } from "../receipt";
import type { SaleReturn, SaleReturnItem } from "../../types";

function item(overrides: Partial<SaleReturnItem> = {}): SaleReturnItem {
  return {
    id: 11,
    sale_item_id: 7,
    product_id: 3,
    product_name: "Kopi Hitam",
    quantity: 2,
    unit_price: 5000,
    subtotal: 10000,
    dpp_amount: 9009,
    tax_amount: 991,
    tax_rate: 0.11,
    condition: "damaged",
    outcome: "refund",
    notes: null,
    replacements: [],
    ...overrides,
  };
}

function saleReturn(overrides: Partial<SaleReturn> = {}): SaleReturn {
  return {
    id: 9,
    return_number: "RETUR-0001",
    original_sale_id: 42,
    cashier_id: 1,
    store_id: 1,
    customer_id: null,
    shift_id: 5,
    return_type: "refund",
    reason: "damaged",
    condition: "damaged",
    notes: null,
    subtotal: 10000,
    discount: 0,
    restocking_fee: 0,
    tax_reversal: 991,
    total_refund_amount: 10000,
    status: "completed",
    change_due: 0,
    created_at: "2026-08-26 11:00:00",
    updated_at: "2026-08-26 11:00:00",
    expires_at: null,
    items: [item()],
    payments: [
      {
        id: 1,
        payment_method_id: 1,
        payment_method_code: "CASH",
        amount: 10000,
      },
    ],
    approval: null,
    ...overrides,
  };
}

describe("buildReturnReceipt", () => {
  it("tags the payload as a return document keyed by the return number", () => {
    const out = buildReturnReceipt(saleReturn(), "INV-2026-0001");
    expect(out.document_type).toBe("return");
    expect(out.invoice_number).toBe("RETUR-0001");
    expect(out.original_invoice_number).toBe("INV-2026-0001");
    expect(out.created_at).toBe("2026-08-26 11:00:00");
  });

  it("maps a refund: money and payout tenders, translated reason/condition", () => {
    const out = buildReturnReceipt(saleReturn(), "INV-1");
    expect(out.return_type).toBe("refund");
    expect(out.refund_amount).toBe(10000);
    expect(out.payments).toEqual([{ method: "CASH", amount: 10000 }]);
    // Never the wire codes: the agent and the paper do not translate.
    expect(out.reason).toBe(labels.returnReasonCodeDamaged);
    expect(out.condition).toBe(labels.returnConditionDamaged);
    expect(out.reason).not.toBe("damaged");
  });

  it("maps a replacement: same-product units, no replacement record noise", () => {
    const out = buildReturnReceipt(
      saleReturn({
        return_type: "replacement",
        total_refund_amount: 0,
        payments: [],
        items: [
          item({
            outcome: "written_off",
            replacements: [{ id: 1, product_id: 3, quantity: 2 }],
          }),
        ],
      }),
      "INV-1",
    );
    expect(out.return_type).toBe("replacement");
    expect(out.replacements).toEqual([{ name: "Kopi Hitam", quantity: 2 }]);
    expect(out.refund_amount).toBe(0);
  });

  it("copies item lines as name/qty/price display data", () => {
    const out = buildReturnReceipt(saleReturn(), "INV-1");
    expect(out.items).toEqual([
      { name: "Kopi Hitam", quantity: 2, unit_price: 5000 },
    ]);
  });

  it("passes an unknown reason code through verbatim instead of blanking", () => {
    const out = buildReturnReceipt(
      saleReturn({ reason: "custom_code" }),
      "INV-1",
    );
    expect(out.reason).toBe("custom_code");
  });

  it("omits empty optional fields (notes, replacements)", () => {
    const out = buildReturnReceipt(saleReturn(), "INV-1");
    expect("notes" in out).toBe(false);
    expect("replacements" in out).toBe(false);
  });

  it("carries notes through when present", () => {
    const out = buildReturnReceipt(
      saleReturn({ notes: "customer complained" }),
      "INV-1",
    );
    expect(out.notes).toBe("customer complained");
  });
});
