<script lang="ts">
  import { onMount } from "svelte";
  import { useAuthStore } from "$modules/auth";
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import { goto } from "$app/router";
  import {
    Button,
    Modal,
    SelectSearch,
    EmptyState,
    Badge,
    SearchBar,
    Pagination,
  } from "$shared/ui";
  import {
    Plus,
    ClipboardList,
    Truck,
    RotateCcw,
    Wallet,
    ArrowLeft,
    AlertTriangle,
    ExternalLink,
    Square,
  } from "lucide-svelte";
  import { useRBAC } from "$shared/composables/useRBAC.svelte";
  import { Roles } from "$shared/constants/roles";
  import { debounce } from "$shared/utils/debounce";
  import { labels, t } from "$shared/i18n";
  import {
    listArrangements,
    getArrangement,
    createArrangement,
    endArrangement,
    listConsignmentSuppliers,
    listStock,
    bulkReturnAllStock,
  } from "../services/consignment-service";
  import { getActiveStores } from "$modules/stores/services/stores-service";
  import type { Arrangement, ConsignmentSupplierRef, StockRow } from "../types";
  import type { ArrangementListParams } from "../services/consignment-service";
  import {
    ARRANGEMENT_STATUS_LABELS,
    ARRANGEMENT_STATUS_ACTIVE,
    ARRANGEMENT_STATUS_ENDED,
  } from "../types";
  import { formatDate } from "../lib/format";
  import TermsEditor from "./TermsEditor.svelte";
  import ReceiptEntry from "./ReceiptEntry.svelte";
  import PendingReturnPage from "./PendingReturnPage.svelte";
  import ReturnPage from "./ReturnPage.svelte";
  import SettlementPage from "./SettlementPage.svelte";
  import StockPage from "./StockPage.svelte";

  const authStore = useAuthStore();
  const rbac = useRBAC();
  const isSuperadmin = $derived(rbac.userRole === Roles.superadmin);
  const userPermissions = $derived(authStore.user?.permissions || []);
  const canCreate = $derived(userPermissions.includes("consignment.create"));
  const canUpdate = $derived(userPermissions.includes("consignment.update"));
  const canSettle = $derived(userPermissions.includes("consignment.settle"));
  const canPay = $derived(userPermissions.includes("consignment.pay"));

  let loading = $state(true);
  let arrangements = $state<Arrangement[]>([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state("");
  let statusFilter = $state("all");
  let suppliers = $state<ConsignmentSupplierRef[]>([]);
  let supplierOptions = $state<{ value: number; label: string }[]>([]);

  let showCreateModal = $state(false);
  let creating = $state(false);
  let createSupplierId = $state<number | undefined>(undefined);
  let createStoreId = $state<number | undefined>(undefined);
  let storeOptions = $state<{ value: number; label: string }[]>([]);

  let activeArrangement = $state<Arrangement | null>(null);
  let activeTab = $state<
    "terms" | "receipt" | "pending" | "return" | "settlement" | "stock"
  >("receipt");

  let showEndModal = $state(false);
  let ending = $state(false);
  let showReturnBanner = $state(false);

  let showBulkReturnModal = $state(false);
  let bulkReturning = $state(false);
  let arrangementStock = $state<StockRow[]>([]);

  const filteredStock = $derived(
    arrangementStock.filter(
      (s) =>
        s.arrangement_id === activeArrangement?.id &&
        (s.available_qty > 0 || s.pending_return_qty > 0),
    ),
  );
  const totalReturnQty = $derived(
    filteredStock.reduce((s, r) => s + r.available_qty, 0),
  );

  async function load() {
    loading = true;
    try {
      const params: ArrangementListParams = { limit, offset };
      if (searchQuery) params.search = searchQuery;
      if (statusFilter !== "all") params.status = statusFilter;
      const [arrs, sups] = await Promise.all([
        listArrangements(params),
        listConsignmentSuppliers(),
      ]);
      arrangements = arrs.data;
      total = arrs.total;
      suppliers = sups;
      supplierOptions = sups.map((s) => ({ value: s.id, label: s.name }));

      // Load stores only for superadmin (non-superadmin is auto-assigned)
      if (isSuperadmin) {
        try {
          const stores = await getActiveStores();
          storeOptions = stores.map((s) => ({ value: s.id, label: s.name }));
        } catch {
          storeOptions = [];
        }
      }
    } catch {
      toast.error(labels.consignmentLoadError);
    } finally {
      loading = false;
    }
  }

  const debouncedSearch = debounce(() => {
    offset = 0;
    load();
  }, 400);

  function handleSearch() {
    offset = 0;
    debouncedSearch();
  }

  function handleStatusChange() {
    offset = 0;
    load();
  }

  function handlePageChange(newOffset: number, newLimit: number) {
    offset = newOffset;
    limit = newLimit;
    load();
  }

  function openCreate() {
    createSupplierId = undefined;
    createStoreId = isSuperadmin ? undefined : authStore.user?.store_id;
    showCreateModal = true;
  }

  async function submitCreate() {
    if (!createSupplierId) {
      toast.error(labels.consignmentSelectSupplier);
      return;
    }
    if (isSuperadmin && !createStoreId) {
      toast.error(labels.consignmentSelectStore);
      return;
    }
    creating = true;
    try {
      const created = await createArrangement({
        supplier_id: createSupplierId,
        store_id: createStoreId,
      });
      toast.success(
        t("consignmentCreatedFor", { name: created.supplier_name || "" }),
      );
      showCreateModal = false;
      await load();
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.consignmentCreateError));
    } finally {
      creating = false;
    }
  }

  async function openArrangement(a: Arrangement) {
    activeArrangement = a;
    try {
      const full = await getArrangement(a.id);
      activeArrangement = full;
      activeTab = (full.terms?.length ?? 0) > 0 ? "receipt" : "terms";
    } catch {
      activeTab = "terms";
    }
  }

  function backToList() {
    activeArrangement = null;
    load();
  }

  async function refreshArrangement() {
    if (!activeArrangement) return;
    const arrangementId = activeArrangement.id;
    const params: ArrangementListParams = { limit: 100, offset: 0 };
    if (searchQuery) params.search = searchQuery;
    if (statusFilter !== "all") params.status = statusFilter;
    const arrs = await listArrangements(params);
    arrangements = arrs.data;
    try {
      const updated = await getArrangement(arrangementId);
      activeArrangement = updated;
    } catch {
      const updated = arrs.data.find((a) => a.id === arrangementId);
      if (updated) activeArrangement = updated;
    }
  }

  function openEndModal() {
    showEndModal = true;
  }

  async function confirmEndArrangement() {
    if (!activeArrangement) return;
    ending = true;
    try {
      await endArrangement(activeArrangement.id);
      toast.success(labels.consignmentArrangementEnded);
      showEndModal = false;
      showReturnBanner = false;
      await refreshArrangement();
    } catch (e: unknown) {
      const msg = getApiErrorMessage(e, "");
      if (msg.includes("stock must be returned")) {
        showEndModal = false;
        showReturnBanner = true;
        activeTab = "return";
        try {
          arrangementStock = await listStock(activeArrangement.supplier_id);
        } catch {
          arrangementStock = [];
        }
      } else {
        toast.error(getApiErrorMessage(e, labels.consignmentEndError));
      }
    } finally {
      ending = false;
    }
  }

  async function confirmBulkReturn() {
    if (!activeArrangement) return;
    bulkReturning = true;
    try {
      await bulkReturnAllStock(activeArrangement.id, activeArrangement.supplier_id);
      toast.success(labels.consignmentBulkReturnSuccess);
      showBulkReturnModal = false;
      showReturnBanner = false;
      await refreshArrangement();
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.consignmentBulkReturnError));
    } finally {
      bulkReturning = false;
    }
  }

  onMount(load);
