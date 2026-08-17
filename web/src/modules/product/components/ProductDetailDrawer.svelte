<script lang="ts">
  import { Badge, Button, Drawer } from '$shared/ui';
  import { Pencil, Trash2, Copy, Percent } from 'lucide-svelte';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
  import { toast } from '$shared/stores/toast.svelte';
  import { getPricingRules } from '$modules/pricing/services/pricing-service';
  import type { PricingRule } from '$modules/pricing/types';
  import RackStockPanel from '$modules/inventory/components/RackStockPanel.svelte';
  import { useRBAC } from '$shared/composables/useRBAC.svelte';
  import { Permissions } from '$shared/constants/permissions';
  import { labels, t } from '$shared/i18n';
  import { formatCurrency as _formatCurrency } from '$shared/utils/currency';

  const rbac = useRBAC();

  let shouldShowProductHistory = $derived(rbac.can(Permissions.product.historyView));

  let {
    selectedProduct,
    showDetailDrawer = $bindable(),
    showCopySuccess = $bindable(),
    warningThreshold = 10,
    criticalThreshold = 5,
    canEdit = false,
    canDelete = false,
    canAdjustStock = false,
    isSensitive = false,
    oncopy = (_value: string, _field: string) => {},
    onedit = () => {},
    ondelete = () => {},
    onstockchanged = () => {},
  }: {
    selectedProduct: any;
    showDetailDrawer: boolean;
    showCopySuccess: any;
    warningThreshold?: number;
    criticalThreshold?: number;
    canEdit?: boolean;
    canDelete?: boolean;
    canAdjustStock?: boolean;
    isSensitive?: boolean;
    oncopy?: (value: string, field: string) => void;
    onedit?: () => void;
    ondelete?: () => void;
    onstockchanged?: () => void;
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

  function statusLabel(status?: string): string {
    switch ((status || '').toLowerCase()) {
      case 'active': return labels.active;
      case 'draft': return labels.draft;
      case 'inactive': return labels.inactive;
      case 'discontinued': return labels.discontinued;
      case 'archived': return labels.archived;
      default: return '- ';
    }
  }

  function statusInfo(status?: string): { variant: 'success' | 'muted' | 'danger'; label: string } {
    switch ((status || '').toLowerCase()) {
      case 'active': return { variant: 'success', label: labels.active };
      case 'draft':
      case 'inactive':
        return { variant: 'muted', label: statusLabel(status) };
      case 'discontinued':
      case 'archived':
        return { variant: 'danger', label: statusLabel(status) };
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
      toast.success(labels.copiedToClipboard);
      setTimeout(() => {
        const removed = new Set(next);
        removed.delete(field);
        showCopySuccess = removed;
      }, ms);
    });
  }

  function formatCurrency(value?: number): string {
    return _formatCurrency(value, '-');
  }

  function formatDate(value?: string): string {
    if (!value) return '-';
    return formatDateTimeInJakarta(value);
  }
</script>

