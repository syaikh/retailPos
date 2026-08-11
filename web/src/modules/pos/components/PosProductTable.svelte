<script lang="ts">
  import { Badge, Button, Pagination, Skeleton } from '$shared/ui';
  import { Plus, Copy, Package } from 'lucide-svelte';
  import { labels } from '$shared/i18n';

  let {
    products = [],
    loading = false,
    total = 0,
    limit = 20,
    offset = 0,
    showCopySuccess = $bindable(null as Set<string> | null),
    warningThreshold = 10,
    criticalThreshold = 5,
    selectedIndex = $bindable(-1),
    element = $bindable(undefined as HTMLElement | undefined),
    onaddtocart = (product: any) => {},
    oncopy = (value: string, field: string) => {},
    onpagechange = (newOffset: number) => {},
  }: {
    products: any[];
    loading: boolean;
    total: number;
    limit: number;
    offset: number;
    showCopySuccess: Set<string> | null;
    warningThreshold: number;
    criticalThreshold: number;
    selectedIndex?: number;
    element?: HTMLElement;
    onaddtocart?: (product: any) => void;
    oncopy?: (value: string, field: string) => void;
    onpagechange?: (newOffset: number) => void;
  } = $props();
</script>

{#if loading}
  <div class="flex-1 overflow-y-auto">
    {#each { length: 8 } as _}
      <div class="flex items-center gap-4 px-4 py-3 border-b border-border">
        <Skeleton width="w-40" height="h-4" />
        <Skeleton width="w-20" height="h-4" class="ml-auto" />
        <Skeleton width="w-16" height="h-4" />
        <Skeleton width="w-14" height="h-7" rounded="rounded-xl" />
      </div>
    {/each}
  </div>
{:else if products.length === 0}
  <div class="px-4 py-12 text-center">
    <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
      <Package size={32} class="text-text-muted" />
    </div>
    <p class="text-text-primary font-semibold mt-4">{labels.noProductsFound}</p>
    <p class="text-text-muted text-sm mt-1">{labels.addProductsToStartSelling}</p>
  </div>
{:else}
  <div class="flex-1 overflow-y-auto" bind:this={element}>
    <table class="w-full table-fixed">
      <thead class="sticky top-0 bg-bg-secondary z-10 shadow-sm">
        <tr>
          <th class="p-4 w-52 font-semibold">{labels.productName}</th>
          <th class="p-4 text-right w-20 font-semibold">{labels.stock}</th>
          <th class="p-4 text-right w-28 font-semibold">{labels.price}</th>
          <th class="p-4 text-right w-20 font-semibold"></th>
        </tr>
      </thead>
      <tbody>
        {#each products as product, idx (product.id)}
          <tr
            class="border-t border-border transition-colors cursor-pointer {idx === selectedIndex ? 'bg-primary/10 hover:bg-primary/15' : 'hover:bg-surface-hover/50'}"
            tabindex="-1"
            onclick={() => selectedIndex = idx}
            ondblclick={() => { selectedIndex = idx; onaddtocart(product); }}
          >
            <td class="p-4 w-52">
              <div class="font-medium truncate w-full text-text-primary" title={product.name}>
                {product.name}
              </div>
              <div class="flex items-baseline gap-2 mt-1 text-xs text-text-muted">
                <span class="flex items-center gap-1">
                  {product.sku}
                  <button
                    class="p-0.5 hover:text-primary transition-colors"
                    title={labels.copySku}
                    aria-label={labels.copySku}
                    onclick={() => oncopy(product.sku, `sku_${product.id}`)}
                  >
                    {#if showCopySuccess?.has(`sku_${product.id}`)}
                      <span class="text-sm text-primary font-semibold">✓</span>
                    {:else}
                      <Copy size={14} class="text-text-muted hover:text-primary" />
                    {/if}
                  </button>
                </span>
                {#if product.barcode}
                  <span class="flex items-center gap-1 ml-4">
                     {product.barcode}
                    <button
                      class="p-0.5 hover:text-primary transition-colors"
                      title={labels.copyBarcode}
                      aria-label={labels.copyBarcode}
                      onclick={() => oncopy(product.barcode, `barcode_${product.id}`)}
                    >
                      {#if showCopySuccess?.has(`barcode_${product.id}`)}
                        <span class="text-sm text-primary font-semibold">✓</span>
                      {:else}
                        <Copy size={14} class="text-text-muted hover:text-primary" />
                      {/if}
                    </button>
                  </span>
                {/if}
              </div>
            </td>
            <td class="p-4 text-right w-20">
              {#if product.stock === 0}
                <Badge variant="danger" size="sm">0</Badge>
              {:else if product.stock <= criticalThreshold}
                <Badge variant="danger" size="sm">{product.stock}</Badge>
              {:else if product.stock <= warningThreshold}
                <Badge variant="warning" size="sm">{product.stock}</Badge>
              {:else}
                <Badge variant="success" size="sm">{product.stock}</Badge>
              {/if}
            </td>
            <td class="p-4 text-right font-semibold text-text-primary w-28">
              {product.price?.toLocaleString('id-ID')}
            </td>
            <td class="p-4 text-right w-20">
              <Button
                variant="primary"
                size="sm"
                onclick={() => onaddtocart(product)}
                disabled={product.stock === 0}
              >
                <Plus size={14} /> {labels.add}
              </Button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
  <div class="p-4 bg-surface-subtle/30 border-t border-border/50">
    <Pagination {total} {limit} {offset} onPageChange={onpagechange} />
  </div>
{/if}

<style></style>
