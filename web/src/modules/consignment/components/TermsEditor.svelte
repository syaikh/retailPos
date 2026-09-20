<script lang="ts">
  import { onMount, tick } from "svelte";
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import {
    Button,
    EmptyState,
    Pagination,
    FormattedNumberInput,
  } from "$shared/ui";
  import { Plus, Trash2, Loader2, Check, Copy, X } from "lucide-svelte";
  import { SvelteSet } from "svelte/reactivity";
  import { labels } from "$shared/i18n";
  import {
    addTerm,
    removeTerm,
    searchAvailableProducts,
  } from "../services/consignment-service";
  import {
    createProduct,
    getNextSku,
  } from "$modules/product/services/product-service";
  import type { Arrangement, Term } from "../types";
  import { SHARE_TYPE_PERCENTAGE, SHARE_TYPE_LABELS } from "../types";
  import { formatCurrency } from "../lib/format";

  const {
    arrangement,
    canUpdate,
    onsaved,
  }: {
    arrangement: Arrangement;
    canUpdate: boolean;
    onsaved?: () => void;
  } = $props();

  let terms = $state<Term[]>([]);
  let loading = $state(true);

  // Draft row state
  let editingTerm = $state<{
    product_id: number;
    product_label: string;
    price: number;
    share_type: string;
    share_value: number;
  } | null>(null);
  let savingTerm = $state(false);

  // Search state
  let searchQuery = $state("");
  let searchResults = $state<{ id: number; sku: string; name: string }[]>([]);
  let searchLoading = $state(false);
  let exactMatch = $state(false);
  let showResults = $state(false);
  let debounceTimer = $state<ReturnType<typeof setTimeout>>();
  let searchInput = $state<HTMLInputElement>();
  let dropdownStyle = $state("");

  let pageLimit = $state(20);
  let pageOffset = $state(0);
  const pagedTerms = $derived(terms.slice(pageOffset, pageOffset + pageLimit));

  async function load() {
    loading = true;
    try {
      terms = arrangement.terms || [];
    } finally {
      loading = false;
    }
  }

  function positionDropdown() {
    if (!searchInput) return;
    const rect = searchInput.getBoundingClientRect();
    const spaceBelow = window.innerHeight - rect.bottom - 8;
    const maxHeight = Math.min(spaceBelow, 300);
    dropdownStyle = `position:fixed;left:${rect.left}px;top:${rect.bottom + 4}px;width:${rect.width}px;max-height:${maxHeight}px;z-index:10000;`;
  }

  function clearSearchState() {
    if (debounceTimer) clearTimeout(debounceTimer);
    searchQuery = "";
    searchResults = [];
    exactMatch = false;
    showResults = false;
  }

  function localExactMatch(query: string): boolean {
    const q = query.trim().toLowerCase();
    if (!q) return false;
    return terms.some((t) => t.product_name?.toLowerCase() === q);
  }

  async function startAddTerm() {
    editingTerm = {
      product_id: 0,
      product_label: "",
      price: 0,
      share_type: SHARE_TYPE_PERCENTAGE,
      share_value: 20,
    };
    clearSearchState();
    await tick();
    searchInput?.focus();
  }

  function handleSearchInput(value: string) {
    searchQuery = value;
    if (debounceTimer) clearTimeout(debounceTimer);
    if (value.length < 2) {
      searchResults = [];
      exactMatch = false;
      showResults = false;
      return;
    }
    const localMatch = localExactMatch(value);
    debounceTimer = setTimeout(async () => {
      searchLoading = true;
      positionDropdown();
      showResults = true;
      try {
        const result = await searchAvailableProducts(arrangement.id, value);
        searchResults = result.products;
        exactMatch = result.exactMatch || localMatch;
      } catch {
        searchResults = [];
        exactMatch = localMatch;
      } finally {
        searchLoading = false;
      }
    }, 300);
  }

  function collapseResults() {
    showResults = false;
  }

  function handleCreateNew() {
    if (!editingTerm) return;
    editingTerm.product_label = searchQuery.trim();
    showResults = false;
  }

  function cancelEdit() {
    editingTerm = null;
    clearSearchState();
  }

  async function clearProduct() {
    if (!editingTerm) return;
    editingTerm.product_id = 0;
    editingTerm.product_label = "";
    clearSearchState();
    await tick();
    searchInput?.focus();
  }

  async function confirmAddTerm() {
    if (!editingTerm || editingTerm.price <= 0) return;
    if (editingTerm.product_id <= 0 && !editingTerm.product_label.trim())
      return;
    savingTerm = true;
    try {
      let productId = editingTerm.product_id;
      if (productId <= 0) {
        const sku = await getNextSku();
        const product = await createProduct({
          sku,
          name: editingTerm.product_label.trim(),
          barcode: null,
          category: "",
          brand_id: null,
          price: editingTerm.price,
          cost: 0,
          stock: 0,
          unit_of_measure_id: null,
          tax_class_id: null,
          weight_grams: null,
          description: "",
          status: "active",
        });
        productId = product.id;
        toast.success(labels.consignmentProductCreated);
      }
      const term = await addTerm(arrangement.id, {
        product_id: productId,
        price: editingTerm.price,
        store_share_type: editingTerm.share_type,
        store_share_value: editingTerm.share_value,
      });
      terms = [...terms, term];
      toast.success(labels.consignmentTermAdded);
      editingTerm = null;
      clearSearchState();
      onsaved?.();
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.consignmentTermsSaveError));
    } finally {
      savingTerm = false;
    }
  }

  async function handleRemoveTerm(t: Term) {
    const labelParts = (t.product_name || "").split(" (");
    const displayName =
      labelParts[0] || t.product_name || `Product #${t.product_id}`;
    if (
      !confirm(
        labels.consignmentRemoveTermConfirm.replace("{product}", displayName),
      )
    )
      return;

    try {
      await removeTerm(arrangement.id, t.product_id);
      terms = terms.filter((term) => term.product_id !== t.product_id);
      toast.success(labels.consignmentTermRemoved);
      onsaved?.();
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.consignmentTermsSaveError));
    }
  }

  function shareLabel(t: {
    store_share_type: string;
    store_share_value: number;
  }): string {
    if (t.store_share_type === SHARE_TYPE_PERCENTAGE)
      return `${t.store_share_value}%`;
    return formatCurrency(t.store_share_value);
  }

  let showCopied = new SvelteSet<string>();

  function copySku(sku: string) {
    navigator.clipboard.writeText(sku).then(() => {
      showCopied.add(sku);
      toast.success(labels.copiedToClipboard);
      setTimeout(() => {
        showCopied.delete(sku);
      }, 2000);
    });
  }

  function handlePageChange(newOffset: number, newLimit: number) {
    pageOffset = newOffset;
    pageLimit = newLimit;
  }

  onMount(() => {
    load();
  });
