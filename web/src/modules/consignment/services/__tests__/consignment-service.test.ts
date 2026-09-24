import { describe, it, expect, vi, beforeEach } from "vitest";

const mockGet = vi.fn();
const mockPost = vi.fn();
const mockDelete = vi.fn();

vi.mock("$shared/api/http-client", () => ({
  default: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
    delete: (...args: unknown[]) => mockDelete(...args),
  },
}));

describe("consignment-service", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("bulkReturnAllStock", () => {
    it("builds return items from available stock and open pending returns", async () => {
      const stock = [
        {
          product_id: 10,
          product_name: "Product A",
          product_sku: "SKU-A",
          arrangement_id: 1,
          supplier_id: 5,
          available_qty: 3,
          pending_return_qty: 0,
        },
        {
          product_id: 20,
          product_name: "Product B",
          product_sku: "SKU-B",
          arrangement_id: 1,
          supplier_id: 5,
          available_qty: 2,
          pending_return_qty: 0,
        },
      ];
      const pendingReturns = [
        {
          id: 100,
          product_id: 10,
          qty: 2,
          status: "open",
          arrangement_id: 1,
          reason: "damaged",
        },
      ];

      mockGet
        .mockResolvedValueOnce({ data: { data: stock } }) // listStock
        .mockResolvedValueOnce({ data: { data: pendingReturns } }); // listPendingReturns

      const returnResult = {
        id: 1,
        return_number: "RET-001",
        items: [],
      };
      mockPost.mockResolvedValueOnce({ data: { data: returnResult } });

      const { bulkReturnAllStock } = await import("../consignment-service");
      const result = await bulkReturnAllStock(1, 5);

      expect(result).toEqual(returnResult);

      // listStock called with supplierId
      expect(mockGet).toHaveBeenCalledWith(
        expect.stringContaining("supplier_id=5"),
      );

      // createReturn called with correct items
      expect(mockPost).toHaveBeenCalledWith(
        "/consignment/returns",
        expect.objectContaining({
          arrangement_id: 1,
          notes: expect.stringContaining("Bulk return"),
          items: expect.arrayContaining([
            // pending return for product 10
            expect.objectContaining({
              product_id: 10,
              qty: 2,
              reason: "termination",
              pending_return_id: 100,
            }),
            // available stock for product 10
            expect.objectContaining({
              product_id: 10,
              qty: 3,
              reason: "termination",
            }),
            // available stock for product 20
            expect.objectContaining({
              product_id: 20,
              qty: 2,
              reason: "termination",
            }),
          ]),
        }),
      );

      // 3 items total: 1 pending + 2 available
      const callBody = mockPost.mock.calls[0][1];
      expect(callBody.items).toHaveLength(3);
    });

    it("skips stock rows with zero available and zero pending", async () => {
      const stock = [
        {
          product_id: 10,
          product_name: "Product A",
          arrangement_id: 1,
          available_qty: 0,
          pending_return_qty: 0,
        },
      ];
      const pendingReturns: unknown[] = [];

      mockGet
        .mockResolvedValueOnce({ data: { data: stock } })
        .mockResolvedValueOnce({ data: { data: pendingReturns } });

      const { bulkReturnAllStock } = await import("../consignment-service");

      await expect(bulkReturnAllStock(1, 5)).rejects.toThrow(
        "No stock to return",
      );
    });

    it("skips stock rows belonging to a different arrangement", async () => {
      const stock = [
        {
          product_id: 10,
          arrangement_id: 99,
          available_qty: 5,
          pending_return_qty: 0,
        },
      ];
      const pendingReturns: unknown[] = [];

      mockGet
        .mockResolvedValueOnce({ data: { data: stock } })
        .mockResolvedValueOnce({ data: { data: pendingReturns } });

      const { bulkReturnAllStock } = await import("../consignment-service");

      await expect(bulkReturnAllStock(1, 5)).rejects.toThrow(
        "No stock to return",
      );
    });

    it("resolves open pending returns before adding available stock", async () => {
      const stock = [
        {
          product_id: 10,
          arrangement_id: 1,
          available_qty: 4,
          pending_return_qty: 2,
        },
      ];
      const pendingReturns = [
        {
          id: 50,
          product_id: 10,
          qty: 2,
          status: "open",
          arrangement_id: 1,
        },
      ];

      mockGet
        .mockResolvedValueOnce({ data: { data: stock } })
        .mockResolvedValueOnce({ data: { data: pendingReturns } });

      mockPost.mockResolvedValueOnce({
        data: { data: { id: 1, items: [] } },
      });

      const { bulkReturnAllStock } = await import("../consignment-service");
      await bulkReturnAllStock(1, 5);

      const callBody = mockPost.mock.calls[0][1];
      // pending return item first, then available qty item
      expect(callBody.items).toHaveLength(2);
      expect(callBody.items[0]).toMatchObject({
        product_id: 10,
        qty: 2,
        reason: "termination",
        pending_return_id: 50,
      });
      expect(callBody.items[1]).toMatchObject({
        product_id: 10,
        qty: 4,
        reason: "termination",
      });
    });

    it("ignores closed pending returns", async () => {
      const stock = [
        {
          product_id: 10,
          arrangement_id: 1,
          available_qty: 3,
          pending_return_qty: 0,
        },
      ];
      const pendingReturns = [
        {
          id: 50,
          product_id: 10,
          qty: 2,
          status: "returned",
          arrangement_id: 1,
        },
      ];

      mockGet
        .mockResolvedValueOnce({ data: { data: stock } })
        .mockResolvedValueOnce({ data: { data: pendingReturns } });

      mockPost.mockResolvedValueOnce({
        data: { data: { id: 1, items: [] } },
      });

      const { bulkReturnAllStock } = await import("../consignment-service");
      await bulkReturnAllStock(1, 5);

      const callBody = mockPost.mock.calls[0][1];
      // only the available stock item, no pending return item
      expect(callBody.items).toHaveLength(1);
      expect(callBody.items[0]).toMatchObject({
        product_id: 10,
        qty: 3,
        reason: "termination",
      });
    });
  });

  describe("addTerm", () => {
    it("posts term payload and returns created term", async () => {
      const term = {
        id: 1,
        arrangement_id: 10,
        product_id: 42,
        price: 15000,
        store_share_type: "percentage",
        store_share_value: 25,
        product_name: "Test Product",
      };
      mockPost.mockResolvedValueOnce({ data: { data: term } });

      const { addTerm } = await import("../consignment-service");
      const result = await addTerm(10, {
        product_id: 42,
        price: 15000,
        store_share_type: "percentage",
        store_share_value: 25,
      });

      expect(result).toEqual(term);
      expect(mockPost).toHaveBeenCalledWith(
        "/consignment/arrangements/10/terms",
        {
          product_id: 42,
          price: 15000,
          store_share_type: "percentage",
          store_share_value: 25,
        },
      );
    });
  });

  describe("removeTerm", () => {
    it("sends delete request for the product term", async () => {
      mockDelete.mockResolvedValueOnce({});

      const { removeTerm } = await import("../consignment-service");
      await removeTerm(10, 42);

      expect(mockDelete).toHaveBeenCalledWith(
        "/consignment/arrangements/10/terms/42",
      );
    });
  });

  describe("searchAvailableProducts", () => {
    it("sends search query and returns products with exactMatch flag", async () => {
      mockGet.mockResolvedValueOnce({
        data: {
          data: [
            { id: 1, sku: "SKU-001", name: "Apple Juice" },
            { id: 2, sku: "SKU-002", name: "Apple Sauce" },
          ],
          exact_match: true,
        },
      });

      const { searchAvailableProducts } =
        await import("../consignment-service");
      const result = await searchAvailableProducts(10, "Apple");

      expect(result.products).toHaveLength(2);
      expect(result.exactMatch).toBe(true);
      expect(mockGet).toHaveBeenCalledWith(
        "/consignment/arrangements/10/available-products",
        { params: { search: "Apple" } },
      );
    });

    it("defaults exactMatch to false when not provided", async () => {
      mockGet.mockResolvedValueOnce({
        data: { data: [{ id: 3, sku: "SKU-003", name: "Banana" }] },
      });

      const { searchAvailableProducts } =
        await import("../consignment-service");
      const result = await searchAvailableProducts(10, "Ban");

      expect(result.products).toHaveLength(1);
      expect(result.exactMatch).toBe(false);
    });

    it("returns empty array when no products match", async () => {
      mockGet.mockResolvedValueOnce({
        data: { data: [], exact_match: false },
      });

      const { searchAvailableProducts } =
        await import("../consignment-service");
      const result = await searchAvailableProducts(10, "NonExistent");

      expect(result.products).toHaveLength(0);
      expect(result.exactMatch).toBe(false);
    });
  });

  describe("listSettlements", () => {
    it("fetches settlements without filters", async () => {
      const settlements = [
        { id: 1, settlement_number: "STL-001", status: "pending_payment" },
      ];
      mockGet.mockResolvedValueOnce({ data: { data: settlements } });

      const { listSettlements } = await import("../consignment-service");
      const result = await listSettlements();

      expect(result).toEqual(settlements);
      expect(mockGet).toHaveBeenCalledWith("/consignment/settlements");
    });

    it("fetches settlements with supplierId", async () => {
      const settlements = [
        { id: 1, settlement_number: "STL-001", status: "paid" },
      ];
      mockGet.mockResolvedValueOnce({ data: { data: settlements } });

      const { listSettlements } = await import("../consignment-service");
      const result = await listSettlements(5);

      expect(result).toEqual(settlements);
      expect(mockGet).toHaveBeenCalledWith(
        "/consignment/settlements?supplier_id=5",
      );
    });

    it("fetches settlements with status filter", async () => {
      const settlements = [
        { id: 1, settlement_number: "STL-001", status: "pending_payment" },
      ];
      mockGet.mockResolvedValueOnce({ data: { data: settlements } });

      const { listSettlements } = await import("../consignment-service");
      const result = await listSettlements(undefined, "pending_payment");

      expect(result).toEqual(settlements);
      expect(mockGet).toHaveBeenCalledWith(
        "/consignment/settlements?status=pending_payment",
      );
    });

    it("fetches settlements with both supplierId and status", async () => {
      const settlements = [
        { id: 1, settlement_number: "STL-001", status: "paid" },
      ];
      mockGet.mockResolvedValueOnce({ data: { data: settlements } });

      const { listSettlements } = await import("../consignment-service");
      const result = await listSettlements(5, "paid");

      expect(result).toEqual(settlements);
      expect(mockGet).toHaveBeenCalledWith(
        "/consignment/settlements?supplier_id=5&status=paid",
      );
    });
  });

  describe("listReceipts", () => {
    it("fetches receipts without productId", async () => {
      const receipts = [
        { id: 1, receipt_number: "RCP-001", items: [] },
        { id: 2, receipt_number: "RCP-002", items: [] },
      ];
      mockGet.mockResolvedValueOnce({ data: { data: receipts } });

      const { listReceipts } = await import("../consignment-service");
      const result = await listReceipts(5);

      expect(result).toEqual(receipts);
      expect(mockGet).toHaveBeenCalledWith(
        "/consignment/receipts?supplier_id=5",
      );
    });

    it("fetches receipts with productId", async () => {
      const receipts = [
        { id: 1, receipt_number: "RCP-001", items: [{ product_id: 42 }] },
      ];
      mockGet.mockResolvedValueOnce({ data: { data: receipts } });

      const { listReceipts } = await import("../consignment-service");
      const result = await listReceipts(5, 42);

      expect(result).toEqual(receipts);
      expect(mockGet).toHaveBeenCalledWith(
        "/consignment/receipts?supplier_id=5&product_id=42",
      );
    });

    it("returns empty array when no receipts match", async () => {
      mockGet.mockResolvedValueOnce({ data: { data: [] } });

      const { listReceipts } = await import("../consignment-service");
      const result = await listReceipts(5, 999);

      expect(result).toEqual([]);
      expect(mockGet).toHaveBeenCalledWith(
        "/consignment/receipts?supplier_id=5&product_id=999",
      );
    });
  });
});
