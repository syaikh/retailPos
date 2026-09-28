import { describe, it, expect } from "vitest";
import { getDefaultRoute } from "../default-route";

describe("getDefaultRoute", () => {
  it("sends cashiers to the shift screen", () => {
    expect(getDefaultRoute({ role: "cashier" })).toBe("/shifts");
  });

  it("sends inventory staff to stock opname, not the products page", () => {
    // inventory_staff has stock_opname.view but no product.view (migration 044)
    expect(getDefaultRoute({ role: "inventory_staff" })).toBe("/stock-opnames");
  });

  it("sends the remaining roles to the dashboard", () => {
    for (const role of [
      "superadmin",
      "manager",
      "supervisor",
      "finance",
      "custom_role",
    ]) {
      expect(getDefaultRoute({ role })).toBe("/");
    }
  });

  it("accepts role objects and missing users", () => {
    expect(getDefaultRoute({ role: { name: "inventory_staff" } })).toBe(
      "/stock-opnames",
    );
    expect(getDefaultRoute(null)).toBe("/");
    expect(getDefaultRoute(undefined)).toBe("/");
  });
});
