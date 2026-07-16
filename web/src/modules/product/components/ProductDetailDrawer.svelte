<script lang="ts">
  import { Badge, Button, Drawer } from '$shared/ui';
  import { Pencil, Trash2, Copy, Percent } from 'lucide-svelte';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
  import { toast } from '$shared/stores/toast.svelte';
  import { getPricingRules } from '$modules/pricing/services/pricing-service';
  import type { PricingRule } from '$modules/pricing/types';

  let {
    selectedProduct,
    showDetailDrawer = $bindable(),
    showCopySuccess = $bindable(),
    warningThreshold = 10,
    criticalThreshold = 5,
    canEdit = false,
    canDelete = false,
    isSensitive = false,
    isFullAudit = false,
    isSuperAdmin = false,
    isAdmin = false,
    oncopy = (_value: string, _field: string) => {},
    onedit = () => {},
    ondelete = () => {},
  }: {
    selectedProduct: any;
    showDetailDrawer: boolean;
    showCopySuccess: any;
    warningThreshold?: number;
    criticalThreshold?: number;
    canEdit?: boolean;
    canDelete?: boolean;
    isSensitive?: boolean;
    isFullAudit?: boolean;
    isSuperAdmin?: boolean;
    isAdmin?: boolean;
    oncopy?: (value: string, field: string) => void;
    onedit?: () => void;
    ondelete?: () => void;
  } = $props();

  let stock_stk = $derived(selectedProduct?.stock ?? 0);
  let pricingRules = $state<PricingRule[]>([]);
  let loadingPricing = $state(false);

  $effect(() => {
    if (showDetailDrawer && selectedProduct?.id) {
      loadPricingRules(selectedProduct.id);
    } else {
      pricingRules = [];
    }
  });

  async function loadPricingRules(productId: number) {
    loadingPricing = true;
    try {
      const result = await getPricingRules({ limit: 50, offset: 0, product_id: productId });
      pricingRules = result.data;
    } catch {
      pricingRules = [];
    } finally {
      loadingPricing = false;
    }
  }

  function statusInfo(status?: string): { variant: 'success' | 'muted' | 'danger'; label: string } {
    switch ((status || '').toLowerCase()) {
      case 'active': return { variant: 'success', label: 'Active' };
      case 'draft':
      case 'inactive':
        return { variant: 'muted', label: (status || 'Draft').charAt(0).toUpperCase() + (status || 'draft').slice(1) };
      case 'discontinued':
      case 'archived':
        return { variant: 'danger', label: status!.charAt(0).toUpperCase() + status!.slice(1) };
      default: return { variant: 'muted', label: '- ' };
    }
  }

  let status_ = $derived(statusInfo(selectedProduct?.status || 'draft'));

  let margin = $derived.by(() => {
    const p = selectedProduct;
    if (!p) return null;
    return (p.price || 0) - (p.cost || 0);
  });

  let marginPct = $derived.by(() => {
    const p = selectedProduct;
    if (!p) return null;
    const price = p.price;
    const cost = p.cost;
    if (!price || !cost) return null;
    return ((price - cost) / price) * 100;
  });

  let margVal = $derived(margin);
  let margPctVal = $derived(marginPct);
  let margIsLoss = $derived(margVal !== null && margVal < 0);
  let uomLabel = $derived(selectedProduct?.unit_of_measure || null);

  function copyToClipboard(value: string, field: string, ms = 2000): void {
    navigator.clipboard.writeText(value).then(() => {
      const base = showCopySuccess || new Set();
      const next = new Set(base);
      next.add(field);
      showCopySuccess = next;
      toast.success('Copied to clipboard');
      setTimeout(() => {
        const removed = new Set(next);
        removed.delete(field);
        showCopySuccess = removed;
      }, ms);
    });
  }

  function formatCurrency(value?: number): string {
    if (value == null || isNaN(value)) return '-';
    return 'Rp ' + value.toLocaleString('id-ID');
  }

  function formatDate(value?: string): string {
    if (!value) return '-';
    return formatDateTimeInJakarta(value);
  }
</script>

