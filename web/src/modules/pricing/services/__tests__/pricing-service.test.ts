import { describe, it, expect, vi, beforeEach } from "vitest";

const mockApiFetch = vi.fn();

vi.mock("$shared/api/http-client", () => ({
  apiFetch: (...args: unknown[]) => mockApiFetch(...args),
}));

describe("pricing-service", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("approve/reject", () => {
    it("resolves without throwing on success", async () => {
      mockApiFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({}),
      });

      const { approvePricingRule, rejectPricingRule } =
        await import("../pricing-service");

      await expect(approvePricingRule(1)).resolves.toBeUndefined();
      await expect(rejectPricingRule(1)).resolves.toBeUndefined();
    });

    // The self-approval check answers 403 with a reason; returning a bare
    // boolean used to discard it and show a generic toast instead.
    it("throws the server's reason so the UI can surface it", async () => {
      mockApiFetch.mockResolvedValue({
        ok: false,
        json: () =>
          Promise.resolve({
            error: "you cannot approve your own pricing rule",
          }),
      });

      const { approvePricingRule } = await import("../pricing-service");

      await expect(approvePricingRule(1)).rejects.toThrow(
        "you cannot approve your own pricing rule",
      );
    });

    it("reads a nested error object", async () => {
      mockApiFetch.mockResolvedValue({
        ok: false,
        json: () => Promise.resolve({ error: { message: "rule not found" } }),
      });

      const { rejectPricingRule } = await import("../pricing-service");

      await expect(rejectPricingRule(1)).rejects.toThrow("rule not found");
    });

    it("falls back when the body is not JSON", async () => {
      mockApiFetch.mockResolvedValue({
        ok: false,
        json: () => Promise.reject(new Error("not json")),
      });

      const { approvePricingRule } = await import("../pricing-service");

      await expect(approvePricingRule(1)).rejects.toThrow(
        "Failed to approve pricing rule",
      );
    });
  });

  describe("getPricingRules", () => {
    it("builds basic query params", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: [{ id: 1 }], total: 1 }),
      });

      const { getPricingRules } = await import("../pricing-service");
      const result = await getPricingRules({ limit: 20, offset: 0 });

      expect(mockApiFetch).toHaveBeenCalled();
      const url = mockApiFetch.mock.calls[0][0] as string;
      expect(url).toContain("limit=20");
      expect(url).toContain("offset=0");
      expect(result.data).toHaveLength(1);
      expect(result.total).toBe(1);
    });

    it("includes search and type filters", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: [], total: 0 }),
      });

      const { getPricingRules } = await import("../pricing-service");
      await getPricingRules({
        limit: 10,
        offset: 0,
        search: "diskon",
        pricing_type: "promotion",
      });

      const url = mockApiFetch.mock.calls[0][0] as string;
      expect(url).toContain("search=diskon");
      expect(url).toContain("pricing_type=promotion");
    });

    it("includes category and brand filters", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: [], total: 0 }),
      });

      const { getPricingRules } = await import("../pricing-service");
      await getPricingRules({
        limit: 10,
        offset: 0,
        category_id: 5,
        brand_id: 3,
      });

      const url = mockApiFetch.mock.calls[0][0] as string;
      expect(url).toContain("category_id=5");
      expect(url).toContain("brand_id=3");
    });

    it("includes store and customer_group filters", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: [], total: 0 }),
      });

      const { getPricingRules } = await import("../pricing-service");
      await getPricingRules({
        limit: 10,
        offset: 0,
        store_id: 1,
        customer_group_id: 2,
        is_active: true,
      });

      const url = mockApiFetch.mock.calls[0][0] as string;
      expect(url).toContain("store_id=1");
      expect(url).toContain("customer_group_id=2");
      expect(url).toContain("is_active=true");
    });

    it("throws on error rather than reporting an empty page", async () => {
      // `{ data: [], total: 0 }` is a valid successful answer, so returning it on
      // failure made a 403/500 indistinguishable from "no rules match these filters".
      mockApiFetch.mockResolvedValueOnce({
        ok: false,
        json: () => Promise.resolve({ error: { message: "Forbidden" } }),
      });

      const { getPricingRules } = await import("../pricing-service");
      await expect(getPricingRules({ limit: 10, offset: 0 })).rejects.toThrow(
        "Forbidden",
      );
    });

    it("returns an empty page for a genuine no-match", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: [], total: 0 }),
      });

      const { getPricingRules } = await import("../pricing-service");
      const result = await getPricingRules({ limit: 10, offset: 0 });
      expect(result.data).toEqual([]);
      expect(result.total).toBe(0);
    });
  });

  describe("createPricingRule", () => {
    it("sends POST with correct payload", async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true });

      const { createPricingRule } = await import("../pricing-service");
      const result = await createPricingRule({
        product_id: 1,
        pricing_type: "promotion",
        pricing_method: "discount_percent",
        pricing_value: 10,
        name: "Test Rule",
        minimum_quantity: 1,
        priority: 0,
        is_active: true,
      });

      expect(result).toEqual({ ok: true });
      expect(mockApiFetch).toHaveBeenCalledWith(
        "/api/pricing-rules",
        expect.objectContaining({
          method: "POST",
        }),
      );
      const body = JSON.parse(mockApiFetch.mock.calls[0][1].body);
      expect(body.pricing_type).toBe("promotion");
      expect(body.pricing_method).toBe("discount_percent");
      expect(body.pricing_value).toBe(10);
    });

    it("returns error object on failure", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: false,
        json: () => Promise.resolve({ error: "nama rule sudah digunakan" }),
      });

      const { createPricingRule } = await import("../pricing-service");
      const result = await createPricingRule({
        product_id: 1,
        pricing_type: "promotion",
        pricing_method: "discount_percent",
        pricing_value: 10,
        name: "Duplicate Rule",
        minimum_quantity: 1,
        priority: 0,
        is_active: true,
      });
      expect(result.ok).toBe(false);
      expect(result.error).toBe("nama rule sudah digunakan");
    });

    it("returns default error when json parse fails", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: false,
        json: () => Promise.reject(new Error("not json")),
      });

      const { createPricingRule } = await import("../pricing-service");
      const result = await createPricingRule({
        product_id: 1,
        pricing_type: "promotion",
        pricing_method: "discount_percent",
        pricing_value: 10,
        name: "Fail Rule",
        minimum_quantity: 1,
        priority: 0,
        is_active: true,
      });
      expect(result.ok).toBe(false);
      expect(result.error).toBe("Gagal menyimpan rule");
    });
  });

  describe("resolvePrices", () => {
    it("sends items and returns resolved prices", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () =>
          Promise.resolve({
            data: [
              {
                unit_price: 90000,
                original_price: 100000,
                discount: 10000,
                pricing_type: "promotion",
              },
            ],
          }),
      });

      const { resolvePrices } = await import("../pricing-service");
      const result = await resolvePrices([{ product_id: 1, quantity: 1 }]);

      expect(result).toHaveLength(1);
      expect(result[0].unit_price).toBe(90000);
      expect(result[0].pricing_type).toBe("promotion");
    });

    it("throws on error rather than reporting an empty match", async () => {
      // An empty array means "no rule matched" to the simulation modal, so a
      // failure must not be flattened into it: a failed request would otherwise be
      // reported to the manager as the authoritative conclusion that the rule does
      // not apply.
      mockApiFetch.mockResolvedValueOnce({
        ok: false,
        json: () => Promise.resolve({ error: { message: "Forbidden" } }),
      });

      const { resolvePrices } = await import("../pricing-service");
      await expect(
        resolvePrices([{ product_id: 1, quantity: 1 }]),
      ).rejects.toThrow("Forbidden");
    });

    it("returns an empty list for a genuine no-match", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: [] }),
      });

      const { resolvePrices } = await import("../pricing-service");
      const result = await resolvePrices([{ product_id: 1, quantity: 1 }]);
      expect(result).toEqual([]);
    });
  });

  describe("searchProducts", () => {
    it("builds search URL with query", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () =>
          Promise.resolve({
            data: [{ id: 1, name: "Indomie", sku: "IND-001", price: 3500 }],
          }),
      });

      const { searchProducts } = await import("../pricing-service");
      const result = await searchProducts("indomie");

      const url = mockApiFetch.mock.calls[0][0] as string;
      expect(url).toContain("q=indomie");
      expect(result).toHaveLength(1);
      expect(result[0].name).toBe("Indomie");
    });
  });

  describe("getCustomerGroups", () => {
    it("returns customer groups", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () =>
          Promise.resolve({
            data: [
              { id: 1, name: "Walk-in" },
              { id: 2, name: "Member" },
            ],
          }),
      });

      const { getCustomerGroups } = await import("../pricing-service");
      const result = await getCustomerGroups();
      expect(result).toHaveLength(2);
      expect(result[0].name).toBe("Walk-in");
    });
  });

  describe("getStores", () => {
    it("returns active stores", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: [{ id: 1, name: "Main Store" }] }),
      });

      const { getStores } = await import("../pricing-service");
      const result = await getStores();
      expect(result).toHaveLength(1);
      expect(result[0].name).toBe("Main Store");
    });
  });

  describe("getPricingRule", () => {
    it("returns single rule", async () => {
      const rule = { id: 1, name: "Rule 1", pricing_type: "promotion" };
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: rule }),
      });

      const { getPricingRule } = await import("../pricing-service");
      const result = await getPricingRule(1);

      expect(mockApiFetch).toHaveBeenCalledWith("/api/pricing-rules/1");
      expect(result).toEqual(rule);
    });

    it("returns null on error", async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: false });

      const { getPricingRule } = await import("../pricing-service");
      const result = await getPricingRule(1);
      expect(result).toBeNull();
    });
  });

  describe("updatePricingRule", () => {
    it("sends PUT with correct payload", async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true });

      const { updatePricingRule } = await import("../pricing-service");
      const payload = { pricing_value: 20, name: "Updated Rule" };
      const result = await updatePricingRule(1, payload);

      expect(result).toEqual({ ok: true });
      expect(mockApiFetch).toHaveBeenCalledWith(
        "/api/pricing-rules/1",
        expect.objectContaining({
          method: "PUT",
        }),
      );
      const body = JSON.parse(mockApiFetch.mock.calls[0][1].body);
      expect(body.pricing_value).toBe(20);
      expect(body.name).toBe("Updated Rule");
    });

    it("returns error object on failure", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: false,
        json: () => Promise.resolve({ error: "nama rule sudah digunakan" }),
      });

      const { updatePricingRule } = await import("../pricing-service");
      const result = await updatePricingRule(1, { pricing_value: 20 });
      expect(result.ok).toBe(false);
      expect(result.error).toBe("nama rule sudah digunakan");
    });

    it("returns default error when json parse fails", async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: false,
        json: () => Promise.reject(new Error("not json")),
      });

      const { updatePricingRule } = await import("../pricing-service");
      const result = await updatePricingRule(1, { pricing_value: 20 });
      expect(result.ok).toBe(false);
      expect(result.error).toBe("Gagal menyimpan rule");
    });
  });

  describe("deletePricingRule", () => {
    it("sends DELETE request", async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true });

      const { deletePricingRule } = await import("../pricing-service");
      const result = await deletePricingRule(1);

      expect(result).toBe(true);
      expect(mockApiFetch).toHaveBeenCalledWith("/api/pricing-rules/1", {
        method: "DELETE",
      });
    });

    it("returns false on error", async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: false });

      const { deletePricingRule } = await import("../pricing-service");
      const result = await deletePricingRule(1);
      expect(result).toBe(false);
    });
  });
});
