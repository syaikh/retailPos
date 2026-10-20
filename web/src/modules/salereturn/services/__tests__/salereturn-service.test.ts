import { describe, it, expect, vi, beforeEach } from "vitest";

const mockGet = vi.fn();
const mockPost = vi.fn();

vi.mock("$shared/api/http-client", () => ({
  default: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
  },
}));

describe("salereturn-service", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("getReturnableSale", () => {
    it("GETs the pre-flight snapshot for the sale", async () => {
      const snapshot = { sale_id: 7, eligible: true, items: [] };
      mockGet.mockResolvedValueOnce({ data: { data: snapshot } });

      const { getReturnableSale } = await import("../salereturn-service");
      const result = await getReturnableSale(7);

      expect(result).toEqual(snapshot);
      expect(mockGet).toHaveBeenCalledWith("/sales/7/returnable");
    });

    it("returns null when the payload is missing", async () => {
      mockGet.mockResolvedValueOnce({ data: {} });

      const { getReturnableSale } = await import("../salereturn-service");
      expect(await getReturnableSale(7)).toBeNull();
    });
  });

  describe("processReturn", () => {
    it("POSTs the request body verbatim and unwraps data", async () => {
      const created = {
        id: 1,
        return_number: "RET-2601-0001",
        status: "completed",
      };
      mockPost.mockResolvedValueOnce({ data: { data: created } });

      const { processReturn } = await import("../salereturn-service");
      const result = await processReturn(7, {
        reason: "damaged",
        condition: "damaged",
        items: [{ sale_item_id: 3, quantity: 1 }],
      });

      expect(result).toEqual(created);
      expect(mockPost).toHaveBeenCalledWith("/sales/7/return", {
        reason: "damaged",
        condition: "damaged",
        items: [{ sale_item_id: 3, quantity: 1 }],
      });
    });
  });

  describe("listReturnsBySale", () => {
    it("requests a single max-size page and defaults to an empty list", async () => {
      mockGet.mockResolvedValueOnce({ data: { data: [] } });

      const { listReturnsBySale } = await import("../salereturn-service");
      expect(await listReturnsBySale(7)).toEqual([]);
      expect(mockGet).toHaveBeenCalledWith("/sales/7/returns", {
        params: { limit: 100 },
      });
    });
  });

  describe("getReturnById", () => {
    it("GETs one return's detail", async () => {
      const detail = { id: 9, return_number: "RET-2601-0001" };
      mockGet.mockResolvedValueOnce({ data: { data: detail } });

      const { getReturnById } = await import("../salereturn-service");
      expect(await getReturnById(9)).toEqual(detail);
      expect(mockGet).toHaveBeenCalledWith("/sales/returns/9");
    });
  });

  describe("listPendingApprovals", () => {
    it("GETs the approval queue from the pending endpoint", async () => {
      const queue = [{ id: 9, status: "pending_approval" }];
      mockGet.mockResolvedValueOnce({ data: { data: queue } });

      const { listPendingApprovals } = await import("../salereturn-service");
      expect(await listPendingApprovals()).toEqual(queue);
      expect(mockGet).toHaveBeenCalledWith("/sales/returns/pending", {
        params: { limit: 100 },
      });
    });
  });

  describe("approveReturn", () => {
    it("POSTs an optional note as { note }", async () => {
      const approved = { id: 9, status: "completed" };
      mockPost.mockResolvedValueOnce({ data: { data: approved } });

      const { approveReturn } = await import("../salereturn-service");
      expect(await approveReturn(9, "ok")).toEqual(approved);
      expect(mockPost).toHaveBeenCalledWith("/sales/returns/9/approve", {
        note: "ok",
      });
    });

    it("serialises a missing note as null", async () => {
      mockPost.mockResolvedValueOnce({ data: { data: { id: 9 } } });

      const { approveReturn } = await import("../salereturn-service");
      await approveReturn(9);
      expect(mockPost).toHaveBeenCalledWith("/sales/returns/9/approve", {
        note: null,
      });
    });
  });

  describe("rejectReturn", () => {
    it("POSTs the mandatory rejection reason", async () => {
      const rejected = { id: 9, status: "cancelled" };
      mockPost.mockResolvedValueOnce({ data: { data: rejected } });

      const { rejectReturn } = await import("../salereturn-service");
      expect(await rejectReturn(9, "barang tidak rusak")).toEqual(rejected);
      expect(mockPost).toHaveBeenCalledWith("/sales/returns/9/reject", {
        reason: "barang tidak rusak",
      });
    });
  });
});
