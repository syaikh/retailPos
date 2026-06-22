<script lang="ts">
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import { Search, X, ChevronDown } from 'lucide-svelte';

  let {
    open = $bindable(false),
    mode = 'add',
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
    isSuperAdmin = false,
    isAdmin = false,
    onSubmit,
    onCancel,
  } = $props();

  let showModalCategoryDropdown = $state(false);

  let filteredModalCategories = $derived(
    categories.filter(cat =>
      cat !== 'All' && cat.toLowerCase().includes(modalCategorySearch.toLowerCase())
    )
  );

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
    onSubmit();
  }
</script>

<Modal bind:open title={mode === 'add' ? 'Add Product' : 'Edit Product'}>
  <form onsubmit={handleSubmit} class="space-y-4">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label for="prod-name" class="block text-sm font-medium text-text-secondary mb-2">Name <span class="text-destructive">*</span></label>
<Input id="prod-name" bind:value={form.name} type="text" required />
      </div>
      <div>
        <label for="prod-sku" class="block text-sm font-medium text-text-secondary mb-2">SKU <span class="text-destructive">*</span></label>
<Input id="prod-sku" bind:value={form.sku} type="text" required />
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label for="prod-barcode" class="block text-sm font-medium text-text-secondary mb-2">Barcode <span class="text-text-muted text-xs">(optional)</span></label>
        <Input id="prod-barcode" bind:value={form.barcode} type="text" placeholder="Optional barcode" />
      </div>
      <div>
        <label for="prod-category" class="block text-sm font-medium text-text-secondary mb-2">Category <span class="text-destructive">*</span></label>
        <div class="relative">
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
            required
          />
          {#if modalCategorySearch}
            <button
              type="button"
              onclick={() => { modalCategorySearch = ''; form.category = ''; }}
              class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
              title="Clear"
            >
              <X size={14} />
            </button>
          {:else}
            <ChevronDown size={16} class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
          {/if}
          {#if showModalCategoryDropdown}
            <div class="absolute top-full mt-2 w-full card-glass p-1.5 z-50 min-w-0 flex flex-col gap-0.5 max-h-48 overflow-y-auto">
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
        <Input id="prod-price" bind:value={form.price} type="number" required />
      </div>
      <div>
        <label for="prod-cost" class="block text-sm font-medium text-text-secondary mb-2">Cost (IDR)</label>
        <Input id="prod-cost" bind:value={form.cost} type="number" />
      </div>
      <div>
        <label for="prod-stock" class="block text-sm font-medium text-text-secondary mb-2">Stock <span class="text-destructive">*</span></label>
        <Input id="prod-stock" bind:value={form.stock} type="number" required />
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
        {#if isSuperAdmin || isAdmin}
        <option value="archived">Archived</option>
        {/if}
      </Input>
    </div>

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
