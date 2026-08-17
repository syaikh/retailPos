<script lang="ts">
  import { onMount } from 'svelte';
  import { usePurchaseOrderStore } from '../stores/po-store.svelte';
  import { getSuppliers } from '$modules/supplier/services/supplier-service';
  import { getProductsBySupplier } from '$modules/supplier/services/supplier-service';
  import { getActiveStores } from '$modules/stores/services/stores-service';
  import { Button, Input, Modal, SelectSearch } from '$shared/ui';
  import CurrencyInput from '$shared/ui/CurrencyInput.svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { labels, t } from '$shared/i18n';
  import { Loader2, ChevronLeft, ChevronRight } from 'lucide-svelte';
  import { getTodayInJakarta } from '$shared/utils/jakartaTime';
  import { useAuthStore } from '$modules/auth';

  const store = usePurchaseOrderStore();

  let {
    open = $bindable(false),
  }: {
    open?: boolean;
  } = $props();

  let saving = $state(false);
  let currentStep = $state(1);
  let supplierProducts = $state<any[]>([]);
  let loadingProducts = $state(false);

  let po = $state<any>(store.selectedPO || {
    supplier_id: 0,
    expected_date: '',
    payment_term: 'Cash on Delivery',
    delivery_address: '',
    supplier_reference_number: '',
    notes: '',
    items: [],
  });
  let suppliers = $state<any[]>([]);
  let stores = $state<any[]>([]);
  let selectedStoreId = $state<number | null>(null);
  let customPaymentTerm = $state('');

  const PAYMENT_TERMS = [
    'Cash on Delivery',
    'Net 15',
    'Net 30',
    'Net 60',
    'Net 90',
    'Due on Receipt',
    '50% Upfront, 50% on Delivery',
  ];

  let selectedPaymentTerm = $state('Cash on Delivery');
  const todayJakarta = $derived(getTodayInJakarta());

  $effect(() => {
    if (store.selectedPO) {
      po = { ...store.selectedPO, items: store.selectedPO.items?.map((item: any) => ({ ...item })) || [] };
      selectedStoreId = store.selectedPO.store_id ?? useAuthStore().user?.store_id ?? null;
      open = true;
      currentStep = 1;
      if (!selectedStoreId) {
        loadStores();
      }
    }
  });

  $effect(() => {
    if (open && !store.selectedPO) {
      currentStep = 1;
      po = {
        supplier_id: 0,
        expected_date: '',
        payment_term: 'Cash on Delivery',
        delivery_address: '',
        supplier_reference_number: '',
        notes: '',
        items: [],
      };
      supplierProducts = [];
      const userStoreId = useAuthStore().user?.store_id ?? null;
      selectedStoreId = userStoreId;
      if (!userStoreId) {
        loadStores();
      }
    }
  });

  $effect(() => {
    if (PAYMENT_TERMS.includes(po.payment_term)) {
      selectedPaymentTerm = po.payment_term;
      customPaymentTerm = '';
    } else {
      selectedPaymentTerm = 'Other';
      customPaymentTerm = po.payment_term;
    }
  });

  $effect(() => {
    if (po.supplier_id > 0) {
      loadSupplierProducts(po.supplier_id);
    }
  });

  $effect(() => {
    if (currentStep === 2 && !store.selectedPO && po.items.length === 0) {
      addItem();
    }
  });

  function handlePaymentTermChange(term: string) {
    selectedPaymentTerm = term;
    if (term === 'Other') {
      po.payment_term = customPaymentTerm;
    } else {
      po.payment_term = term;
    }
  }

  async function loadSupplierProducts(supplierId: number) {
    loadingProducts = true;
    try {
      const data = await getProductsBySupplier(supplierId);
      supplierProducts = data || [];
    } catch {
      supplierProducts = [];
    } finally {
      loadingProducts = false;
    }
  }

  async function loadStores() {
    try {
      const data = await getActiveStores();
      stores = data || [];
    } catch {
      stores = [];
    }
  }

  function nextStep() {
    if (po.supplier_id === 0) return;
    currentStep = 2;
  }

  function prevStep() {
    currentStep = 1;
  }

  onMount(async () => {
    const suppRes = await getSuppliers({ limit: 100, offset: 0 });
    suppliers = suppRes.data || [];
  });

  function displayNum(n: number): string {
    return n ? n.toLocaleString('id-ID') : '';
  }

  function addItem() {
    po.items = [...po.items, { product_id: 0, qty_ordered: 1, unit_cost: 0, discount_amount: 0 }];
  }

  function removeItem(index: number) {
    po.items = po.items.filter((_: any, i: number) => i !== index);
  }

  function calculateSubtotal(item: any) {
    return item.qty_ordered * item.unit_cost - item.discount_amount;
  }

  function getTotalSubtotal() {
    return po.items.reduce((sum: number, item: any) => sum + calculateSubtotal(item), 0);
  }

  async function handleSubmit() {
    saving = true;
    try {
      const items = po.items
        .filter((item: any) => item.product_id > 0 && item.qty_ordered > 0)
        .map((item: any) => ({
          product_id: Number(item.product_id),
          qty_ordered: Number(item.qty_ordered),
          unit_cost: Number(item.unit_cost),
          discount_amount: Number(item.discount_amount || 0),
        }));
      if (items.length === 0) {
        toast.error(labels.pleaseAddItemWithProduct);
        saving = false;
        return;
      }
      const payload = {
        supplier_id: Number(po.supplier_id),
        store_id: selectedStoreId,
        expected_date: po.expected_date,
        payment_term: po.payment_term,
        delivery_address: po.delivery_address,
        supplier_reference_number: po.supplier_reference_number,
        notes: po.notes,
        items,
      };
      if (store.selectedPO) {
        await store.update(store.selectedPO.id, payload);
        toast.success(labels.poUpdated);
      } else {
        await store.create(payload);
        toast.success(labels.poCreated);
      }
      open = false;
      store.load(store.currentFilters);
    } catch (e: any) {
      toast.error(e.message || labels.failedToSavePurchaseOrder);
    } finally {
      saving = false;
    }
  }

  function handleClose() {
    open = false;
    store.selectedPO = null;
    currentStep = 1;
  }
