import { describe, it, expect, vi, beforeEach } from "vitest";

const mockGet = vi.fn();
const mockPost = vi.fn();

vi.mock("$shared/api/http-client", () => ({
  default: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
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

      const { bulkReturnAllStock } = await import(
        "../consignment-service"
      );
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

      const { bulkReturnAllStock } = await import(
        "../consignment-service"
      );

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

      const { bulkReturnAllStock } = await import(
        "../consignment-service"
      );

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

      const { bulkReturnAllStock } = await import(
        "../consignment-service"
      );
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

      const { bulkReturnAllStock } = await import(
        "../consignment-service"
      );
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
});
