import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { setLocale } from "$shared/i18n";

describe("product-utils", () => {
  beforeAll(() => setLocale("id"));
  afterAll(() => setLocale("en"));

  describe("statusInfo", () => {
    it("returns success for active status", async () => {
      const { statusInfo } = await import("../product-utils");
      expect(statusInfo("active")).toEqual({
        variant: "success",
        label: "Active",
      });
    });

    it("returns muted for draft", async () => {
      const { statusInfo } = await import("../product-utils");
      expect(statusInfo("draft")).toEqual({ variant: "muted", label: "Draft" });
    });

    it("returns muted for inactive", async () => {
      const { statusInfo } = await import("../product-utils");
      expect(statusInfo("inactive")).toEqual({
        variant: "muted",
        label: "Inactive",
      });
    });

    it("returns danger for discontinued", async () => {
      const { statusInfo } = await import("../product-utils");
      expect(statusInfo("discontinued")).toEqual({
        variant: "danger",
        label: "Discontinued",
      });
    });

    it("returns danger for archived", async () => {
      const { statusInfo } = await import("../product-utils");
      expect(statusInfo("archived")).toEqual({
        variant: "danger",
        label: "Archived",
      });
    });

    it("returns muted for unknown status", async () => {
      const { statusInfo } = await import("../product-utils");
      expect(statusInfo("unknown")).toEqual({ variant: "muted", label: "- " });
    });

    it("handles undefined status", async () => {
      const { statusInfo } = await import("../product-utils");
      expect(statusInfo()).toEqual({ variant: "muted", label: "- " });
    });
  });

  describe("formatCurrency", () => {
    it("formats number as IDR", async () => {
      const { formatCurrency } = await import("../product-utils");
      expect(formatCurrency(15000)).toBe("Rp 15.000");
    });

    it("returns dash for null", async () => {
      const { formatCurrency } = await import("../product-utils");
      expect(formatCurrency(null as unknown as number)).toBe("-");
    });

    it("returns dash for NaN", async () => {
      const { formatCurrency } = await import("../product-utils");
      expect(formatCurrency(NaN)).toBe("-");
    });
  });
});
