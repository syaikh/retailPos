<script lang="ts">
  import { onMount } from "svelte";
  import { SvelteURLSearchParams } from "svelte/reactivity";
  import { toast } from "$shared/stores/toast.svelte";
  import { useAuthStore } from "$modules/auth";
  import { goto } from "$app/router";
  import { labels } from "$shared/i18n";
  import { useWebSocket } from "$shared/api/websocket";
  import {
    getSuppliers,
    getSupplier,
    getSupplierUsage,
    createSupplier,
    updateSupplier,
    deleteSupplier,
    bulkUpdateSuppliers,
    bulkDeleteSuppliers,
  } from "../services/supplier-service";
  import type { SupplierListParams } from "../services/supplier-service";
  import type {
    Supplier,
    CreateSupplierPayload,
    UpdateSupplierPayload,
    SupplierUsage,
  } from "../types";
  import { Pagination, Modal, Button } from "$shared/ui";
  import { ArrowLeft } from "lucide-svelte";
  import { debounce } from "$shared/utils/debounce";
  import { useSortable } from "$shared/composables/useSortable.svelte";
  import SuppliersToolbar from "./SuppliersToolbar.svelte";
  import SuppliersTable from "./SuppliersTable.svelte";
  import SupplierFormModal from "./SupplierFormModal.svelte";
  import SupplierDetailDrawer from "./SupplierDetailDrawer.svelte";
  import ConfirmDeleteModal from "$shared/ui/ConfirmDeleteModal.svelte";
  import ImportWizard from "$shared/ui/ImportWizard.svelte";

  const authStore = useAuthStore();
  const ws = useWebSocket();

  const userPermissions = $derived(authStore.user?.permissions || []);
  const canCreate = $derived(userPermissions.includes("supplier.create"));
  const canUpdate = $derived(userPermissions.includes("supplier.update"));
  const canDelete = $derived(userPermissions.includes("supplier.delete"));
  const canExport = $derived(userPermissions.includes("supplier.view"));
  const canImport = $derived(userPermissions.includes("supplier.create"));

  let loading = $state(true);
  let suppliers = $state<Supplier[]>([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state("");
  let statusFilter = $state("all");
  let consignmentFilter = $state(false);
  let referrer = $state<string | null>(null);
  const { sortState, handleSort } = useSortable("name", "asc", load);

  let showFormModal = $state(false);
  let formMode = $state<"add" | "edit">("add");
  let selectedSupplier = $state<Supplier | null>(null);
  let saving = $state(false);

  let showDeleteModal = $state(false);
  let deleteTargetName = $state("");
  let deleteDescription = $state("");
  let deleting = $state(false);

  let staleConflictOpen = $state(false);
  let staleConflictPayload = $state<UpdateSupplierPayload | null>(null);
  let staleConflictVersion = $state<number | undefined>(undefined);

  let showImportWizard = $state(false);
  let showDetailDrawer = $state(false);
  let detailSupplier = $state<Supplier | null>(null);

  async function load() {
    loading = true;
    try {
      const params: SupplierListParams = {
        limit,
        offset,
        search: searchQuery,
        sort_by: sortState.sortBy,
        sort_dir: sortState.sortDir,
      };
      if (statusFilter === "active") params.is_active = true;
      else if (statusFilter === "inactive") params.is_active = false;
      if (consignmentFilter) params.is_consignment = true;

      const result = await getSuppliers(params);
      suppliers = result.data;
      total = result.total;
    } catch {
      toast.error("Failed to load suppliers");
    } finally {
      loading = false;
    }
  }

  const debouncedSearch = debounce(() => {
    offset = 0;
    load();
  }, 300);

  function handleSearch() {
    debouncedSearch();
  }
  function handleStatusChange() {
    offset = 0;
    syncUrl();
    load();
  }
  function handleConsignmentChange() {
    offset = 0;
    syncUrl();
    load();
  }

  function syncUrl() {
    const params = new SvelteURLSearchParams(window.location.search);
    if (consignmentFilter) params.set("is_consignment", "true");
    else params.delete("is_consignment");
    const qs = params.toString();
    const url = qs
      ? `${window.location.pathname}?${qs}`
      : window.location.pathname;
    window.history.replaceState({}, "", url);
  }
  function handlePageChange(newOffset: number, newLimit: number) {
    limit = newLimit;
    offset = newOffset;
    load();
  }

  function openAdd() {
    formMode = "add";
    selectedSupplier = null;
    showFormModal = true;
  }

  function openEdit(supplier: Supplier) {
    formMode = "edit";
    selectedSupplier = supplier;
    showFormModal = true;
  }

  function usageBreakdown(usage?: SupplierUsage): string {
    if (!usage) return "";
    const parts: string[] = [];
    if (usage.product_links)
      parts.push(`${usage.product_links} product link(s)`);
    if (usage.open_purchase_orders)
      parts.push(`${usage.open_purchase_orders} open purchase order(s)`);
    if (usage.active_consignments)
      parts.push(`${usage.active_consignments} active consignment(s)`);
    return parts.join(", ");
  }

  async function handleFormSave(
    data: CreateSupplierPayload | UpdateSupplierPayload,
  ) {
    saving = true;
    try {
      if (formMode === "add") {
        const ok = await createSupplier(data as CreateSupplierPayload);
        if (ok) {
          toast.success("Supplier created");
          showFormModal = false;
          await load();
        } else {
          toast.error("Failed to create supplier");
        }
      } else {
        const payload: UpdateSupplierPayload = {
          ...(data as UpdateSupplierPayload),
          version: selectedSupplier?.version,
        };
        const result = await updateSupplier(selectedSupplier!.id, payload);
        if (result.ok) {
          toast.success("Supplier updated");
          showFormModal = false;
          await load();
        } else if (result.code === "supplier_version_conflict") {
          const fresh = await getSupplier(selectedSupplier!.id);
          staleConflictPayload = data as UpdateSupplierPayload;
          staleConflictVersion = fresh?.version;
          staleConflictOpen = true;
          toast.error(
            "This supplier changed elsewhere. Re-apply your changes on the latest version.",
          );
        } else {
          toast.error("Failed to update supplier");
        }
      }
    } catch {
      toast.error("Failed to save supplier");
    } finally {
      saving = false;
    }
  }

  async function reapplyStaleChanges() {
    if (!selectedSupplier || !staleConflictPayload) return;
    saving = true;
    try {
      const result = await updateSupplier(selectedSupplier.id, {
        ...staleConflictPayload,
        version: staleConflictVersion,
      });
      if (result.ok) {
        toast.success("Supplier updated");
        staleConflictOpen = false;
        staleConflictPayload = null;
        showFormModal = false;
        await load();
      } else if (result.code === "supplier_version_conflict") {
        const fresh = await getSupplier(selectedSupplier.id);
        staleConflictVersion = fresh?.version;
        toast.error("Changed again — review the latest version and retry.");
      } else {
        staleConflictOpen = false;
        toast.error("Failed to update supplier");
      }
    } catch {
      toast.error("Failed to update supplier");
    } finally {
      saving = false;
    }
  }

  function openDelete(supplier: Supplier) {
    selectedSupplier = supplier;
    deleteTargetName = supplier.name;
    deleteDescription = "";
    showDeleteModal = true;
    void getSupplierUsage(supplier.id).then((usage) => {
      if (selectedSupplier?.id !== supplier.id) return;
      const breakdown = usageBreakdown(usage ?? undefined);
      deleteDescription = breakdown
        ? `Referenced by ${breakdown}. Deactivate or unlink first.`
        : "";
    });
  }

  async function handleDeleteConfirm() {
    if (!selectedSupplier) return;
    deleting = true;
    try {
      const result = await deleteSupplier(selectedSupplier.id);
      if (result.ok) {
        toast.success("Supplier deleted");
        showDeleteModal = false;
        selectedSupplier = null;
        await load();
      } else if (result.code === "supplier_in_use") {
        const breakdown = usageBreakdown(result.usage);
        deleteDescription = breakdown
          ? `Referenced by ${breakdown}. Deactivate or unlink first.`
          : "This supplier is still referenced by other records.";
      } else {
        toast.error("Failed to delete supplier");
      }
    } catch {
      toast.error("Failed to delete supplier");
    } finally {
      deleting = false;
    }
  }

  async function handleBulkActivate(ids: number[]) {
    try {
      const result = await bulkUpdateSuppliers(ids, true);
      if (result.ok) {
        toast.success(`${result.updated ?? ids.length} suppliers activated`);
        await load();
      } else {
        toast.error("Failed to activate suppliers");
      }
    } catch {
      toast.error("Failed to activate suppliers");
    }
  }

  async function handleBulkDeactivate(ids: number[]) {
    try {
      const result = await bulkUpdateSuppliers(ids, false);
      if (result.ok) {
        toast.success(`${result.updated ?? ids.length} suppliers deactivated`);
        await load();
      } else if (result.code === "supplier_in_use") {
        const breakdown = usageBreakdown(result.usage);
        toast.error(
          breakdown
            ? `Cannot deactivate: blocked by ${breakdown}.`
            : "One or more suppliers are still in active use.",
        );
      } else {
        toast.error("Failed to deactivate suppliers");
      }
    } catch {
      toast.error("Failed to deactivate suppliers");
    }
  }

  async function handleBulkDelete(ids: number[]) {
    try {
      const result = await bulkDeleteSuppliers(ids);
      if (result.ok) {
        toast.success(`${result.deleted ?? ids.length} suppliers deleted`);
        await load();
      } else if (result.code === "supplier_in_use") {
        const breakdown = usageBreakdown(result.usage);
        const blocked = result.blocked_ids?.length
          ? ` Blocked: ${result.blocked_ids.join(", ")}.`
          : "";
        toast.error(
          breakdown
            ? `Cannot delete: blocked by ${breakdown}.${blocked}`
            : `One or more suppliers are still referenced.${blocked}`,
        );
      } else {
        toast.error("Failed to delete suppliers");
      }
    } catch {
      toast.error("Failed to delete suppliers");
    }
  }

  function handleImport() {
    showImportWizard = true;
  }

  function openDetail(supplier: Supplier) {
    detailSupplier = supplier;
    showDetailDrawer = true;
  }

  function duplicateSupplier(supplier: Supplier) {
    selectedSupplier = {
      ...supplier,
      name: `${supplier.name} (Copy)`,
      id: 0,
    } as Supplier;
    formMode = "add";
    showFormModal = true;
  }

  function viewSupplierProducts(supplier: Supplier) {
    const params = new URLSearchParams({
      supplier_id: supplier.id.toString(),
      supplier_name: supplier.name,
    });
    goto(`/inventory/products?${params.toString()}`);
  }

  onMount(() => {
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get("is_consignment") === "true") {
      consignmentFilter = true;
    }
    referrer = urlParams.get("referrer");
    load();
    return ws.on("supplier_changed", () => load());
  });
</script>

<div class="space-y-5">
  {#if referrer === "consignment"}
    <button
      class="inline-flex items-center gap-1.5 text-sm text-text-muted hover:text-text-secondary transition-colors"
      onclick={() => goto("/consignment")}
    >
      <ArrowLeft size={16} />
      {labels.back}
    </button>
  {/if}
  <SuppliersToolbar
    bind:searchQuery
    bind:statusFilter
    bind:consignmentFilter
    {canCreate}
    {canExport}
    {canImport}
    onsearch={handleSearch}
    onstatuschange={handleStatusChange}
    onconsignmentchange={handleConsignmentChange}
    oncreate={openAdd}
    onimport={handleImport}
  />

  <div class="card overflow-x-auto">
    <SuppliersTable
      {suppliers}
      {loading}
      {searchQuery}
      canEdit={canUpdate}
      {canDelete}
      {canCreate}
      sortBy={sortState.sortBy}
      sortDir={sortState.sortDir}
      onsort={handleSort}
      onedit={openEdit}
      ondelete={openDelete}
      onduplicate={duplicateSupplier}
      onviewproducts={viewSupplierProducts}
      onrowclick={openDetail}
      onbulkactivate={handleBulkActivate}
      onbulkdeactivate={handleBulkDeactivate}
      onbulkdelete={handleBulkDelete}
    />

    {#if !loading && suppliers.length > 0}
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
        <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
      </div>
    {/if}
  </div>
</div>

<SupplierFormModal
  bind:open={showFormModal}
  mode={formMode}
  supplier={selectedSupplier}
  {saving}
  onsave={handleFormSave}
  oncancel={() => {
    showFormModal = false;
    selectedSupplier = null;
  }}
/>

<ConfirmDeleteModal
  bind:open={showDeleteModal}
  onconfirm={handleDeleteConfirm}
  loading={deleting}
  itemName={deleteTargetName}
  description={deleteDescription}
/>

<Modal bind:open={staleConflictOpen} title="Supplier changed" size="sm">
  <p class="text-sm text-text-secondary">
    This supplier was changed by someone else since you opened it. Re-apply your
    changes on the latest version?
  </p>
  {#snippet footer()}
    <Button
      variant="secondary"
      disabled={saving}
      onclick={() => (staleConflictOpen = false)}>{labels.cancel}</Button
    >
    <Button variant="primary" disabled={saving} onclick={reapplyStaleChanges}
      >{labels.update}</Button
    >
  {/snippet}
</Modal>

<ImportWizard
  bind:open={showImportWizard}
  module="suppliers"
  displayName="Suppliers"
  onComplete={() => load()}
/>

<SupplierDetailDrawer
  bind:open={showDetailDrawer}
  supplier={detailSupplier}
  canEdit={canUpdate}
  {canDelete}
  onclose={() => {
    showDetailDrawer = false;
    detailSupplier = null;
  }}
  onedit={(s) => {
    showDetailDrawer = false;
    openEdit(s);
  }}
  ondelete={(s) => {
    showDetailDrawer = false;
    openDelete(s);
  }}
  onviewproducts={(s) => {
    showDetailDrawer = false;
    viewSupplierProducts(s);
  }}
/>
