import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "NotificationBell.svelte"),
    "utf-8",
  );
}

describe("NotificationBell.svelte source-structure guards", () => {
  const src = getSource();

  it("imports routePermissions and RBAC composable", () => {
    expect(src).toContain(
      'import { routePermissions } from "$app/config/permissions"',
    );
    expect(src).toContain(
      'import { useRBAC } from "$shared/composables/useRBAC.svelte"',
    );
  });

  it("imports notification store and helpers", () => {
    expect(src).toContain('from "$shared/stores/notifications.svelte"');
    expect(src).toContain("notifications.markAsRead");
  });

  it("permission-gates navigation in handleNotificationClick", () => {
    expect(src).toContain('const targetPath = n.navigateTo.split("?")[0]');
    expect(src).toContain("const requiredPerms =");
    expect(src).toContain("routePermissions[targetPath]");
    expect(src).toContain(
      "if (requiredPerms && !rbac.canAny(requiredPerms)) return;",
    );
  });

  it("falls back to base path so detail deep-links (e.g. /stock-opnames/<id>) are permission-gated", () => {
    expect(src).toContain("?? routePermissions[basePath(targetPath)]");
    expect(src).toContain("function basePath(path: string)");
  });

  it("marks notification as read before navigating", () => {
    const fnStart = src.indexOf("function handleNotificationClick");
    const fnEnd = src.indexOf("}", src.indexOf("goto(n.navigateTo)", fnStart));
    const fnBody = src.slice(fnStart, fnEnd);
    const markIdx = fnBody.indexOf("notifications.markAsRead(n.id)");
    const gotoIdx = fnBody.indexOf("goto(n.navigateTo)");
    expect(markIdx).toBeGreaterThan(-1);
    expect(gotoIdx).toBeGreaterThan(markIdx);
  });

  it("stock_update and product_updated notifications deep link with product_id", () => {
    expect(src).toContain(
      "navigateTo: `/inventory/products?product_id=${data.id}`",
    );
  });

  it("low_stock alert navigates with low_stock filter param", () => {
    expect(src).toContain('navigateTo: "/inventory/products?low_stock=true"');
  });

  it("stock opname notifications gated by stock_opname.view permission at receive time", () => {
    expect(src).toContain("canReceiveStockOpnameNotifications");
    expect(src).toContain("if (!canSeeStockOpname) return;");
  });

  it("gates the return-approval request on the approve permission at receive time", () => {
    expect(src).toContain("canReceiveReturnApprovalNotifications");
    expect(src).toContain("if (!canSeeReturnApprovals) return;");
    expect(src).toContain('ws.on(\n        "return_approval_requested"');
  });

  it("notifies the cashier of a resolution and deep-links into the queue", () => {
    expect(src).toContain('ws.on(\n        "return_approval_resolved"');
    expect(src).toContain('if (canSeeReturnApprovals) return;');
    expect(src).toContain('navigateTo: "/transactions?return_approvals=true"');
  });

  it("branches the resolution copy on the terminal status", () => {
    // completed → approved, cancelled → rejected, expired → expired.
    expect(src).toContain('data.status === "completed"');
    expect(src).toContain('data.status === "cancelled"');
    expect(src).toContain('data.status === "expired"');
    expect(src).toContain("labels.returnApprovalApprovedTitle");
    expect(src).toContain("labels.returnApprovalRejectedTitle");
    expect(src).toContain("labels.returnApprovalExpiredTitle");
  });
});
