<script lang="ts">
  import { onMount } from "svelte";
  import { toast } from "$shared/stores/toast.svelte";
  import { getApiErrorMessage } from "$shared/utils/error-utils";
  import {
    Button,
    Modal,
    Input,
    NumberInput,
    EmptyState,
    Pagination,
  } from "$shared/ui";
  import { Plus, Check, Trash2 } from "lucide-svelte";
  import { setDropdownOpen } from "$shared/ui/dropdown-state";
  import { labels } from "$shared/i18n";
  import {
    setTerms,
    listAddTermProductOptions,
  } from "../services/consignment-service";
  import type { Arrangement, Term, SetTermsPayload } from "../types";
  import {
    SHARE_TYPE_PERCENTAGE,
    SHARE_TYPE_FIXED_AMOUNT,
    SHARE_TYPE_LABELS,
  } from "../types";
  import { formatCurrency } from "../lib/format";
  import AddProductInline from "./AddProductInline.svelte";
  import AddBatchProductsInline from "./AddBatchProductsInline.svelte";

  interface EditableTerm {
    product_id: number;
    product_name: string;
    product_sku: string;
    price: number;
    store_share_type: string;
    store_share_value: number;
  }

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
  let productOptions = $state<{ value: number; label: string }[]>([]);
  let showManageModal = $state(false);
  let showCreateSingleModal = $state(false);
  let showCreateBatchModal = $state(false);
  let saving = $state(false);

  // Multi-select state
  let selectedProductIds = $state<number[]>([]);
  let dropdownOpen = $state(false);
  let searchTerm = $state("");
  let triggerEl = $state<HTMLButtonElement>();
  let dropdownStyle = $state("");

  // Editable terms in modal
  let editableTerms = $state<EditableTerm[]>([]);

  // Default share type/value for newly added products
  let defaultShareType = $state(SHARE_TYPE_PERCENTAGE);
  let defaultShareValue = $state(20);

  let pageLimit = $state(20);
  let pageOffset = $state(0);
  const pagedTerms = $derived(terms.slice(pageOffset, pageOffset + pageLimit));

  const filteredOptions = $derived(
    searchTerm
      ? productOptions.filter((o) =>
          o.label.toLowerCase().includes(searchTerm.toLowerCase()),
        )
      : productOptions,
  );

  const editableTermCount = $derived(editableTerms.length);

  async function load() {
    loading = true;
    try {
      terms = arrangement.terms || [];
    } finally {
      loading = false;
    }
  }

  async function loadProducts() {
    try {
      const opts = await listAddTermProductOptions(arrangement.id);
      productOptions = opts.map((p) => ({
        value: p.id,
        label: p.sku ? `${p.name} (${p.sku})` : p.name,
      }));
    } catch {
      productOptions = [];
    }
  }

  function openManage() {
    // Initialize editable terms from existing terms
    editableTerms = terms.map((t) => ({
      product_id: t.product_id,
      product_name: t.product_name || "",
      product_sku: t.product_sku || "",
      price: t.price,
      store_share_type: t.store_share_type,
      store_share_value: t.store_share_value,
    }));
    selectedProductIds = editableTerms.map((t) => t.product_id);
    defaultShareType = SHARE_TYPE_PERCENTAGE;
    defaultShareValue = 20;
    searchTerm = "";
    showManageModal = true;
  }

  function toggleDropdown() {
    if (dropdownOpen) {
      dropdownOpen = false;
      setDropdownOpen(false);
      return;
    }
    if (triggerEl) {
      const rect = triggerEl.getBoundingClientRect();
      const spaceBelow = window.innerHeight - rect.bottom;
      const dropHeight = 380;
      const openUp = spaceBelow < dropHeight;
      dropdownStyle = `position:fixed;left:${rect.left}px;width:${rect.width}px;z-index:100;${openUp ? `bottom:${window.innerHeight - rect.top + 4}px` : `top:${rect.bottom + 4}px`}`;
    }
    searchTerm = "";
    dropdownOpen = true;
    setDropdownOpen(true);
  }

  $effect(() => {
    if (!dropdownOpen) return;
    function handleClick(e: MouseEvent) {
      const target = e.target as HTMLElement;
      if (!target.closest("[data-dropdown-popover]")) {
        dropdownOpen = false;
        setDropdownOpen(false);
      }
    }
    function handleKeydown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        e.stopPropagation();
        dropdownOpen = false;
        setDropdownOpen(false);
      }
    }
    document.addEventListener("click", handleClick, true);
    document.addEventListener("keydown", handleKeydown, true);
    return () => {
      document.removeEventListener("click", handleClick, true);
      document.removeEventListener("keydown", handleKeydown, true);
    };
  });

  function toggleProductSelection(productId: number) {
    const opt = productOptions.find((o) => o.value === productId);
    if (!opt) return;

    if (selectedProductIds.includes(productId)) {
      // Remove from selection and editable terms
      selectedProductIds = selectedProductIds.filter((id) => id !== productId);
      editableTerms = editableTerms.filter((t) => t.product_id !== productId);
    } else {
      // Add to selection — price must be set explicitly in the table
      selectedProductIds = [...selectedProductIds, productId];
      const labelParts = opt.label.match(/^(.+)\s*\((.+)\)$/);
      editableTerms = [
        ...editableTerms,
        {
          product_id: productId,
          product_name: labelParts ? labelParts[1].trim() : opt.label,
          product_sku: labelParts ? labelParts[2] : "",
          price: 0,
          store_share_type: defaultShareType,
          store_share_value: defaultShareValue,
        },
      ];
    }
  }

  function removeTerm(index: number) {
    const removed = editableTerms[index];
    editableTerms = editableTerms.filter((_, i) => i !== index);
    selectedProductIds = selectedProductIds.filter(
      (id) => id !== removed.product_id,
    );
  }

  function updateTerm(
    index: number,
    field: keyof EditableTerm,
    value: string | number,
  ) {
    editableTerms = editableTerms.map((row, i) =>
      i === index ? { ...row, [field]: value } : row,
    );
  }

  function onSingleProductCreated(product: {
    id: number;
    sku: string;
    name: string;
    price: number;
  }) {
    showCreateSingleModal = false;
    selectedProductIds = [...selectedProductIds, product.id];
    editableTerms = [
      ...editableTerms,
      {
        product_id: product.id,
        product_name: product.name,
        product_sku: product.sku,
        price: product.price,
        store_share_type: defaultShareType,
        store_share_value: defaultShareValue,
      },
    ];
    loadProducts();
  }

  function onBatchProductsCreated(
    products: { id: number; sku: string; name: string; price: number }[],
  ) {
    showCreateBatchModal = false;
    for (const p of products) {
      if (!selectedProductIds.includes(p.id)) {
        selectedProductIds = [...selectedProductIds, p.id];
        editableTerms = [
          ...editableTerms,
          {
            product_id: p.id,
            product_name: p.name,
            product_sku: p.sku,
            price: p.price,
            store_share_type: defaultShareType,
            store_share_value: defaultShareValue,
          },
        ];
      }
    }
    loadProducts();
  }

  function validateRow(row: EditableTerm): string | null {
    if (row.price < 1) return labels.consignmentPriceAtLeastOne;
    if (row.store_share_type === SHARE_TYPE_PERCENTAGE) {
      if (row.store_share_value < 1 || row.store_share_value >= 100)
        return labels.consignmentPercentRange;
    } else {
      if (row.store_share_value < 1)
        return labels.consignmentShareGreaterThanZero;
      if (row.store_share_value >= row.price)
        return labels.consignmentShareValueRange.replace(
          "{max}",
          formatCurrency(row.price - 1),
        );
    }
    return null;
  }

  async function submitTerms() {
    if (editableTerms.length === 0) {
      toast.error(labels.consignmentSelectAtLeastOneProduct);
      return;
    }

    // Validate all rows
    for (const row of editableTerms) {
      const err = validateRow(row);
      if (err) {
        toast.error(err);
        return;
      }
    }

    const confirmMsg = labels.consignmentConfirmReplace.replace(
      "{count}",
      String(editableTerms.length),
    );
    if (!confirm(confirmMsg)) return;

    saving = true;
    try {
      const payload: SetTermsPayload[] = editableTerms.map((t) => ({
        product_id: t.product_id,
        price: t.price,
        store_share_type: t.store_share_type,
        store_share_value: t.store_share_value,
      }));

      const saved = await setTerms(arrangement.id, payload);
      terms = saved;
      toast.success(labels.consignmentTermsSaved);
      showManageModal = false;
      onsaved?.();
      await loadProducts();
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.consignmentTermsSaveError));
    } finally {
      saving = false;
    }
  }

  function shareLabel(t: Term): string {
    if (t.store_share_type === SHARE_TYPE_PERCENTAGE)
      return `${t.store_share_value}%`;
    return formatCurrency(t.store_share_value);
  }

  function handlePageChange(newOffset: number, newLimit: number) {
    pageOffset = newOffset;
    pageLimit = newLimit;
  }

  onMount(() => {
    load();
    loadProducts();
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
      <Button variant="secondary" size="sm" onclick={openManage}>
        <Plus class="w-4 h-4" />
        {labels.consignmentManageTerms}
      </Button>
    {/if}
  </div>

  {#if loading}
    <div class="p-8 text-center text-sm text-text-secondary">
      {labels.loading}
    </div>
  {:else if terms.length === 0}
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
          </tr>
        </thead>
        <tbody>
          {#each pagedTerms as t (t.id || t)}
            <tr
              class="border-t border-border hover:bg-surface-hover/50 transition-colors"
            >
              <td class="p-4">
                <div class="font-medium text-text-primary">
                  {t.product_name}
                </div>
                <div class="text-xs text-text-secondary">{t.product_sku}</div>
              </td>
              <td class="p-4 text-right text-text-primary"
                >{formatCurrency(t.price)}</td
              >
              <td class="p-4 text-text-secondary">
                {labels[SHARE_TYPE_LABELS[t.store_share_type]]} — {shareLabel(
                  t,
                )}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
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

<!-- Manage Terms Modal -->
<Modal
  bind:open={showManageModal}
  title={labels.consignmentManageTerms}
  size="xl"
  panelClass="max-h-[92vh]"
>
  <div class="space-y-4">
    <p class="text-sm text-text-secondary">
      {labels.consignmentManageTermsDescription}
    </p>

    <!-- Product Multi-Select -->
    <label
      class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary"
    >
      <span>{labels.consignmentAddProduct}</span>

      <div class="relative">
        <button
          type="button"
          bind:this={triggerEl}
          onclick={toggleDropdown}
          class="w-full rounded-xl border bg-bg-secondary px-3.5 py-2.5 text-sm text-left transition-colors duration-200 flex items-center gap-2 border-border-default hover:border-primary-default focus:outline-none focus:ring-2 focus:ring-primary/30"
        >
          <span class="flex-1 truncate">
            {labels.consignmentSelectProduct}
          </span>
          <Plus
            size={16}
            class="text-text-muted transition-transform duration-200 shrink-0 {dropdownOpen
              ? 'rotate-45'
              : ''}"
          />
        </button>

        {#if dropdownOpen}
          <div
            data-dropdown-popover
            style={dropdownStyle}
            class="bg-surface-default border border-border rounded-xl shadow-xl py-1 overflow-hidden max-h-[360px] flex flex-col"
          >
            <div class="px-2 pb-1.5">
              <input
                type="text"
                bind:value={searchTerm}
                placeholder={labels.consignmentSearchProduct}
                class="w-full bg-bg-secondary border border-border-default rounded-lg px-3 py-1.5 text-sm outline-none"
              />
            </div>

            <div class="overflow-y-auto flex-1 min-h-0">
              {#if filteredOptions.length === 0}
                <div class="px-3 py-4 text-sm text-text-muted text-center">
                  {labels.consignmentProductNotFound}
                </div>
              {:else}
                {#each filteredOptions as opt (opt.value)}
                  <button
                    type="button"
                    onclick={() => toggleProductSelection(opt.value)}
                    class="w-full text-left px-3 py-2 text-sm transition-colors flex items-center gap-2 hover:bg-surface-hover"
                  >
                    <div
                      class="w-4 h-4 rounded border flex items-center justify-center {selectedProductIds.includes(
                        opt.value,
                      )
                        ? 'bg-primary border-primary text-white'
                        : 'border-border-default'}"
                    >
                      {#if selectedProductIds.includes(opt.value)}
                        <Check size={12} />
                      {/if}
                    </div>
                    <span>{opt.label}</span>
                  </button>
                {/each}
              {/if}
            </div>

            <div class="border-t border-border px-2 py-1.5 shrink-0">
              <button
                type="button"
                onclick={() => {
                  dropdownOpen = false;
                  setDropdownOpen(false);
                  showCreateSingleModal = true;
                }}
                class="w-full text-left px-3 py-2 text-sm text-primary hover:bg-surface-hover rounded-lg"
              >
                {labels.consignmentCreateNewProduct}
              </button>
              <button
                type="button"
                onclick={() => {
                  dropdownOpen = false;
                  setDropdownOpen(false);
                  showCreateBatchModal = true;
                }}
                class="w-full text-left px-3 py-2 text-sm text-primary hover:bg-surface-hover rounded-lg"
              >
                {labels.consignmentCreateMultipleProducts}
              </button>
            </div>
          </div>
        {/if}
      </div>
    </label>

    <!-- Terms Table -->
    {#if editableTerms.length > 0}
      <div>
        <div class="text-sm font-medium text-text-secondary mb-2">
          {labels.consignmentTermsCount.replace(
            "{count}",
            String(editableTermCount),
          )}
        </div>
        <div class="border border-border rounded-lg overflow-hidden">
          <table class="w-full text-sm">
            <thead class="bg-muted/50">
              <tr
                class="text-left text-xs uppercase tracking-wider text-text-secondary"
              >
                <th class="p-3">{labels.consignmentProduct}</th>
                <th class="p-3 w-28">{labels.consignmentPrice} (Rp)</th>
                <th class="p-3 w-44">{labels.consignmentShareType}</th>
                <th class="p-3 w-28">{labels.consignmentShareValue}</th>
                <th class="p-3 w-12"></th>
              </tr>
            </thead>
            <tbody>
              {#each editableTerms as row, i (row.product_id)}
                <tr class="border-t border-border">
                  <td class="p-2">
                    <div class="font-medium text-text-primary">
                      {row.product_name}
                    </div>
                    <div class="text-xs text-text-secondary">
                      {row.product_sku}
                    </div>
                  </td>
                  <td class="p-2">
                    <NumberInput
                      min="1"
                      value={row.price}
                      oninput={(e: Event) =>
                        updateTerm(
                          i,
                          "price",
                          Number((e.target as HTMLInputElement).value),
                        )}
                      class="h-8 text-sm"
                    />
                  </td>
                  <td class="p-2">
                    <Input
                      tag="select"
                      value={row.store_share_type}
                      oninput={(e: Event) =>
                        updateTerm(
                          i,
                          "store_share_type",
                          (e.target as HTMLSelectElement).value,
                        )}
                      class="h-8 text-sm"
                    >
                      <option value={SHARE_TYPE_PERCENTAGE}
                        >{labels[
                          SHARE_TYPE_LABELS[SHARE_TYPE_PERCENTAGE]
                        ]}</option
                      >
                      <option value={SHARE_TYPE_FIXED_AMOUNT}
                        >{labels[
                          SHARE_TYPE_LABELS[SHARE_TYPE_FIXED_AMOUNT]
                        ]}</option
                      >
                    </Input>
                  </td>
                  <td class="p-2">
                    <NumberInput
                      min="1"
                      max={row.store_share_type === SHARE_TYPE_PERCENTAGE
                        ? "99"
                        : row.price >= 2
                          ? String(row.price - 1)
                          : "0"}
                      value={row.store_share_value}
                      oninput={(e: Event) =>
                        updateTerm(
                          i,
                          "store_share_value",
                          Number((e.target as HTMLInputElement).value),
                        )}
                      class="h-8 text-sm"
                      disabled={row.store_share_type ===
                        SHARE_TYPE_FIXED_AMOUNT && row.price < 2}
                    />
                  </td>
                  <td class="p-2">
                    <button
                      type="button"
                      onclick={() => removeTerm(i)}
                      class="p-1 text-text-muted hover:text-danger"
                    >
                      <Trash2 size={14} />
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {:else}
      <div class="text-sm text-text-muted text-center py-4">
        {labels.consignmentNoTermsHint}
      </div>
    {/if}

    <p class="text-xs text-text-muted">{labels.consignmentTermsNote}</p>
  </div>
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button
        variant="secondary"
        onclick={() => {
          showManageModal = false;
          dropdownOpen = false;
          setDropdownOpen(false);
        }}
      >
        {labels.cancel}
      </Button>
      <Button
        onclick={submitTerms}
        disabled={saving || editableTermCount === 0}
      >
        {saving
          ? labels.saving
          : labels.consignmentTermsCount.replace(
              "{count}",
              String(editableTermCount),
            )}
      </Button>
    </div>
  {/snippet}
</Modal>

{#if showCreateSingleModal}
  <AddProductInline
    bind:open={showCreateSingleModal}
    oncreated={onSingleProductCreated}
  />
{/if}

{#if showCreateBatchModal}
  <AddBatchProductsInline
    bind:open={showCreateBatchModal}
    oncreated={onBatchProductsCreated}
  />
{/if}
