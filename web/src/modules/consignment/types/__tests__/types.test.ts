import { describe, it, expect } from "vitest";
import type { SettlementItem } from "..";
import {
  RETURN_REASON_DAMAGED,
  RETURN_REASON_EXPIRED,
  RETURN_REASON_CUSTOMER_RETURN,
  RETURN_REASON_TERMINATION,
  RETURN_REASON_OTHER,
  RETURN_REASONS,
  RETURN_REASON_LABELS,
} from "..";

describe("Consignment types", () => {
  describe("SettlementItem", () => {
    it("product_id accepts null (nullable column)", () => {
      const item: SettlementItem = {
        id: 1,
        consignment_settlement_id: 1,
        consignment_sale_item_id: 1,
        product_id: null,
        quantity: 1,
        unit_price: 10000,
        subtotal: 10000,
        store_share: 2000,
      };
      expect(item.product_id).toBeNull();
    });

    it("product_id accepts a number", () => {
      const item: SettlementItem = {
        id: 1,
        consignment_settlement_id: 1,
        consignment_sale_item_id: 1,
        product_id: 42,
        quantity: 1,
        unit_price: 10000,
        subtotal: 10000,
        store_share: 2000,
      };
      expect(item.product_id).toBe(42);
    });
  });

  describe("Return reasons", () => {
    it("RETURN_REASON_TERMINATION constant is 'termination'", () => {
      expect(RETURN_REASON_TERMINATION).toBe("termination");
    });

    it("RETURN_REASONS includes termination", () => {
      expect(RETURN_REASONS).toContain(RETURN_REASON_TERMINATION);
      expect(RETURN_REASONS).toEqual([
        RETURN_REASON_DAMAGED,
        RETURN_REASON_EXPIRED,
        RETURN_REASON_CUSTOMER_RETURN,
        RETURN_REASON_TERMINATION,
        RETURN_REASON_OTHER,
      ]);
    });

    it("RETURN_REASON_LABELS has label for termination", () => {
      expect(RETURN_REASON_LABELS[RETURN_REASON_TERMINATION]).toBe(
        "returnReasonTermination",
      );
    });

    it("all five reasons have labels", () => {
      for (const reason of RETURN_REASONS) {
        expect(RETURN_REASON_LABELS[reason]).toBeDefined();
        expect(typeof RETURN_REASON_LABELS[reason]).toBe("string");
      }
    });
  });
});
