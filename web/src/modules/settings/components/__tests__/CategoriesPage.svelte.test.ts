import { describe, it, expect } from "vitest";
import { fileURLToPath } from "node:url";
import { readFileSync } from "node:fs";
import path from "node:path";

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(
    path.join(path.dirname(__filename), "..", "CategoriesPage.svelte"),
    "utf-8",
  );
}

describe("CategoriesPage.svelte source-structure guards", () => {
  const src = getSource();

  it("imports auth store", () => {
    expect(src).toContain('import { useAuthStore } from "$modules/auth"');
  });

  it("imports Jakarta time utility", () => {
    expect(src).toContain('import { formatDateInJakarta } from "$shared/utils/jakartaTime"');
  });

  it("imports apiFetch", () => {
    expect(src).toContain('import { apiFetch } from "$shared/api/http-client"');
  });

  it("imports shared UI components", () => {
    expect(src).toContain(
      'import {\n    Button,\n    Input,\n    Modal,\n    Pagination,\n    SearchBar,\n    Skeleton,\n    BulkActionDropdown,\n    ImportWizard,\n    ToggleSwitch,\n    ConfirmDeleteModal,\n    SortableHeader,\n  } from "$shared/ui"',
    );
  });

  it("uses $state for categories, loading, pagination", () => {
    expect(src).toContain("let loading = $state(true)");
    expect(src).toContain("let categories = $state([])");
    expect(src).toContain("let total = $state(0)");
    expect(src).toContain("let searchQuery = $state");
  });

  it("has RBAC derived from the shared composable", () => {
    expect(src).toContain("const rbac = useRBAC()");
    expect(src).toContain("const canCreate = $derived(rbac.can(Permissions.category.create))");
    expect(src).toContain("const canEdit = $derived(rbac.can(Permissions.category.update))");
    expect(src).toContain("const canDelete = $derived(rbac.can(Permissions.category.delete))");
    expect(src).toContain("const _canView = $derived(authStore.user != null)");
  });

  it("has sort state and handleSort function", () => {
    expect(src).toContain('const { sortState, handleSort } = useSortable("name", "asc")');
    expect(src).toContain("sortState.sortBy");
    expect(src).toContain("sortState.sortDir");
  });

  it("has fetchCategories, openAdd, openEdit, saveCategory functions", () => {
    expect(src).toContain("async function fetchCategories");
    expect(src).toContain("function openAdd");
    expect(src).toContain("function openEdit");
    expect(src).toContain("async function saveCategory");
  });

  it("renders Pagination component", () => {
    expect(src).toContain("<Pagination");
  });
});