</script>

<Modal bind:open title={store.selectedPO ? labels.editPurchaseOrder : labels.createPurchaseOrder} size="xl" panelClass="max-w-6xl">
  <div class="flex items-center gap-2 mb-4">
    <span class="w-7 h-7 rounded-full bg-primary-default text-white text-xs font-bold flex items-center justify-center">1</span>
    <span class="text-sm font-medium text-text-primary">{labels.poDetails}</span>
    <span class="text-text-muted text-sm mx-1">→</span>
    <span class="w-7 h-7 rounded-full {currentStep === 2 ? 'bg-primary-default text-white' : 'bg-muted text-text-muted'} text-xs font-bold flex items-center justify-center">2</span>
    <span class="text-sm {currentStep === 2 ? 'font-medium text-text-primary' : 'text-text-muted'}">{labels.items}</span>
  </div>

  {#if currentStep === 1}
    <div class="min-h-[340px] grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div>
        <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
          <span>{labels.supplier} <span class="text-danger">*</span></span>
          <SelectSearch
            bind:value={po.supplier_id}
            options={suppliers.map(s => ({ value: s.id, label: s.name }))}
            placeholder={labels.selectSupplier}
          />
        </label>
      </div>
      {#if !useAuthStore().user?.store_id}
        <div>
          <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
            <span>{labels.store} <span class="text-danger">*</span></span>
            <SelectSearch
              bind:value={selectedStoreId}
              options={stores.map((s: any) => ({ value: s.id, label: s.name }))}
              placeholder={labels.pilihToko}
            />
          </label>
        </div>
      {/if}
      <div>
        <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
          <span>{labels.expectedDate} <span class="text-danger">*</span></span>
          <Input type="date" bind:value={po.expected_date} min={todayJakarta} selectOnFocus />
        </label>
      </div>
      <div>
        <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
          <span>{labels.paymentTerm} <span class="text-danger">*</span></span>
          <Input tag="select" bind:value={selectedPaymentTerm} onchange={(e: Event) => handlePaymentTermChange((e.target as HTMLSelectElement).value)}>
            <option value="" disabled>{labels.selectPaymentTerm}</option>
            {#each PAYMENT_TERMS as term}
              <option value={term}>{term}</option>
            {/each}
            <option value="Other">{labels.other}...</option>
          </Input>
        </label>
        {#if selectedPaymentTerm === 'Other'}
          <Input type="text" bind:value={customPaymentTerm} oninput={() => { po.payment_term = customPaymentTerm; }} placeholder={labels.enterCustomTerm} class="mt-2" required selectOnFocus />
        {/if}
      </div>
      <div>
        <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
          <span>{labels.supplierReferenceNumber}</span>
          <Input type="text" bind:value={po.supplier_reference_number} selectOnFocus />
        </label>
      </div>
      <div class="sm:col-span-2">
        <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
          <span>{labels.deliveryAddress}</span>
          <Input tag="textarea" bind:value={po.delivery_address} rows={2} selectOnFocus />
        </label>
      </div>
      <div class="sm:col-span-2">
        <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
          <span>{labels.notes}</span>
          <Input tag="textarea" bind:value={po.notes} rows={2} selectOnFocus />
        </label>
      </div>
    </div>
  {:else}
    <div class="min-h-[340px]">
    <div class="bg-muted/30 rounded-xl px-4 py-3 text-sm text-text-secondary mb-4">
      {labels.supplier}: <span class="font-medium text-text-primary">{suppliers.find(s => s.id === po.supplier_id)?.name || labels.unknown}</span>
    </div>

    {#if loadingProducts}
      <div class="flex items-center justify-center py-8">
        <Loader2 size={24} class="animate-spin text-text-muted" />
      </div>
    {:else}
      <div>
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-base font-semibold text-text-primary">{labels.items}</h3>
          <Button variant="secondary" size="sm" onclick={addItem} disabled={supplierProducts.length === 0}>{labels.addItem}</Button>
        </div>

        {#if supplierProducts.length === 0}
          <p class="text-text-muted text-sm">{labels.noProductsForSupplier}</p>
        {/if}

        {#if po.items.length === 0}
          <p class="text-text-muted text-sm">{labels.noItemsAdded}</p>
        {:else}
          <div class="border border-border rounded-xl">
            <table class="w-full">
              <thead class="bg-muted/50">
                <tr class="border-b text-left text-xs text-text-muted">
                  <th class="px-3 py-2 font-semibold w-[32%]" scope="col">{labels.product}</th>
                  <th class="px-3 py-2 font-semibold text-right w-[12%]" scope="col">{labels.qty}</th>
                  <th class="px-3 py-2 font-semibold text-right w-[22%]" scope="col">{labels.unitCost}</th>
                  <th class="px-3 py-2 font-semibold text-right w-[14%]" scope="col">{labels.discount}</th>
                  <th class="px-3 py-2 font-semibold text-right w-[14%]" scope="col">{labels.subtotal}</th>
                  <th class="px-3 py-2 w-[48px]" scope="col"></th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                {#each po.items as item, index}
                  <tr>
                    <td class="px-3 py-2">
                      <SelectSearch
                        bind:value={item.product_id}
                        options={supplierProducts.map(sp => ({ value: sp.product_id, label: `${sp.product_name || t('productWithId', { id: sp.product_id })} (${sp.product_sku || 'N/A'})` }))}
                        placeholder={labels.selectProduct}
                        searchPlaceholder={labels.searchProducts}
                      />
                    </td>
                    <td class="px-3 py-2">
                      <div class="flex items-center bg-bg-secondary border border-border-default rounded-xl px-3 h-[42px] w-full transition-colors duration-200">
                        <input type="text" inputmode="numeric" value={displayNum(item.qty_ordered)} oninput={(e) => { const el = e.target as HTMLInputElement; const raw = el.value.replace(/[^0-9]/g, ''); item.qty_ordered = raw ? parseInt(raw, 10) : 0; const fmt = displayNum(item.qty_ordered); if (el.value !== fmt) el.value = fmt; }} onfocus={(e) => (e.target as HTMLInputElement).select()} class="w-full bg-transparent text-sm text-right text-text-primary outline-none placeholder:text-text-muted" placeholder="0" />
                      </div>
                    </td>
                    <td class="px-3 py-2">
                      <CurrencyInput bind:value={item.unit_cost} class="w-full text-sm" />
                    </td>
                    <td class="px-3 py-2">
                      <div class="flex items-center bg-bg-secondary border border-border-default rounded-xl px-3 h-[42px] w-full transition-colors duration-200">
                        <input type="text" inputmode="numeric" value={displayNum(item.discount_amount)} oninput={(e) => { const el = e.target as HTMLInputElement; const raw = el.value.replace(/[^0-9]/g, ''); item.discount_amount = raw ? parseInt(raw, 10) : 0; const fmt = displayNum(item.discount_amount); if (el.value !== fmt) el.value = fmt; }} onfocus={(e) => (e.target as HTMLInputElement).select()} class="w-full bg-transparent text-sm text-right text-text-primary outline-none placeholder:text-text-muted" placeholder="0" />
                      </div>
                    </td>
                    <td class="px-3 py-3 text-sm text-text-secondary text-right tabular-nums">{calculateSubtotal(item).toLocaleString('id-ID')}</td>
                    <td class="px-3 py-2">
                      <Button variant="ghost" size="icon" onclick={() => removeItem(index)} aria-label={labels.removeItem} class="text-danger hover:text-danger-light">
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                      </Button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    {/if}
    </div>
  {/if}

  {#snippet footer()}
    <div class="flex items-center justify-between w-full">
      {#if currentStep === 1}
        <div></div>
        <div class="flex items-center gap-2">
          <Button variant="secondary" onclick={handleClose}>{labels.cancel}</Button>
          <Button variant="primary" onclick={nextStep} disabled={po.supplier_id === 0 || !po.expected_date || !po.payment_term || (selectedPaymentTerm === 'Other' && !customPaymentTerm) || (!useAuthStore().user?.store_id && !selectedStoreId)}>
            {labels.next}
            <ChevronRight size={16} />
          </Button>
        </div>
      {:else}
        <div class="text-base font-semibold text-text-primary tabular-nums">
          {labels.total}: {getTotalSubtotal().toLocaleString('id-ID')}
        </div>
        <div class="flex items-center gap-2">
          <Button variant="secondary" onclick={prevStep}>
            <ChevronLeft size={16} />
            {labels.back}
          </Button>
          <Button variant="secondary" onclick={handleClose}>{labels.cancel}</Button>
          <Button variant="primary" onclick={handleSubmit} disabled={saving || po.items.length === 0}>
            {#if saving}
              <Loader2 size={16} class="animate-spin" />
            {/if}
            {saving ? labels.saving : (store.selectedPO ? labels.update : labels.createDraft)}
          </Button>
        </div>
      {/if}
    </div>
  {/snippet}
</Modal>
