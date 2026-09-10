import { describe, it, expect } from "vitest";
import type { SettlementItem } from "..";

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
});