<Drawer bind:open={showDetailDrawer} width={480} ariaLabel="Product detail">
  {#if selectedProduct}
    <div class="flex items-center gap-3 mb-4">
      <h2 class="text-lg font-bold text-text-primary">Detail Produk</h2>
      <Badge variant={status_.variant} size="sm">{status_.label}</Badge>
    </div>

    <div class="flex flex-wrap items-center gap-x-4 gap-y-1 mt-0.5">
      <span class="flex items-center gap-1 min-w-0">
        <span class="text-[11px] font-semibold tracking-widest text-text-muted/60">SKU</span>
        <span class="text-text-secondary font-mono text-sm max-w-[130px] truncate">{selectedProduct.sku || '-'}</span>
        <button type="button" class="p-0.5 rounded transition-colors" title="Salin SKU" aria-label="Salin SKU" onclick={() => copyToClipboard(selectedProduct.sku, 'sku')}>
          {#if showCopySuccess?.has('sku')}
            <span class="text-sm text-primary font-semibold">✓</span>
          {:else}
            <Copy size={11} class="text-text-muted/70 hover:text-primary transition-colors"/>
          {/if}
        </button>
      </span>
      {#if selectedProduct.barcode}
        <span class="flex items-center gap-1 ml-1">
          <span class="text-[11px] font-semibold tracking-widest text-text-muted/60">Barcode</span>
          <span class="text-text-secondary font-mono text-sm max-w-[150px] truncate">{selectedProduct.barcode}</span>
          <button type="button" class="p-0.5 rounded transition-colors" title="Salin barcode" aria-label="Salin barcode" onclick={() => copyToClipboard(selectedProduct.barcode!, 'barcode')}>
            {#if showCopySuccess?.has('barcode')}
              <span class="text-sm text-primary font-semibold">✓</span>
            {:else}
              <Copy size={11} class="text-text-muted/70 hover:text-primary transition-colors"/>
            {/if}
          </button>
        </span>
      {/if}
    </div>

    <div>
      <h3 class="text-lg font-bold text-text-primary leading-tight">{selectedProduct.name || '—'}</h3>
      {#if selectedProduct.category_name || selectedProduct.brand_name}
        <span class="text-sm text-text-muted font-medium mt-1 block">
          {#if selectedProduct.category_name}<span>{selectedProduct.category_name}</span>{/if}
          {#if selectedProduct.category_name && selectedProduct.brand_name}<span class="text-text-muted/40 mx-1.5">•</span>{/if}
          {#if selectedProduct.brand_name}<span>{selectedProduct.brand_name}</span>{/if}
        </span>
      {/if}
    </div>

    <div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
      <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
        <span class="text-base leading-none">📦</span>
        <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">Stok &amp; Logistik</h4>
      </div>
      <div class="px-3.5 py-2.5 grid grid-cols-2 gap-x-4 gap-y-3">
        <div class="flex items-center gap-2">
          {#if stock_stk <= criticalThreshold}
            <span class="inline-flex items-center justify-center h-8 w-8 rounded-lg shrink-0 font-mono text-primary-light text-sm font-bold leading-none" style="background: rgba(239,68,68,0.12);">{stock_stk}</span>
          {:else if stock_stk <= warningThreshold}
            <span class="inline-flex items-center justify-center h-8 w-8 rounded-lg shrink-0 font-mono text-warning-light text-sm font-bold leading-none" style="background: rgba(245,158,11,0.12);">{stock_stk}</span>
          {:else}
            <span class="inline-flex items-center justify-center h-8 w-8 rounded-lg shrink-0 font-mono text-success-light text-sm font-bold leading-none" style="background: rgba(16,185,129,0.12);">{stock_stk}</span>
          {/if}
          <span class="text-text-secondary text-xs">Unit {uomLabel ? `: ${uomLabel}` : ''}</span>
        </div>
        {#if selectedProduct.weight_grams != null}
          <div class="text-right">
            <span class="text-[10px] text-text-muted/60 font-medium uppercase tracking-wider">Berat Produk</span>
            <p class="text-text-secondary text-xs pt-0.5">
              {selectedProduct.weight_grams >= 1000 ? `${(selectedProduct.weight_grams / 1000).toFixed(1)} kg` : `${selectedProduct.weight_grams} gram`}
            </p>
          </div>
        {/if}
        {#if selectedProduct.store_id || selectedProduct.store_name}
          <div class="text-right col-span-2">
            <span class="text-text-secondary text-xs">{selectedProduct.store_name || `Store #${selectedProduct.store_id ?? '-'}`}</span>
          </div>
        {/if}
      </div>
    </div>

    <div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
      <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
        <span class="text-base leading-none">💰</span>
        <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">Keuangan</h4>
      </div>
      <div class="p-4 grid grid-cols-2 gap-x-6 gap-y-5">
        <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
          <span class="text-xs font-semibold tracking-wide text-text-muted/80">Harga Jual</span>
          <p class="text-primary-light text-base font-bold mt-0.5">{formatCurrency(selectedProduct.price)}</p>
        </div>
        <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
          <span class="text-xs font-semibold tracking-wide text-text-muted/80">Diskon</span>
          <p class="text-text-secondary text-xs mt-0.5">
            {selectedProduct.default_discount_percent != null ? `${selectedProduct.default_discount_percent}%` : '0%'}
          </p>
        </div>
        {#if isSensitive}
          <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">Harga Beli</span>
            <p class="text-danger-light text-sm font-semibold mt-0.5">{formatCurrency(selectedProduct.cost ?? 0)}</p>
          </div>
          <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">Margin</span>
            <p class="text-sm font-bold {margIsLoss ? 'text-danger-light' : 'text-emerald-400'} mt-0.5">
              {margVal !== null ? formatCurrency(margVal) : '—'}
              {#if margPctVal !== null}
                <span class="{margIsLoss ? 'text-danger-light/70' : 'text-slate-400'} not-italic font-normal text-xs ml-0.5">
                  {margIsLoss ? '-' : ''}{margPctVal.toFixed(1)}%
                </span>
              {/if}
            </p>
          </div>
        {:else}
          <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
            <span class="text-[11px] text-text-muted/50 tracking-wide">Harga Beli &amp; Margin</span>
            <p class="text-text-muted/40 text-sm italic mt-0.5">(tersembunyi)</p>
          </div>
          <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">Margin</span>
            <p class="text-text-muted/40 text-[11px] italic mt-0.5">Hanya tampil untuk admin, manager, dan superadmin</p>
          </div>
        {/if}
        <div class="flex flex-col gap-1 col-span-2 pt-1">
          <span class="text-xs font-semibold tracking-wide text-text-muted/80">Pajak</span>
          <span class="text-text-secondary text-xs">
            {selectedProduct.tax_rate != null ? `${selectedProduct.tax_rate}%` : (selectedProduct.tax_class_id ? `Class #${selectedProduct.tax_class_id}` : '-')}
          </span>
        </div>
      </div>
    </div>

    {#if selectedProduct.description}
      <div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
        <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
          <span class="text-base leading-none">📝</span>
          <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">Deskripsi</h4>
        </div>
        <div class="px-3.5 py-2.5">
          <p class="text-text-secondary text-xs leading-relaxed whitespace-pre-wrap break-words">{selectedProduct.description}</p>
        </div>
      </div>
    {/if}

    <div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
      <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
        <Percent size={14} class="text-primary-light" />
        <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">Aturan Harga</h4>
        {#if pricingRules.length > 0}
          <span class="text-[10px] font-medium text-text-muted bg-surface px-1.5 py-0.5 rounded-full">{pricingRules.length}</span>
        {/if}
      </div>
      <div class="px-3.5 py-2.5">
        {#if loadingPricing}
          <p class="text-xs text-text-muted">Memuat aturan harga...</p>
        {:else if pricingRules.length === 0}
          <p class="text-xs text-text-muted">Tidak ada aturan harga. Harga dasar: {formatCurrency(selectedProduct.price)}</p>
        {:else}
          <div class="space-y-2">
            {#each pricingRules as rule}
              <div class="flex items-center justify-between text-xs py-1.5 {rule.is_active ? '' : 'opacity-50'}">
                <div class="flex items-center gap-2">
                  <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold
                    {rule.pricing_type === 'discount' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
                     rule.pricing_type === 'wholesale' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' :
                     'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'}">
                    {rule.pricing_type}
                  </span>
                  <span class="text-text-secondary font-medium">{rule.name}</span>
                </div>
                <div class="flex items-center gap-3">
                  {#if rule.minimum_quantity > 1}
                    <span class="text-text-muted">min {rule.minimum_quantity}</span>
                  {/if}
                  <span class="text-text-primary font-semibold">{formatCurrency(rule.price)}</span>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>

    {#if isFullAudit}
      <div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
        <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
          <span class="text-base leading-none">📅</span>
          <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">Audit Trail</h4>
        </div>
        <div class="px-4 py-3 grid grid-cols-2 gap-x-6 gap-y-3">
          <div>
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">Dibuat pada</span>
            <p class="text-text-secondary text-xs mt-1">{formatDate(selectedProduct.created_at)}</p>
          </div>
          <div>
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">Diubah pada</span>
            <p class="text-text-secondary text-xs mt-1">{formatDate(selectedProduct.updated_at)}</p>
          </div>
        </div>
      </div>
    {/if}
  {/if}

  {#snippet footer()}
    {#if canEdit}
      <div class="flex items-center gap-3">
        {#if canDelete && selectedProduct?.stock === 0}
          <Button
            variant="secondary"
            class="flex-1 rounded-xl px-4 h-11 text-sm font-semibold text-text-secondary border border-border hover:border-danger hover:text-danger hover:bg-danger-subtle transition-all duration-200"
            onclick={ondelete}
          >
            <Trash2 size={15} class="mr-1.5" />Hapus Produk
          </Button>
        {/if}
        <Button
          variant="primary"
          class="flex-1 rounded-xl px-4 h-11 text-sm font-semibold text-white shadow-glow-primary-sm transition-all duration-200"
          onclick={onedit}
        >
          <Pencil size={15} class="mr-1.5" />Edit Produk
        </Button>
      </div>
    {/if}
  {/snippet}
</Drawer>
