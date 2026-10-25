import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "Home.svelte"),
    "utf-8",
  );
}

describe("Home.svelte source-structure guards", () => {
  const src = getSource();

  it("imports onMount from svelte", () => {
    expect(src).toContain('import { onMount } from "svelte"');
  });

  it("imports goto from $app/router", () => {
    expect(src).toContain('import { goto } from "$app/router"');
  });

  it("imports apiFetch from shared/api", () => {
    expect(src).toContain('import { apiFetch } from "$shared/api/http-client"');
  });

  it("imports StatCard from shared/ui", () => {
    expect(src).toContain('import { StatCard, RpIcon } from "$shared/ui"');
  });

  it("imports i18n labels", () => {
    expect(src).toContain('import { labels } from "$shared/i18n"');
  });

  it("imports auth store from $shared", () => {
    expect(src).toContain(
      'import { useAuthStore } from "$shared/stores/auth.svelte"',
    );
  });

  it("uses $state for dashboard data", () => {
    expect(src).toContain("let todaysRevenue = $state");
    expect(src).toContain("let todaysSales = $state");
    expect(src).toContain("let outOfStockCount = $state");
    expect(src).toContain("let categoriesCount = $state");
    expect(src).toContain("let loading = $state");
    expect(src).toContain("let wsConnected = $state");
  });

  it("uses $derived for revenue subtitle", () => {
    expect(src).toContain("const revSubText = $derived");
  });

  it("has fetchLiveStats function calling dashboard API", () => {
    expect(src).toContain("async function fetchLiveStats");
    expect(src).toContain("/api/dashboard/live");
  });

  it("sets up WebSocket and interval in onMount", () => {
    expect(src).toContain("fetchLiveStats()");
    expect(src).toContain("ws.on");
  });

  it("renders StatCard for revenue and transactions", () => {
    expect(src).toContain("labels.todayRevenue");
    expect(src).toContain("labels.transactionsCard");
  });

  it("has role-based stat cards for categories and out-of-stock", () => {
    expect(src).toContain("categoriesCount");
    expect(src).toContain("outOfStockCount");
    expect(src).toContain("visibleStatCards");
  });

  it("has Quick Access modules section with permission filtering", () => {
    expect(src).toContain("labels.quickAccess");
    expect(src).toContain("labels.pointOfSale");
    expect(src).toContain("labels.inventory");
    expect(src).toContain("labels.reports");
    expect(src).toContain("labels.administration");
    expect(src).toContain("rbac.canAny");
  });

  it("has finance quick access for pending settlements", () => {
    expect(src).toContain("financeQuickAccess");
    expect(src).toContain("consignment.view");
    expect(src).toContain("PendingSettlementsModal");
  });

  it("has WebSocket live status indicator", () => {
    expect(src).toContain("wsConnected ? labels.live : labels.offline");
  });

  it("categoriesCount stat card is gated by showCategories for superadmin/manager", () => {
    expect(src).toContain("showCategories");
    expect(src).toContain('role === "superadmin" || role === "manager"');
  });

  it("outOfStockCount stat card is gated by showOutOfStock for superadmin/manager/supervisor", () => {
    expect(src).toContain("showOutOfStock");
    expect(src).toContain(
      'role === "superadmin" || role === "manager" || role === "supervisor"',
    );
  });

  it("source uses visibleStatCards.showCategories conditional in template", () => {
    expect(src).toContain("{#if visibleStatCards.showCategories}");
  });

  it("source uses visibleStatCards.showOutOfStock conditional in template", () => {
    expect(src).toContain("{#if visibleStatCards.showOutOfStock}");
  });

  it("quick access modules array uses the shared rbac.canAny for filtering", () => {
    expect(src).toContain(".filter((m) => rbac.canAny(m.required))");
  });

  it("source uses the shared RBAC composable instead of a local permission helper", () => {
    expect(src).toContain(
      'import { useRBAC } from "$shared/composables/useRBAC.svelte"',
    );
    expect(src).toContain("const rbac = useRBAC()");
  });

  it("source imports PendingSettlementsModal from consignment module", () => {
    expect(src).toContain(
      'import { PendingSettlementsModal } from "$modules/consignment"',
    );
  });

  it("financeQuickAccess is a $derived based on the consignment route permission", () => {
    expect(src).toContain("const financeQuickAccess = $derived(");
    expect(src).toContain("rbac.can(Permissions.consignment.view)");
  });

  it("source declares showPendingSettlements state for modal binding", () => {
    expect(src).toContain("let showPendingSettlements = $state(false)");
  });

  it("source binds showPendingSettlements to PendingSettlementsModal in template", () => {
    expect(src).toContain("bind:show={showPendingSettlements}");
  });
});

describe("permission gating", () => {
  const src = getSource();

  it("gates the Administration card on the same permission as the /admin route", () => {
    expect(src).toContain("required: [Permissions.user.view]");
  });

  it("does not define a second, local permission checker", () => {
    expect(src).not.toContain("function hasPermission");
    expect(src).not.toContain("function hasAnyPermission");
  });

  it("imports the shared Permissions registry", () => {
    expect(src).toContain(
      'import { Permissions } from "$shared/constants/permissions"',
    );
  });
});