<Drawer bind:open={showDetailDrawer} width={480} ariaLabel={labels.detailProduk}>
  {#if selectedProduct}
    <div class="space-y-4">
    <div class="flex items-center gap-3">
      <h2 class="text-lg font-bold text-text-primary">{labels.detailProduk}</h2>
      <Badge variant={status_.variant} size="sm">{status_.label}</Badge>
    </div>

    <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
      <span class="flex items-center gap-1 min-w-0">
        <span class="text-[11px] font-semibold tracking-widest text-text-muted/60">SKU</span>
        <span class="text-text-secondary font-mono text-sm max-w-[130px] truncate">{selectedProduct.sku || '-'}</span>
        <button type="button" class="p-0.5 rounded transition-colors" title={labels.copySku} aria-label={labels.copySku} onclick={() => copyToClipboard(selectedProduct.sku, 'sku')}>
          {#if showCopySuccess?.has('sku')}
            <span class="text-sm text-primary font-semibold">✓</span>
          {:else}
            <Copy size={11} class="text-text-muted/70 hover:text-primary transition-colors"/>
          {/if}
        </button>
      </span>
      {#if selectedProduct.barcode}
        <span class="flex items-center gap-1 ml-1">
          <span class="text-[11px] font-semibold tracking-widest text-text-muted/60">{labels.barcode}</span>
          <span class="text-text-secondary font-mono text-sm max-w-[150px] truncate">{selectedProduct.barcode}</span>
          <button type="button" class="p-0.5 rounded transition-colors" title={labels.copyBarcode} aria-label={labels.copyBarcode} onclick={() => copyToClipboard(selectedProduct.barcode!, 'barcode')}>
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
        <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">{labels.stokLogistik}</h4>
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
          <span class="text-text-secondary text-xs">{labels.unitOfMeasure}{uomLabel ? `: ${uomLabel}` : ''}</span>
        </div>
        {#if selectedProduct.weight_grams != null}
          <div class="text-right">
            <span class="text-[10px] text-text-muted/60 font-medium uppercase tracking-wider">{labels.beratProduk}</span>
            <p class="text-text-secondary text-xs pt-0.5">
              {selectedProduct.weight_grams >= 1000 ? t('weightKg', { value: (selectedProduct.weight_grams / 1000).toFixed(1) }) : t('weightGram', { value: selectedProduct.weight_grams })}
            </p>
          </div>
        {/if}
        {#if selectedProduct.supplier_name}
          <div class="col-span-2">
            <span class="text-[10px] text-text-muted/60 font-medium uppercase tracking-wider">{labels.supplierUtama}</span>
            <p class="text-text-secondary text-xs pt-0.5">{selectedProduct.supplier_name}</p>
          </div>
        {/if}
        {#if selectedProduct.store_id || selectedProduct.store_name}
          <div class="text-right col-span-2">
            <span class="text-text-secondary text-xs">{selectedProduct.store_name || t('storeWithId', { id: selectedProduct.store_id ?? '-' })}</span>
          </div>
        {/if}
      </div>
    </div>

    <RackStockPanel
      productId={selectedProduct.id}
      productName={selectedProduct.name || ''}
      canAdjust={canAdjustStock}
      onChanged={onstockchanged}
    />

    <div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
      <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
        <span class="text-base leading-none">💰</span>
        <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">{labels.keuangan}</h4>
      </div>
      <div class="p-4 grid grid-cols-2 gap-x-6 gap-y-5">
        <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
          <span class="text-xs font-semibold tracking-wide text-text-muted/80">{labels.hargaJual}</span>
          <p class="text-primary-light text-base font-bold mt-0.5">{formatCurrency(selectedProduct.price)}</p>
        </div>
        <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
          <span class="text-xs font-semibold tracking-wide text-text-muted/80">{labels.diskon}</span>
          <p class="text-text-secondary text-xs mt-0.5">
            {selectedProduct.default_discount_percent != null ? `${selectedProduct.default_discount_percent}%` : '0%'}
          </p>
        </div>
        {#if isSensitive}
          <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">{labels.hargaBeli}</span>
            <p class="text-danger-light text-sm font-semibold mt-0.5">{formatCurrency(selectedProduct.cost ?? 0)}</p>
          </div>
          <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">{labels.margin}</span>
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
            <span class="text-[11px] text-text-muted/50 tracking-wide">{labels.hargaBeliDanMargin}</span>
            <p class="text-text-muted/40 text-sm italic mt-0.5">{labels.hiddenParenthetical}</p>
          </div>
          <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">{labels.margin}</span>
            <p class="text-text-muted/40 text-[11px] italic mt-0.5">{labels.visibleOnlyToAdmins}</p>
          </div>
        {/if}
        <div class="flex flex-col gap-1 col-span-2 pt-1">
          <span class="text-xs font-semibold tracking-wide text-text-muted/80">{labels.pajak}</span>
          <span class="text-text-secondary text-xs">
            {selectedProduct.tax_rate != null ? `${selectedProduct.tax_rate}%` : (selectedProduct.tax_class_id ? t('classWithId', { id: selectedProduct.tax_class_id }) : '-')}
          </span>
        </div>
      </div>
    </div>

    {#if selectedProduct.description}
      <div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
        <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
          <span class="text-base leading-none">📝</span>
          <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">{labels.description}</h4>
        </div>
        <div class="px-3.5 py-2.5">
          <p class="text-text-secondary text-xs leading-relaxed whitespace-pre-wrap break-words">{selectedProduct.description}</p>
        </div>
      </div>
    {/if}

    <div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
      <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
        <Percent size={14} class="text-primary-light" />
        <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">{labels.pricingRules}</h4>
        {#if pricingRules.length > 0}
          <span class="text-[10px] font-medium text-text-muted bg-surface px-1.5 py-0.5 rounded-full">{pricingRules.length}</span>
        {/if}
      </div>
      <div class="px-3.5 py-2.5">
        {#if loadingPricing}
          <p class="text-xs text-text-muted">{labels.loadingPricingRules}</p>
        {:else if pricingRules.length === 0}
          <p class="text-xs text-text-muted">{t('noPricingRulesBasePrice', { price: formatCurrency(selectedProduct.price) })}</p>
        {:else}
          <div class="space-y-2">
            {#each pricingRules as rule}
              <div class="flex items-center justify-between text-xs py-1.5 {rule.is_active ? '' : 'opacity-50'}">
                <div class="flex items-center gap-2">
                  <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold
                    {rule.pricing_type === 'promotion' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
                     rule.pricing_type === 'special_price' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' :
                     'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'}">
                    {rule.pricing_type}
                  </span>
                  <span class="text-text-secondary font-medium">{rule.name}</span>
                </div>
                <div class="flex items-center gap-3">
                  {#if rule.minimum_quantity > 1}
                    <span class="text-text-muted">{t('minWithValue', { qty: rule.minimum_quantity })}</span>
                  {/if}
                  <span class="text-text-primary font-semibold">{formatCurrency(rule.pricing_value)}</span>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>

    {#if shouldShowProductHistory}
      <div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
        <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
          <span class="text-base leading-none">📅</span>
          <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">{labels.auditTrail}</h4>
        </div>
        <div class="px-4 py-3 grid grid-cols-2 gap-x-6 gap-y-3">
          <div>
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">{labels.createdAt}</span>
            <p class="text-text-secondary text-xs mt-1">{formatDate(selectedProduct.created_at)}</p>
          </div>
          <div>
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">{labels.updatedAt}</span>
            <p class="text-text-secondary text-xs mt-1">{formatDate(selectedProduct.updated_at)}</p>
          </div>
        </div>
      </div>
    {/if}
    </div>
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
            <Trash2 size={15} class="mr-1.5" />{labels.deleteProduct}
          </Button>
        {/if}
        <Button
          variant="primary"
          class="flex-1 rounded-xl px-4 h-11 text-sm font-semibold text-white shadow-glow-primary-sm transition-all duration-200"
          onclick={onedit}
        >
          <Pencil size={15} class="mr-1.5" />{labels.editProduk}
        </Button>
      </div>
    {/if}
  {/snippet}
</Drawer>
