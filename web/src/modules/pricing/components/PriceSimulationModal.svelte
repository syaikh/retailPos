<script lang="ts">
  import { Modal, Button, Input } from '$shared/ui';
  import { SearchBar } from '$shared/ui';
  import { Calculator, Loader2, TrendingDown, TrendingUp, Minus } from 'lucide-svelte';
  import { labels, t } from '$shared/i18n';
  import { searchProducts, resolvePrices, getCustomerGroups, getStores } from '../services/pricing-service';
  import type { ProductSearchResult } from '../types';
  import type { ResolvedPrice } from '../services/pricing-service';

  let {
    open = $bindable(false),
  }: {
    open?: boolean;
  } = $props();

  let productQuery = $state('');
  let productResults = $state<ProductSearchResult[]>([]);
  let selectedProduct = $state<ProductSearchResult | null>(null);
  let quantity = $state(1);
  let customerGroupId = $state<number | ''>('');
  let storeId = $state<number | ''>('');
  let customerGroups = $state<{ id: number; name: string }[]>([]);
  let stores = $state<{ id: number; name: string }[]>([]);
  let resolving = $state(false);
  let result = $state<ResolvedPrice | null>(null);
  let searched = $state(false);

  let canSimulate = $derived(!!selectedProduct && quantity > 0);

  $effect(() => {
    if (open) {
      productQuery = '';
      productResults = [];
      selectedProduct = null;
      quantity = 1;
      customerGroupId = '';
      storeId = '';
      result = null;
      searched = false;
      loadDropdowns();
    }
  });

  async function loadDropdowns() {
    const [cg, st] = await Promise.all([getCustomerGroups(), getStores()]);
    customerGroups = cg;
    stores = st;
  }

  let searchTimeout: ReturnType<typeof setTimeout> | undefined;
  let productSearchContainer: HTMLDivElement | undefined = $state();
  let productMenuStyle = $state('');

  function computeProductSearchPosition() {
    if (!productSearchContainer) return;
    const r = productSearchContainer.getBoundingClientRect();
    productMenuStyle = `position:fixed;top:${r.bottom + 4}px;left:${r.left}px;width:${r.width}px`;
  }

  $effect(() => {
    if (productResults.length === 0) return;
    computeProductSearchPosition();
    function reposition() { computeProductSearchPosition(); }
    window.addEventListener('scroll', reposition, { passive: true, capture: true });
    window.addEventListener('resize', reposition, { passive: true });
    return () => {
      window.removeEventListener('scroll', reposition, { capture: true } as EventListenerOptions);
      window.removeEventListener('resize', reposition);
    };
  });

  function handleProductSearch() {
    clearTimeout(searchTimeout);
    if (productQuery.length < 2) { productResults = []; return; }
    searchTimeout = setTimeout(async () => {
      productResults = await searchProducts(productQuery, 8);
    }, 250);
  }

  function selectProduct(p: ProductSearchResult) {
    selectedProduct = p;
    productQuery = p.name;
    productResults = [];
  }

  function formatCurrency(v: number): string {
    return v.toLocaleString('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 });
  }

  async function simulate() {
    if (!selectedProduct) return;
    resolving = true;
    result = null;
    searched = true;
    try {
      const items = [{
        product_id: selectedProduct.id,
        quantity,
        customer_group_id: typeof customerGroupId === 'number' ? customerGroupId : undefined,
        store_id: typeof storeId === 'number' ? storeId : undefined,
      }];
      const results = await resolvePrices(items);
      result = results[0] || null;
    } catch {
      result = null;
    } finally {
      resolving = false;
    }
  }
</script>

<Modal bind:open title={labels.simulasiHarga} size="md">
  <div class="space-y-5">
    <div>
      <label for="sim-product" class="block text-sm font-medium text-text-primary mb-1.5">{labels.produk}</label>
      {#if selectedProduct}
        <div class="flex items-center justify-between p-3 bg-surface-subtle/50 rounded-xl border border-border">
          <div>
            <p class="text-sm font-medium text-text-primary">{selectedProduct.name}</p>
            <p class="text-xs text-text-muted">{t('skuAndPrice', { sku: selectedProduct.sku, price: formatCurrency(selectedProduct.price) })}</p>
          </div>
          <Button variant="ghost" size="sm" onclick={() => { selectedProduct = null; productQuery = ''; }}>{labels.ganti}</Button>
        </div>
      {:else}
        <div bind:this={productSearchContainer} class="relative">
          <SearchBar bind:value={productQuery} placeholder={labels.searchProducts} oninput={handleProductSearch} id="sim-product" />
          {#if productResults.length > 0}
            <div style={productMenuStyle} class="fixed z-20 bg-surface-default border border-border rounded-xl shadow-lg max-h-48 overflow-y-auto">
              {#each productResults as p}
                <button type="button" class="w-full px-3 py-2 text-left hover:bg-surface-hover transition-colors text-sm" onclick={() => selectProduct(p)}>
                  <span class="font-medium text-text-primary">{p.name}</span>
                  <span class="text-xs text-text-muted ml-2">SKU: {p.sku}</span>
                  <span class="text-xs text-text-muted ml-2">{formatCurrency(p.price)}</span>
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {/if}
    </div>

    <div class="grid grid-cols-3 gap-3">
      <div>
        <label for="sim-qty" class="block text-sm font-medium text-text-primary mb-1.5">{labels.jumlah}</label>
        <Input id="sim-qty" type="number" bind:value={quantity} min={1} />
      </div>
      <div>
        <label for="sim-cg" class="block text-sm font-medium text-text-primary mb-1.5">{labels.customerGroup}</label>
        <select id="sim-cg" bind:value={customerGroupId} class="w-full h-10 px-3 rounded-xl border border-border bg-surface-default text-sm text-text-primary">
          <option value="">{labels.semua}</option>
          {#each customerGroups as cg}
            <option value={cg.id}>{cg.name}</option>
          {/each}
        </select>
      </div>
      <div>
        <label for="sim-store" class="block text-sm font-medium text-text-primary mb-1.5">{labels.toko}</label>
        <select id="sim-store" bind:value={storeId} class="w-full h-10 px-3 rounded-xl border border-border bg-surface-default text-sm text-text-primary">
          <option value="">{labels.semua}</option>
          {#each stores as s}
            <option value={s.id}>{s.name}</option>
          {/each}
        </select>
      </div>
    </div>

    {#if result}
      <div class="p-4 rounded-xl border {result.discount > 0 ? 'bg-success-subtle/20 border-success/30' : result.discount < 0 ? 'bg-warning-subtle/20 border-warning/30' : 'bg-surface-subtle/50 border-border'}">
        <div class="flex items-center gap-2 mb-3">
          {#if result.discount > 0}
            <TrendingDown size={18} class="text-success" />
            <span class="text-sm font-semibold text-success">{labels.hargaDiskon}</span>
          {:else if result.discount < 0}
            <TrendingUp size={18} class="text-warning" />
            <span class="text-sm font-semibold text-warning">{labels.hargaMarkup}</span>
          {:else}
            <Minus size={18} class="text-text-muted" />
            <span class="text-sm font-semibold text-text-primary">{labels.hargaNormal}</span>
          {/if}
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <p class="text-xs text-text-muted">{labels.hargaAsli}</p>
            <p class="text-lg font-bold text-text-primary">{formatCurrency(result.original_price)}</p>
          </div>
          <div>
            <p class="text-xs text-text-muted">{labels.hargaFinal}</p>
            <p class="text-lg font-bold {result.discount > 0 ? 'text-success' : result.discount < 0 ? 'text-warning' : 'text-text-primary'}">{formatCurrency(result.unit_price)}</p>
          </div>
        </div>
        {#if result.rule}
          <div class="mt-3 pt-3 border-t {result.discount > 0 ? 'border-success/20' : result.discount < 0 ? 'border-warning/20' : 'border-border'}">
            <p class="text-xs text-text-muted">{labels.appliedRule}</p>
            <p class="text-sm font-medium text-text-primary">{result.rule.name}</p>
            <p class="text-xs text-text-muted mt-0.5">{t('ruleDetails', { type: result.rule.pricing_type, method: result.rule.pricing_method || '', value: result.rule.pricing_value ?? '' })}</p>
          </div>
        {/if}
      </div>
    {:else if searched && !resolving}
      <div class="p-4 rounded-xl bg-surface-subtle/50 border border-border text-center">
        <p class="text-sm text-text-muted">{labels.noMatchingRule}</p>
        <p class="text-xs text-text-muted mt-1">{labels.basePriceWillBeUsed}</p>
      </div>
    {/if}
  </div>

  {#snippet footer()}
    <Button variant="secondary" onclick={() => open = false}>{labels.close}</Button>
    <Button variant="primary" onclick={simulate} disabled={!canSimulate || resolving}>
      {#if resolving}
        <Loader2 class="w-4 h-4 mr-2 animate-spin" />
      {:else}
        <Calculator size={14} class="mr-1" />
      {/if}
      {labels.calculate}
    </Button>
  {/snippet}
</Modal>