</script>

<div class="card">
  <div
    class="flex items-center justify-between px-4 py-3 border-b border-border/50"
  >
    <h2 class="font-semibold text-text-primary">
      {labels.consignmentTermsHeader}
    </h2>
    {#if canUpdate}
      <Button variant="secondary" size="sm" onclick={startAddTerm}>
        <Plus class="w-4 h-4" />
        {labels.consignmentAddProduct}
      </Button>
    {/if}
  </div>

  {#if loading}
    <div class="p-8 text-center text-sm text-text-secondary">
      {labels.loading}
    </div>
  {:else if terms.length === 0 && !editingTerm}
    <EmptyState
      icon={Plus}
      title={labels.consignmentNoTerms}
      subtitle={labels.consignmentNoTermsSubtitle}
    />
  {:else}
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="bg-muted/50">
          <tr
            class="text-left text-xs uppercase tracking-wider text-text-secondary"
          >
            <th class="p-4">{labels.consignmentProduct}</th>
            <th class="p-4 text-right">{labels.consignmentPrice}</th>
            <th class="p-4">{labels.consignmentStoreShare}</th>
            {#if canUpdate}
              <th class="p-4 w-20"></th>
            {/if}
          </tr>
        </thead>
        <tbody>
          <!-- Draft row -->
          {#if editingTerm}
            <tr class="border-t border-primary/30 bg-primary/5">
              <td class="p-4">
                {#if editingTerm.product_id > 0}
                  <div class="flex items-center gap-2">
                    <span class="font-medium text-text-primary"
                      >{editingTerm.product_label}</span
                    >
                    <button
                      type="button"
                      onclick={clearProduct}
                      class="p-0.5 text-text-muted hover:text-danger"
                      title="Change product"
                    >
                      <X size={12} />
                    </button>
                  </div>
                {:else}
                  <input
                    type="text"
                    bind:this={searchInput}
                    value={searchQuery}
                    oninput={(e) =>
                      handleSearchInput((e.target as HTMLInputElement).value)}
                    onfocus={positionDropdown}
                    placeholder={labels.consignmentSearchProduct}
                    class="w-full bg-bg-secondary border border-border-default rounded-lg px-3 py-1.5 text-sm outline-none focus:ring-2 focus:ring-primary-default"
                  />
                {/if}
              </td>
              <td class="p-4 text-right">
                <div onfocusin={collapseResults}>
                  <FormattedNumberInput
                    bind:value={editingTerm.price}
                    placeholder="0"
                    disabled={!editingTerm.product_label}
                    class="w-28 text-right"
                  />
                </div>
              </td>
              <td class="p-4">
                <div class="flex items-center gap-2">
                  <select
                    bind:value={editingTerm.share_type}
                    disabled={!editingTerm.product_label}
                    onfocus={collapseResults}
                    class="bg-bg-secondary border border-border-default rounded-lg px-3 py-1.5 text-sm outline-none disabled:opacity-40"
                  >
                    <option value={SHARE_TYPE_PERCENTAGE}
                      >{labels.shareTypePercentage}</option
                    >
                    <option value="fixed_amount"
                      >{labels.shareTypeFixedAmount}</option
                    >
                  </select>
                  <input
                    type="number"
                    bind:value={editingTerm.share_value}
                    min="1"
                    max={editingTerm.share_type === SHARE_TYPE_PERCENTAGE
                      ? 99
                      : undefined}
                    disabled={!editingTerm.product_label}
                    onfocus={collapseResults}
                    class="w-20 bg-bg-secondary border border-border-default rounded-lg px-3 py-1.5 text-sm text-right outline-none focus:ring-2 focus:ring-primary-default disabled:opacity-40"
                  />
                </div>
              </td>
              <td class="p-4">
                <div class="flex items-center gap-1">
                  <button
                    type="button"
                    onclick={confirmAddTerm}
                    disabled={(editingTerm.product_id <= 0 &&
                      !editingTerm.product_label.trim()) ||
                      editingTerm.price <= 0 ||
                      savingTerm}
                    class="p-1 text-success hover:text-success/80 disabled:opacity-40 disabled:cursor-not-allowed"
                  >
                    {#if savingTerm}
                      <Loader2 size={14} class="animate-spin" />
                    {:else}
                      <Check size={14} />
                    {/if}
                  </button>
                  <button
                    type="button"
                    onclick={cancelEdit}
                    disabled={savingTerm}
                    class="p-1 text-text-muted hover:text-danger disabled:opacity-40"
                  >
                    <X size={14} />
                  </button>
                </div>
              </td>
            </tr>
          {/if}

          <!-- Existing terms -->
          {#each pagedTerms as t (t.id || t)}
            <tr
              class="border-t border-border hover:bg-surface-hover/50 transition-colors"
            >
              <td class="p-4">
                <div class="font-medium text-text-primary">
                  {t.product_name}
                </div>
                <div
                  class="text-xs text-text-secondary flex items-center gap-1"
                >
                  {t.product_sku}
                  {#if t.product_sku}
                    <button
                      type="button"
                      onclick={() => copySku(t.product_sku!)}
                      class="p-0.5 text-text-muted hover:text-text-primary"
                      title={labels.copiedToClipboard}
                    >
                      {#if showCopied.has(t.product_sku)}
                        <Check size={10} class="text-success" />
                      {:else}
                        <Copy size={10} />
                      {/if}
                    </button>
                  {/if}
                </div>
              </td>
              <td class="p-4 text-right text-text-primary"
                >{formatCurrency(t.price)}</td
              >
              <td class="p-4 text-text-secondary">
                {labels[SHARE_TYPE_LABELS[t.store_share_type]]} — {shareLabel(
                  t,
                )}
              </td>
              {#if canUpdate}
                <td class="p-4">
                  <button
                    type="button"
                    onclick={() => handleRemoveTerm(t)}
                    class="p-1 text-text-muted hover:text-danger"
                  >
                    <Trash2 size={14} />
                  </button>
                </td>
              {/if}
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <!-- Search results dropdown — rendered outside overflow-x-auto -->
    {#if editingTerm && editingTerm.product_id <= 0 && showResults && searchQuery.length >= 2}
      <div
        class="bg-surface-default border border-border rounded-xl shadow-xl py-1 flex flex-col overflow-y-auto"
        style={dropdownStyle}
        onfocusin={() => {}}
        onfocusout={collapseResults}
      >
        {#if searchLoading}
          <div
            class="px-3 py-4 text-sm text-text-muted text-center flex items-center justify-center gap-2"
          >
            <Loader2 size={14} class="animate-spin" />
            {labels.loading}
          </div>
        {:else}
          {#if searchResults.length > 0}
            <div class="px-3 py-1.5 text-xs text-text-muted">
              {labels.consignmentSimilarProductsFound}
            </div>
            {#each searchResults as product (product.id)}
              <div class="px-3 py-2 text-sm text-text-secondary">
                {product.name}
                {#if product.sku}
                  <span class="text-text-muted">({product.sku})</span>
                {/if}
              </div>
            {/each}
          {/if}
          {#if exactMatch}
            <div class="px-3 py-2 text-sm text-danger border-t border-border">
              {labels.consignmentProductAlreadyExists}
            </div>
          {:else}
            <button
              type="button"
              onclick={handleCreateNew}
              class="w-full text-left px-3 py-2 text-sm text-primary hover:bg-surface-hover border-t border-border"
            >
              {labels.consignmentCreateAsNewProduct.replace(
                "{name}",
                searchQuery,
              )}
            </button>
          {/if}
        {/if}
      </div>
    {/if}
    <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
      <Pagination
        total={terms.length}
        limit={pageLimit}
        offset={pageOffset}
        onPageChange={handlePageChange}
      />
    </div>
  {/if}
</div>