</script>

<div class="space-y-5">
  {#if activeArrangement}
    <div class="flex items-center justify-between">
      <div>
        <button
          class="flex items-center gap-1.5 text-sm text-text-secondary hover:text-text-primary"
          onclick={backToList}
        >
          <ArrowLeft class="w-4 h-4" />
          {labels.back}
        </button>
        <h1 class="mt-2 text-2xl font-bold text-text-primary">
          {activeArrangement.supplier_name}
        </h1>
        <div class="mt-1 flex items-center gap-2">
          <Badge
            variant={activeArrangement.status === ARRANGEMENT_STATUS_ACTIVE
              ? "success"
              : "muted"}
          >
            {labels[ARRANGEMENT_STATUS_LABELS[activeArrangement.status]] ||
              activeArrangement.status}
          </Badge>
          <span class="text-sm text-text-secondary">
            {labels.consignmentLastVisitLabel}
            {formatDate(activeArrangement.last_visit_at)}
          </span>
        </div>
      </div>
      {#if activeArrangement.status === ARRANGEMENT_STATUS_ACTIVE && canUpdate}
        <Button
          variant="danger"
          size="sm"
          onclick={openEndModal}
          class="shrink-0"
        >
          <Square class="w-4 h-4" />
          {labels.consignmentEndArrangement}
        </Button>
      {/if}
    </div>

    {#if (activeArrangement.terms?.length ?? 0) === 0}
      <div
        class="rounded-xl border border-warning/40 bg-warning-subtle/20 p-4 flex items-start gap-3"
      >
        <AlertTriangle size={18} class="text-warning shrink-0 mt-0.5" />
        <div>
          <p class="text-sm font-semibold text-text-primary">
            {labels.consignmentTermsRequiredBanner}
          </p>
          <p class="text-xs text-text-muted mt-1">
            {labels.consignmentTermsRequiredHint}
          </p>
        </div>
      </div>
    {/if}

    {#if showReturnBanner}
      <div
        class="rounded-xl border border-amber-300 bg-amber-50 p-4 flex items-center justify-between"
      >
        <div class="flex items-center gap-2 text-amber-800 text-sm">
          <AlertTriangle class="w-4 h-4 shrink-0" />
          <span>{t("consignmentReturnStockToEnd")}</span>
        </div>
        <div class="flex items-center gap-2">
          <Button
            variant="danger"
            size="sm"
            onclick={() => (showBulkReturnModal = true)}
          >
            {labels.consignmentReturnAllStock}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onclick={() => (showReturnBanner = false)}
          >
            {labels.dismiss}
          </Button>
        </div>
      </div>
    {/if}

    <div class="flex flex-wrap gap-2 border-b border-border/50 pb-3">
      <Button
        variant={activeTab === "receipt" ? "primary" : "ghost"}
        size="sm"
        onclick={() => (activeTab = "receipt")}
      >
        <Truck class="w-4 h-4" />
        {labels.consignmentTabReceipts}
      </Button>
      <Button
        variant={activeTab === "terms" ? "primary" : "ghost"}
        size="sm"
        onclick={() => (activeTab = "terms")}
        disabled={!canUpdate}
      >
        <ClipboardList class="w-4 h-4" />
        {labels.consignmentTabTerms}
      </Button>
      <Button
        variant={activeTab === "pending" ? "primary" : "ghost"}
        size="sm"
        onclick={() => (activeTab = "pending")}
      >
        <RotateCcw class="w-4 h-4" />
        {labels.consignmentTabPendingReturns}
      </Button>
      <Button
        variant={activeTab === "return" ? "primary" : "ghost"}
        size="sm"
        onclick={() => (activeTab = "return")}
      >
        <RotateCcw class="w-4 h-4" />
        {labels.consignmentTabReturns}
      </Button>
      <Button
        variant={activeTab === "settlement" ? "primary" : "ghost"}
        size="sm"
        onclick={() => (activeTab = "settlement")}
      >
        <Wallet class="w-4 h-4" />
        {labels.consignmentTabSettlement}
      </Button>
      <Button
        variant={activeTab === "stock" ? "primary" : "ghost"}
        size="sm"
        onclick={() => (activeTab = "stock")}
      >
        <Truck class="w-4 h-4" />
        {labels.consignmentTabStock}
      </Button>
    </div>

    {#if activeTab === "receipt"}
      <ReceiptEntry
        arrangement={activeArrangement}
        {canCreate}
        oncreated={refreshArrangement}
      />
    {:else if activeTab === "terms"}
      <TermsEditor
        arrangement={activeArrangement}
        {canUpdate}
        onsaved={refreshArrangement}
      />
    {:else if activeTab === "pending"}
      <PendingReturnPage
        arrangement={activeArrangement}
        {canCreate}
        oncreated={refreshArrangement}
      />
    {:else if activeTab === "return"}
      <ReturnPage
        arrangement={activeArrangement}
        {canCreate}
        oncreated={refreshArrangement}
      />
    {:else if activeTab === "settlement"}
      <SettlementPage
        arrangement={activeArrangement}
        {canSettle}
        {canPay}
        onsettled={refreshArrangement}
      />
    {:else}
      <StockPage arrangement={activeArrangement} />
    {/if}
  {:else}
    <div class="card p-4">
      <div class="flex items-center gap-3">
        <div class="flex-1">
          <SearchBar
            bind:value={searchQuery}
            placeholder={labels.consignmentSearchSupplierPlaceholder}
            oninput={handleSearch}
            inputClass="h-10"
          />
        </div>
        <div
          class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border-default"
          role="group"
          aria-label={labels.consignmentTabTerms}
        >
          <button
            class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter ===
            'all'
              ? 'bg-primary-subtle text-primary-light border border-primary-default/20'
              : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
            onclick={() => {
              statusFilter = "all";
              handleStatusChange();
            }}
            aria-pressed={statusFilter === "all"}>{labels.all}</button
          >
          <button
            class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter ===
            ARRANGEMENT_STATUS_ACTIVE
              ? 'bg-success-subtle text-success-light'
              : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
            onclick={() => {
              statusFilter = ARRANGEMENT_STATUS_ACTIVE;
              handleStatusChange();
            }}
            aria-pressed={statusFilter === ARRANGEMENT_STATUS_ACTIVE}
            >{labels[
              ARRANGEMENT_STATUS_LABELS[ARRANGEMENT_STATUS_ACTIVE]
            ]}</button
          >
          <button
            class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter ===
            ARRANGEMENT_STATUS_ENDED
              ? 'bg-danger-subtle text-danger-light'
              : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
            onclick={() => {
              statusFilter = ARRANGEMENT_STATUS_ENDED;
              handleStatusChange();
            }}
            aria-pressed={statusFilter === ARRANGEMENT_STATUS_ENDED}
            >{labels[
              ARRANGEMENT_STATUS_LABELS[ARRANGEMENT_STATUS_ENDED]
            ]}</button
          >
        </div>
        <Button
          variant="secondary"
          class="shrink-0 px-5"
          onclick={() =>
            goto("/suppliers?is_consignment=true&referrer=consignment")}
        >
          <ExternalLink size={16} />
          {labels.viewSuppliers}
        </Button>
        {#if canCreate}
          <Button
            variant="primary"
            class="shrink-0 shadow-glow-primary-sm px-5"
            onclick={openCreate}
          >
            <Plus size={18} />
            {labels.consignmentNewArrangement}
          </Button>
        {/if}
      </div>
    </div>

    <div class="card overflow-x-auto">
      {#if loading}
        <div class="p-8 text-center text-sm text-text-secondary">
          {labels.loading}
        </div>
      {:else if arrangements.length === 0}
        <EmptyState
          icon={Truck}
          title={labels.consignmentNoArrangements}
          subtitle={labels.consignmentNoArrangementsSubtitle}
        />
      {:else}
        <table class="w-full text-sm">
          <thead>
            <tr
              class="text-left text-xs uppercase tracking-wider text-text-secondary border-b border-border/50"
            >
              <th class="px-4 py-3">{labels.supplier}</th>
              <th class="px-4 py-3">{labels.status}</th>
              <th class="px-4 py-3">{labels.consignmentTabTerms}</th>
              <th class="px-4 py-3">{labels.consignmentLastVisitLabel}</th>
              <th class="px-4 py-3 text-right">{labels.actions}</th>
            </tr>
          </thead>
          <tbody>
            {#each arrangements as a (a.id || a)}
              <tr class="border-b border-border/40 hover:bg-surface-subtle/50">
                <td class="px-4 py-3 font-medium text-text-primary"
                  >{a.supplier_name}</td
                >
                <td class="px-4 py-3">
                  <Badge
                    variant={a.status === ARRANGEMENT_STATUS_ACTIVE
                      ? "success"
                      : "muted"}
                  >
                    {labels[ARRANGEMENT_STATUS_LABELS[a.status]] || a.status}
                  </Badge>
                </td>
                <td class="px-4 py-3 text-text-secondary"
                  >{t("consignmentProductCount", {
                    count: a.terms?.length ?? 0,
                  })}</td
                >
                <td class="px-4 py-3 text-text-secondary"
                  >{formatDate(a.last_visit_at)}</td
                >
                <td class="px-4 py-3 text-right">
                  <Button
                    variant="secondary"
                    size="sm"
                    onclick={() => openArrangement(a)}
                  >
                    {labels.consignmentOpen}
                  </Button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
        {#if !loading && arrangements.length > 0}
          <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
            <Pagination
              {total}
              {limit}
              {offset}
              onPageChange={handlePageChange}
            />
          </div>
        {/if}
      {/if}
    </div>
  {/if}
</div>

<Modal
  bind:open={showCreateModal}
  title={labels.consignmentNewArrangement}
  size="md"
>
  <div class="space-y-4">
    <label
      class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
    >
      <span
        >{labels.consignmentSupplier} <span class="text-danger">*</span></span
      >
      <SelectSearch
        bind:value={createSupplierId}
        options={supplierOptions}
        placeholder={labels.consignmentSupplierPlaceholder}
        searchPlaceholder={labels.consignmentSearchSupplier}
        notFoundText={labels.consignmentNoConsignmentSuppliers}
      />
    </label>
    {#if isSuperadmin}
      <label
        class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
      >
        <span>{labels.consignmentStore} <span class="text-danger">*</span></span
        >
        <SelectSearch
          bind:value={createStoreId}
          options={storeOptions}
          placeholder={labels.consignmentStorePlaceholder}
          searchPlaceholder={labels.consignmentSearchStore}
          notFoundText={labels.consignmentNoStores}
        />
      </label>
    {/if}
    {#if suppliers.length === 0}
      <p class="text-xs text-amber-600">
        {labels.consignmentNoConsignmentSuppliersHint}
      </p>
    {/if}
  </div>
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showCreateModal = false)}
        >{labels.cancel}</Button
      >
      <Button onclick={submitCreate} disabled={creating}>
        {creating ? labels.saving : labels.create}
      </Button>
    </div>
  {/snippet}
</Modal>

<Modal
  bind:open={showEndModal}
  title={labels.consignmentEndArrangement}
  size="md"
>
  <div class="space-y-4">
    <p class="text-sm text-text-secondary">
      {labels.consignmentEndArrangementConfirm}
    </p>
    {#if activeArrangement}
      <div class="rounded-lg bg-muted/30 p-3">
        <p class="text-sm font-medium text-text-primary">
          {activeArrangement.supplier_name}
        </p>
        <p class="text-xs text-text-muted mt-1">
          {t("consignmentProductCount", {
            count: activeArrangement.terms?.length ?? 0,
          })}
        </p>
      </div>
    {/if}
  </div>
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showEndModal = false)}
        >{labels.cancel}</Button
      >
      <Button
        variant="danger"
        onclick={confirmEndArrangement}
        disabled={ending}
      >
        {ending ? labels.saving : labels.consignmentEndArrangement}
      </Button>
    </div>
  {/snippet}
</Modal>

<Modal
  bind:open={showBulkReturnModal}
  title={labels.consignmentReturnAllStock}
  size="md"
>
  <div class="space-y-4">
    <p class="text-sm text-text-secondary">
      {labels.consignmentBulkReturnDescription}
    </p>
    {#if filteredStock.length > 0}
      <div class="rounded-lg border border-border-default overflow-hidden">
        <table class="w-full text-sm">
          <thead class="bg-muted/50">
            <tr
              class="text-left text-xs uppercase tracking-wider text-text-secondary"
            >
              <th class="p-3">{labels.consignmentProduct}</th>
              <th class="p-3 text-right">{labels.consignmentQty}</th>
            </tr>
          </thead>
          <tbody>
            {#each filteredStock as row (row.product_id || row)}
              <tr class="border-t border-border/40">
                <td class="p-3 text-text-primary">{row.product_name}</td>
                <td class="p-3 text-right text-text-primary font-medium">
                  {row.available_qty}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <p class="text-xs text-text-muted">
        {t("consignmentBulkReturnTotalQty", { count: totalReturnQty })}
      </p>
    {:else}
      <p class="text-sm text-amber-600">{labels.consignmentNoStockToReturn}</p>
    {/if}
    <div
      class="rounded-lg bg-amber-50 border border-amber-200 p-3 text-xs text-amber-800"
    >
      {labels.consignmentBulkReturnWarning}
    </div>
  </div>
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showBulkReturnModal = false)}
        >{labels.cancel}</Button
      >
      <Button
        variant="danger"
        onclick={confirmBulkReturn}
        disabled={bulkReturning || filteredStock.length === 0}
      >
        {bulkReturning ? labels.saving : labels.consignmentReturnAllStock}
      </Button>
    </div>
  {/snippet}
</Modal>
