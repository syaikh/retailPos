import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "StoresPage.svelte"),
    "utf-8",
  );
}

describe("StoresPage.svelte source-structure guards", () => {
  const src = getSource();

  it("imports Jakarta time utility", () => {
    expect(src).toContain(
      'import { formatDateInJakarta } from "$shared/utils/jakartaTime"',
    );
  });

  it("imports store service functions", () => {
    expect(src).toContain(
      'import {\n    getStores,\n    updateStore,\n    deleteStore,\n    getReadiness,\n  } from "../services/stores-service"',
    );
    expect(src).toContain(
      'import StoreOnboardingWizard from "./StoreOnboardingWizard.svelte"',
    );
  });

  it("imports shared UI components", () => {
    expect(src).toContain(
      'import {\n    Button,\n    Input,\n    Modal,\n    Skeleton,\n    BulkActionDropdown,\n    ImportWizard,\n    SearchBar,\n    ToggleSwitch,\n    ConfirmDeleteModal,\n    Pagination,\n    SortableHeader,\n    Badge,\n  } from "$shared/ui"',
    );
  });

  it("imports i18n labels", () => {
    expect(src).toContain('import { labels, t } from "$shared/i18n"');
  });

  it("uses $state for stores, loading, pagination", () => {
    expect(src).toContain("let loading = $state(true)");
    expect(src).toContain("let stores = $state([])");
    expect(src).toContain("let total = $state(0)");
    expect(src).toContain("let searchQuery = $state");
    expect(src).toContain("let statusFilter = $state");
  });

  it("has RBAC derived from the shared composable", () => {
    expect(src).toContain("const rbac = useRBAC()");
    expect(src).toContain(
      "const canCreate = $derived(rbac.can(Permissions.store.create))",
    );
    expect(src).toContain(
      "const canEdit = $derived(rbac.can(Permissions.store.update))",
    );
    expect(src).toContain(
      "const canDelete = $derived(rbac.can(Permissions.store.delete))",
    );
  });

  it("has sort state and handleSort function", () => {
    expect(src).toContain(
      'const { sortState, handleSort } = useSortable("name", "asc")',
    );
    expect(src).toContain("sortState.sortBy");
    expect(src).toContain("sortState.sortDir");
  });

  it("has fetchStores, openAdd, openEdit, saveStore functions", () => {
    expect(src).toContain("async function fetchStores");
    expect(src).toContain("function openAdd");
    expect(src).toContain("function openEdit");
    expect(src).toContain("async function saveStore");
  });

  it("has status filter chips", () => {
    expect(src).toContain("labels.all");
    expect(src).toContain("labels.active");
    expect(src).toContain("labels.inactive");
    expect(src).toContain("function setStatusFilter");
  });

  it("renders BulkActionDropdown with module stores", () => {
    expect(src).toContain('module="stores"');
    expect(src).toContain("<BulkActionDropdown");
  });

  it("renders ImportWizard with module stores", () => {
    expect(src).toContain("<ImportWizard");
    expect(src).toContain('module="stores"');
    expect(src).toContain("displayName={labels.stores}");
  });

  it("renders Pagination component", () => {
    expect(src).toContain("<Pagination");
  });

  it("opens the onboarding wizard instead of the plain add modal", () => {
    expect(src).toContain("let showWizard = $state(false)");
    expect(src).toContain("showWizard = true");
    expect(src).toContain("<StoreOnboardingWizard");
    expect(src).toContain("bind:open={showWizard}");
    expect(src).toContain("onComplete={() => fetchStores()}");
  });

  it("tracks per-store readiness and renders a badge", () => {
    expect(src).toContain("let readinessMap = $state({})");
    expect(src).toContain("async function fetchReadiness(ids)");
    expect(src).toContain("const r = await getReadiness(id)");
    expect(src).toContain("readinessMap[store.id]");
    expect(src).toContain('<Badge variant="success" size="sm">');
    expect(src).toContain('<Badge variant="warning" size="sm">');
    expect(src).toContain("labels.onboardingStepReadiness");
  });

  it("caches readiness verdicts behind a TTL instead of refetching every row", () => {
    // limit can be 100, so an uncached refetch fired up to 100 readiness
    // calls on a single page change / filter / search.
    expect(src).toContain("const READINESS_TTL_MS = 30_000;");
    // SvelteMap, not the built-in Map: the svelte lint rule forbids mutable
    // built-in Maps inside components (the CI `Frontend: Lint` annotation).
    expect(src).toContain('import { SvelteMap } from "svelte/reactivity";');
    expect(src).toContain("const readinessFetchedAt = new SvelteMap();");
    expect(src).toContain("const stale = ids.filter((id) => {");
    expect(src).toContain(
      "return at === undefined || now - at > READINESS_TTL_MS;",
    );
    expect(src).toContain("if (stale.length === 0) return;");
    expect(src).toContain("stale.map(async (id) => {");
    expect(src).toContain("readinessFetchedAt.set(id, Date.now());");
  });

  it("does not stamp a readiness fetch that failed, so it is retried", () => {
    // Caching the miss would leave the skeleton resolved to "unknown" for the
    // whole TTL even though a plain retry could answer it.
    const tryIdx = src.indexOf("const r = await getReadiness(id)");
    const catchIdx = src.indexOf("} catch {", tryIdx);
    expect(tryIdx).toBeGreaterThan(-1);
    expect(catchIdx).toBeGreaterThan(tryIdx);
    const catchBlock = src.slice(catchIdx, src.indexOf("}),", catchIdx));
    expect(catchBlock).toContain("readinessMap[id] = null;");
    expect(catchBlock).not.toContain("readinessFetchedAt.set");
  });

  it("drops the cached readiness stamp when a store is updated", () => {
    // Toggling is_active (or any attribute) can change the verdict.
    expect(src).toContain("readinessFetchedAt.delete(selectedStore.id);");
  });
});
