<script lang="ts">
  import { onMount } from "svelte";
  import { toast } from "$shared/stores/toast.svelte";
  import { EmptyState, Badge, Pagination, SearchBar } from "$shared/ui";
  import { Package, Copy, Check, Search } from "lucide-svelte";
  import { SvelteSet } from "svelte/reactivity";
  import { labels, t } from "$shared/i18n";
  import { listStock } from "../services/consignment-service";
  import type { Arrangement, StockRow } from "../types";
  import { formatCurrency } from "../lib/format";

  const {
    arrangement,
  }: {
    arrangement: Arrangement;
  } = $props();

  let rows = $state<StockRow[]>([]);
  let loading = $state(true);
  let total = $state(0);
  let loadSeq = 0;
  let filterDebounce: ReturnType<typeof setTimeout> | undefined;

  let pageLimit = $state(20);
  let pageOffset = $state(0);
  let filterQuery = $state("");

  const priceByProduct = $derived.by(() => {
    const map: Record<number, number> = {};
    for (const term of arrangement.terms || []) {
      map[term.product_id] = term.price;
    }
    return map;
  });

  async function load() {
    const seq = ++loadSeq;
    loading = true;
    try {
      const res = await listStock(arrangement.supplier_id, {
        limit: pageLimit,
        offset: pageOffset,
        search: filterQuery.trim() || undefined,
      });
      if (seq !== loadSeq) return;
      rows = res.data;
      total = res.total;
    } catch {
      if (seq !== loadSeq) return;
      rows = [];
      total = 0;
    } finally {
      if (seq === loadSeq) loading = false;
    }
  }

  onMount(() => {
    load();
    return () => {
      if (filterDebounce) clearTimeout(filterDebounce);
    };
  });

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
    void load();
  }

  function handleFilterInput() {
    pageOffset = 0;
    if (filterDebounce) clearTimeout(filterDebounce);
    filterDebounce = setTimeout(() => {
      void load();
    }, 300);
  }
</script>

<div class="card">
  <div class="flex items-center gap-3 px-4 py-3 border-b border-border/50">
    <h2 class="font-semibold text-text-primary whitespace-nowrap">
      {t("consignmentStockFor", { name: arrangement.supplier_name || "" })}
    </h2>
    {#if rows.length > 0 || filterQuery !== ""}
      <SearchBar
        bind:value={filterQuery}
        placeholder={labels.consignmentFilterByProduct}
        oninput={handleFilterInput}
        class="flex-1 max-w-xs"
      />
    {/if}
  </div>

  {#if loading && rows.length === 0}
    <div class="p-8 text-center text-sm text-text-secondary">
      {labels.loading}
    </div>
  {:else if rows.length === 0}
    {#if filterQuery.trim()}
      <EmptyState
        icon={Search}
        title={labels.noResultsFor.replace("{query}", filterQuery.trim())}
      />
    {:else}
      <EmptyState
        icon={Package}
        title={labels.consignmentNoStock}
        subtitle={labels.consignmentNoStockSubtitle}
      />
    {/if}
  {:else}
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="bg-muted/50">
          <tr
            class="text-left text-xs uppercase tracking-wider text-text-secondary"
          >
            <th class="p-4">{labels.consignmentProduct}</th>
            <th class="p-4 text-right">{labels.consignmentAvailableStock}</th>
            <th class="p-4 text-right">{labels.consignmentPendingReturnQty}</th>
            <th class="p-4 text-right">{labels.consignmentPricePerUnit}</th>
          </tr>
        </thead>
        <tbody>
          {#each rows as r, stockIdx (stockIdx)}
            <tr
              class="border-t border-border hover:bg-surface-hover/50 transition-colors"
            >
              <td class="p-4">
                <div class="font-medium text-text-primary">
                  {r.product_name}
                </div>
                <div
                  class="text-xs text-text-secondary flex items-center gap-1"
                >
                  {r.product_sku}
                  {#if r.product_sku}
                    <button
                      type="button"
                      onclick={() => copySku(r.product_sku ?? "")}
                      class="p-0.5 text-text-muted hover:text-text-primary"
                      title={labels.copiedToClipboard}
                    >
                      {#if showCopied.has(r.product_sku)}
                        <Check size={10} class="text-success" />
                      {:else}
                        <Copy size={10} />
                      {/if}
                    </button>
                  {/if}
                </div>
              </td>
              <td class="p-4 text-right">
                <Badge variant={r.available_qty > 0 ? "success" : "muted"}
                  >{r.available_qty}</Badge
                >
              </td>
              <td class="p-4 text-right text-text-secondary"
                >{r.pending_return_qty}</td
              >
              <td class="p-4 text-right text-text-primary">
                {priceByProduct[r.product_id] != null
                  ? formatCurrency(priceByProduct[r.product_id])
                  : "-"}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
      <Pagination
        {total}
        limit={pageLimit}
        offset={pageOffset}
        onPageChange={handlePageChange}
      />
    </div>
  {/if}
</div>
