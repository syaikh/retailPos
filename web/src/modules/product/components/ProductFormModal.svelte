<script lang="ts">
  import { Button, CurrencyInput, Input, Modal } from '$shared/ui';
  import { Search, X, ChevronDown, Percent } from 'lucide-svelte';
  import { getPricingRules } from '$modules/pricing/services/pricing-service';
  import type { PricingRule } from '$modules/pricing/types';
  import { useRBAC } from '$shared/composables/useRBAC.svelte';
  import { Permissions } from '$shared/constants/permissions';

  let {
    open = $bindable(false),
    mode = $bindable('add'),
    form = $bindable({
      name: '',
      sku: '',
      barcode: '',
      category: '',
      brand_id: null as number | null,
      price: 0,
      cost: 0,
      stock: 0,
      unit_of_measure_id: null as number | null,
      tax_class_id: null as number | null,
      weight_grams: null as number | null,
      description: '',
      status: 'draft'
    }),
    brands = [] as Array<{id: number, name: string}>,
    unitsOfMeasure = [] as Array<{id: number, name: string, code: string}>,
    taxClasses = [] as Array<{id: number, name: string, rate_percent: number}>,
    categories = [] as string[],
    modalCategorySearch = $bindable(''),
    saving = false,
    onSubmit,
    onCancel,
  } = $props();

  let fieldErrors = $state<Record<string, string>>({});

  const rbac = useRBAC();
  let canArchive = $derived(rbac.can(Permissions.product.delete));

  function validate(): boolean {
    const errors: Record<string, string> = {};
    if (!form.name.trim()) errors.name = 'Name is required';
    if (!form.sku.trim()) errors.sku = 'SKU is required';
    if (!form.category.trim()) errors.category = 'Category is required';
    if (form.price <= 0) errors.price = 'Price must be greater than zero';
    if (form.stock < 0) errors.stock = 'Stock must not be negative';
    fieldErrors = errors;
    return Object.keys(errors).length === 0;
  }

  let showModalCategoryDropdown = $state(false);
  let pricingRules = $state<PricingRule[]>([]);
  let loadingPricing = $state(false);
  let categoryContainer: HTMLDivElement;
  let categoryMenuStyle = $state('');

  function computeCategoryPosition() {
    if (!categoryContainer) return;
    const r = categoryContainer.getBoundingClientRect();
    categoryMenuStyle = `position:fixed;top:${r.bottom + 8}px;left:${r.left}px;width:${r.width}px`;
  }

  $effect(() => {
    if (!showModalCategoryDropdown) return;
    computeCategoryPosition();
    function reposition() { computeCategoryPosition(); }
    window.addEventListener('scroll', reposition, { passive: true, capture: true });
    window.addEventListener('resize', reposition, { passive: true });
    return () => {
      window.removeEventListener('scroll', reposition, { capture: true } as EventListenerOptions);
      window.removeEventListener('resize', reposition);
    };
  });

  let filteredModalCategories = $derived(
    categories.filter(cat =>
      cat !== 'All' && cat.toLowerCase().includes(modalCategorySearch.toLowerCase())
    )
  );

  let productFormId = $derived((form as any).id as number | undefined);

  $effect(() => {
    if (open && mode === 'edit' && productFormId) {
      loadPricingRules(productFormId);
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

  function formatCurrency(value: number): string {
    return 'Rp ' + value.toLocaleString('id-ID');
  }

  function selectModalCategory(category: string) {
    form.category = category;
    modalCategorySearch = category;
    showModalCategoryDropdown = false;
  }

  function handleModalCategoryFocus() {
    showModalCategoryDropdown = true;
  }

  function handleModalCategoryBlur() {
    setTimeout(() => {
      showModalCategoryDropdown = false;
    }, 150);
  }

  function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!validate()) return;
    onSubmit();
  }
</script>

<Modal bind:open title={mode === 'add' ? 'Add Product' : 'Edit Product'}>
  <form onsubmit={handleSubmit} class="space-y-4">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label for="prod-name" class="block text-sm font-medium text-text-secondary mb-2">Name <span class="text-destructive">*</span></label>
<Input id="prod-name" bind:value={form.name} type="text" error={fieldErrors.name} required />
      </div>
      <div>
        <label for="prod-sku" class="block text-sm font-medium text-text-secondary mb-2">SKU <span class="text-destructive">*</span></label>
<Input id="prod-sku" bind:value={form.sku} type="text" error={fieldErrors.sku} required />
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label for="prod-barcode" class="block text-sm font-medium text-text-secondary mb-2">Barcode <span class="text-text-muted text-xs">(optional)</span></label>
        <Input id="prod-barcode" bind:value={form.barcode} type="text" placeholder="Optional barcode" />
      </div>
      <div>
        <label for="prod-category" class="block text-sm font-medium text-text-secondary mb-2">Category <span class="text-destructive">*</span></label>
        <div bind:this={categoryContainer} class="relative">
          <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
          <Input
            type="text"
            id="prod-category"
            placeholder="Select a category"
            bind:value={modalCategorySearch}
            oninput={() => form.category = modalCategorySearch}
            onfocus={handleModalCategoryFocus}
            onblur={handleModalCategoryBlur}
            class="pl-10 pr-10"
            error={fieldErrors.category}
          />
          {#if modalCategorySearch}
            <button
              type="button"
              onclick={() => { modalCategorySearch = ''; form.category = ''; }}
              class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
              title="Clear"
              aria-label="Clear"
            >
              <X size={14} />
            </button>
          {:else}
            <ChevronDown size={16} class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
          {/if}
          {#if showModalCategoryDropdown}
            <div style={categoryMenuStyle} class="fixed z-50 card-glass p-1.5 min-w-0 flex flex-col gap-0.5 max-h-48 overflow-y-auto">
              {#if filteredModalCategories.length === 0}
                <div class="px-3 py-2 text-sm text-text-muted">No categories found</div>
              {:else}
                {#each filteredModalCategories as cat}
                  <button
                    type="button"
                    onmousedown={() => selectModalCategory(cat)}
                    class="flex items-center gap-3 px-3 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-xl transition-all duration-200 active:scale-[0.98] w-full text-left"
                    role="menuitem"
                  >
                    {cat}
                  </button>
                {/each}
              {/if}
            </div>
          {/if}
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div>
        <label for="prod-brand" class="block text-sm font-medium text-text-secondary mb-2">Brand</label>
        <Input tag="select" id="prod-brand" bind:value={form.brand_id}>
          <option value={null}>Select brand</option>
          {#each brands as brand}
            <option value={brand.id}>{brand.name}</option>
          {/each}
        </Input>
      </div>
      <div>
        <label for="prod-uom" class="block text-sm font-medium text-text-secondary mb-2">Unit</label>
        <Input tag="select" id="prod-uom" bind:value={form.unit_of_measure_id}>
          <option value={null}>Select unit</option>
          {#each unitsOfMeasure as uom}
            <option value={uom.id}>{uom.name} ({uom.code})</option>
          {/each}
        </Input>
      </div>
      <div>
        <label for="prod-tax" class="block text-sm font-medium text-text-secondary mb-2">Tax Class</label>
        <Input tag="select" id="prod-tax" bind:value={form.tax_class_id}>
          <option value={null}>Select tax</option>
          {#each taxClasses as tax}
            <option value={tax.id}>{tax.name} ({tax.rate_percent}%)</option>
          {/each}
        </Input>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div>
        <label for="prod-price" class="block text-sm font-medium text-text-secondary mb-2">Price (IDR) <span class="text-destructive">*</span></label>
        <CurrencyInput id="prod-price" bind:value={form.price} required />
        {#if fieldErrors.price}
          <p class="text-xs text-danger mt-1" role="alert">{fieldErrors.price}</p>
        {/if}
      </div>
      <div>
        <label for="prod-cost" class="block text-sm font-medium text-text-secondary mb-2">Cost (IDR)</label>
        <CurrencyInput id="prod-cost" bind:value={form.cost} />
      </div>
      <div>
        <label for="prod-stock" class="block text-sm font-medium text-text-secondary mb-2">Stock <span class="text-destructive">*</span></label>
        <Input id="prod-stock" bind:value={form.stock} type="number" error={fieldErrors.stock} required />
      </div>
    </div>

    <div>
      <label for="prod-description" class="block text-sm font-medium text-text-secondary mb-2">Description</label>
      <Input tag="textarea" id="prod-description" bind:value={form.description} rows="2" placeholder="Product description (optional)" />
    </div>

    <div>
      <label for="prod-status" class="block text-sm font-medium text-text-secondary mb-2">Status</label>
      <Input tag="select" id="prod-status" bind:value={form.status}>
        <option value="draft">Draft</option>
        <option value="active">Active</option>
        <option value="inactive">Inactive</option>
        <option value="discontinued">Discontinued</option>
        {#if canArchive}
        <option value="archived">Archived</option>
        {/if}
      </Input>
    </div>

    {#if mode === 'edit' && productFormId}
      <div class="rounded-xl border border-border bg-surface-default overflow-hidden">
        <div class="px-4 py-2.5 border-b border-border/60 flex items-center gap-2">
          <Percent size={14} class="text-primary-light" />
          <span class="text-xs font-semibold uppercase tracking-wide text-text-muted">Pricing Rules</span>
          {#if pricingRules.length > 0}
            <span class="text-[10px] font-medium text-text-muted bg-surface px-1.5 py-0.5 rounded-full">{pricingRules.length}</span>
          {/if}
        </div>
        <div class="px-4 py-3">
          {#if loadingPricing}
            <p class="text-xs text-text-muted">Loading pricing rules...</p>
          {:else if pricingRules.length === 0}
            <p class="text-xs text-text-muted">No pricing rules configured. Base price: {formatCurrency(form.price)}</p>
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
                      <span class="text-text-muted">min {rule.minimum_quantity}</span>
                    {/if}
                    <span class="text-text-primary font-semibold">{formatCurrency(rule.pricing_value)}</span>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    {/if}

    <div class="flex justify-end gap-4 pt-4">
      <Button variant="secondary" class="px-5 disabled:opacity-50 disabled:cursor-not-allowed" onclick={onCancel}>
        Cancel
      </Button>
      <Button variant="primary" type="submit" class="shadow-glow-primary-sm px-5 disabled:opacity-50 disabled:cursor-not-allowed" disabled={saving}>
        {saving ? 'Saving...' : mode === 'add' ? 'Add' : 'Update'}
      </Button>
    </div>
  </form>
</Modal>
