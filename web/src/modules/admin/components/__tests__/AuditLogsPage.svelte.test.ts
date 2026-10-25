import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "AuditLogsPage.svelte"),
    "utf-8",
  );
}

describe("AuditLogsPage.svelte source-structure guards", () => {
  const src = getSource();

  it("imports apiClient for HTTP calls", () => {
    expect(src).toContain('import apiClient from "$shared/api/http-client"');
  });

  it("imports RBAC composable", () => {
    expect(src).toContain(
      'import { useRBAC } from "$shared/composables/useRBAC.svelte"',
    );
  });

  it("imports Jakarta time utilities (remaining)", () => {
    expect(src).toContain("getTodayInJakarta");
    expect(src).toContain("getDateNDaysAgoInJakarta");
    expect(src).toContain("JAKARTA_OFFSET_MS");
    expect(src).toContain('from "$shared/utils/jakartaTime"');
  });

  it("uses $state for items, pagination, filters, and request tracking", () => {
    expect(src).toContain("let items = $state<AuditLog[]>");
    expect(src).toContain("let total = $state(0)");
    expect(src).toContain("let searchQuery = $state");
    expect(src).toContain("let currentRequestId = $state(0)");
    expect(src).toContain("let abortController");
    expect(src).toContain("$state");
  });

  it("has RBAC — only audit.view permission can view", () => {
    expect(src).toContain(
      "const canView = $derived(rbac.can(Permissions.audit.view))",
    );
  });

  it("has drawer open/close functions", () => {
    expect(src).toContain("function openDrawer(log");
    expect(src).toContain("function closeDrawer()");
  });

  it("has fetchLogs function", () => {
    expect(src).toContain("async function fetchLogs()");
  });

  it("has handlePageChange function", () => {
    expect(src).toContain("function handlePageChange(newOffset");
  });

  it("imports the three extracted child components", () => {
    expect(src).toContain(
      'import AuditLogsFilterToolbar from "./AuditLogsFilterToolbar.svelte"',
    );
    expect(src).toContain(
      'import AuditLogsTable from "./AuditLogsTable.svelte"',
    );
    expect(src).toContain(
      'import AuditLogDetailsDrawer from "./AuditLogDetailsDrawer.svelte"',
    );
  });

  it("renders AuditLogsFilterToolbar", () => {
    expect(src).toContain("<AuditLogsFilterToolbar");
  });

  it("renders AuditLogsTable", () => {
    expect(src).toContain("<AuditLogsTable");
  });

  it("renders AuditLogDetailsDrawer", () => {
    expect(src).toContain("<AuditLogDetailsDrawer");
  });

  it("does NOT contain extracted filter constants", () => {
    expect(src).not.toContain("const actionsMap");
  });

  it("does NOT contain extracted table template", () => {
    expect(src).not.toContain('class="empty-state-icon"');
  });

  it("does NOT contain extracted drawer template", () => {
    expect(src).not.toContain("getDiffDescription");
  });

  it("does NOT contain extracted export functions", () => {
    expect(src).not.toContain("function exportToCsv");
    expect(src).not.toContain("function exportToExcel");
    expect(src).not.toContain("function buildExportUrl");
  });

  it("re-points the captured prev-offset when the filter branch resets paging", () => {
    // The bottom-of-effect sync writes `prevOff = so`. If the filter branch
    // reset `offset` without re-pointing `so`, the sync would record the OLD
    // offset, the effect's re-run would see `pageChanged` and fire a duplicate
    // — immediately aborted — fetchLogs for the identical query (a wasted
    // request plus a loading flicker on every filter change made while paged
    // away from page 0).
    expect(src).toContain("let so = offset;");
    expect(src).toContain("const sl = limit;");
    // The reset and the re-point must be in the SAME branch, back to back…
    expect(src).toContain("offset = 0;\n      so = 0;");
    // …so the bottom-of-effect sync records the reset offset, not the old one.
    expect(src.indexOf("so = 0;")).toBeLessThan(src.indexOf("prevOff = so;"));
    // The old const-capture form is gone.
    expect(src).not.toContain("so = offset,\n");
  });

  it("resets prevOff in the debounce callback so a paged search fetches once", () => {
    // debouncedSearchFetch runs outside the effect: it writes `offset = 0`,
    // which re-runs the effect, and without a matching prevOff reset that
    // re-run takes the `pageChanged` branch and fires a duplicate — aborted —
    // fetchLogs for the identical query whenever a search starts from page > 0.
    expect(src).toContain("prevOff = 0;\n    offset = 0;\n    fetchLogs();");
  });

  it("imports i18n labels", () => {
    expect(src).toContain('import { labels } from "$shared/i18n"');
  });

  it("uses i18n labels for permission denied and error messages", () => {
    expect(src).toContain("labels.accessDenied");
    expect(src).toContain("labels.auditLogsRestrictedToSuperadmin");
    expect(src).toContain("labels.failedToLoad");
    expect(src).toContain("labels.unknown");
  });
});
