<script>
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte.ts';
  import { useAuthStore } from '$modules/auth';
  import { getPricingRules, createPricingRule, updatePricingRule, deletePricingRule, searchProducts, getCustomerGroups, getStores } from '../services/pricing-service.ts';
  import { Button, Input, Modal, Skeleton, SearchBar, Pagination, ConfirmDeleteModal, SortableHeader, Dropdown } from '$shared/ui';
  import { Plus, Pencil, Trash2, DollarSign, Loader2, ChevronDown } from 'lucide-svelte';

  const authStore = useAuthStore();

  let loading = $state(true);
  let rules = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedRule = $state(null);
  let modalMode = $state('add');
  let saving = $state(false);
  let sortBy = $state('name');
  let sortDir = $state('asc');
  let statusFilter = $state('all');
  let typeFilter = $state('all');

  let customerGroups = $state([]);
  let stores = $state([]);
  let productSearchResults = $state([]);
  let productSearchQuery = $state('');
  let productSearchTimeout = null;
  let selectedProductName = $state('');

  const dayOptions = [
    { value: 'mon', label: 'Senin' },
    { value: 'tue', label: 'Selasa' },
    { value: 'wed', label: 'Rabu' },
    { value: 'thu', label: 'Kamis' },
    { value: 'fri', label: 'Jumat' },
    { value: 'sat', label: 'Sabtu' },
    { value: 'sun', label: 'Minggu' }
  ];

  let form = $state({
    product_id: null,
    category_id: null,
    brand_id: null,
    pricing_type: 'default',
    pricing_method: 'fixed_price',
    pricing_value: 0,
    name: '',
    minimum_quantity: 1,
    maximum_quantity: null,
    priority: 0,
    customer_group_id: null,
    store_id: null,
    recurrence_days: [],
    time_from: '',
    time_to: '',
    allow_combine: false,
    is_active: true,
    effective_from: '',
    effective_until: ''
  });

  let canCreate = $derived((authStore.user?.permissions || []).includes('pricing:create'));
  let canEdit = $derived((authStore.user?.permissions || []).includes('pricing:update'));
  let canDelete = $derived((authStore.user?.permissions || []).includes('pricing:delete'));

  const pricingTypes = [
    { value: 'default', label: 'Default' },
    { value: 'price_list', label: 'Daftar Harga' },
    { value: 'promotion', label: 'Promosi' }
  ];

  const pricingMethods = [
    { value: 'fixed_price', label: 'Harga Tetap' },
    { value: 'discount_percent', label: 'Diskon (%)' },
    { value: 'discount_amount', label: 'Diskon (Rp)' },
    { value: 'markup_percent', label: 'Markup (%)' }
  ];

  let typeLabel = $derived(typeFilter === 'all' ? 'All Types' : pricingTypes.find(t => t.value === typeFilter)?.label || typeFilter);

  let sortedRules = $derived.by(() => {
    const sorted = [...rules];
    sorted.sort((a, b) => {
      let av = a[sortBy];
      let bv = b[sortBy];
      if (typeof av === 'string') av = av.toLowerCase();
      if (typeof bv === 'string') bv = bv.toLowerCase();
      if (av < bv) return sortDir === 'asc' ? -1 : 1;
      if (av > bv) return sortDir === 'asc' ? 1 : -1;
      return 0;
    });
    return sorted;
  });

  function targetLabel(rule) {
    if (rule.product_id) return `Product #${rule.product_id}`;
    if (rule.category_id) return `Category #${rule.category_id}`;
    if (rule.brand_id) return `Brand #${rule.brand_id}`;
    return '-';
  }

  function methodLabel(method) {
    return pricingMethods.find(m => m.value === method)?.label || method;
  }

  function valueLabel(rule) {
    switch (rule.pricing_method) {
      case 'fixed_price': return `Rp ${formatPrice(rule.pricing_value)}`;
      case 'discount_percent': return `${rule.pricing_value}%`;
      case 'discount_amount': return `-Rp ${formatPrice(rule.pricing_value)}`;
      case 'markup_percent': return `+${rule.pricing_value}%`;
      default: return rule.pricing_value;
    }
  }

  async function fetchRules() {
    loading = true;
    const params = { limit, offset, search: searchQuery };
    if (statusFilter === 'active') params.is_active = true;
    else if (statusFilter === 'inactive') params.is_active = false;
    if (typeFilter !== 'all') params.pricing_type = typeFilter;
    const result = await getPricingRules(params);
    rules = result.data;
    total = result.total;
    loading = false;
  }

  function handleSort(col) {
    if (sortBy === col) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortBy = col;
      sortDir = 'asc';
    }
  }

  function resetForm() {
    form = {
      product_id: null, category_id: null, brand_id: null,
      pricing_type: 'default', pricing_method: 'fixed_price', pricing_value: 0,
      name: '', minimum_quantity: 1, maximum_quantity: null, priority: 0,
      customer_group_id: null, store_id: null, recurrence_days: [],
      time_from: '', time_to: '', allow_combine: false, is_active: true,
      effective_from: '', effective_until: ''
    };
    selectedProductName = '';
    productSearchResults = [];
    productSearchQuery = '';
  }

  function openAdd() {
    modalMode = 'add';
    resetForm();
    showModal = true;
  }

  function openEdit(rule) {
    modalMode = 'edit';
    selectedRule = rule;
    form = {
      product_id: rule.product_id || null,
      category_id: rule.category_id || null,
      brand_id: rule.brand_id || null,
      pricing_type: rule.pricing_type,
      pricing_method: rule.pricing_method,
      pricing_value: rule.pricing_value,
      name: rule.name,
      minimum_quantity: rule.minimum_quantity,
      maximum_quantity: rule.maximum_quantity || null,
      priority: rule.priority,
      customer_group_id: rule.customer_group_id || null,
      store_id: rule.store_id || null,
      recurrence_days: rule.recurrence_days || [],
      time_from: rule.time_from || '',
      time_to: rule.time_to || '',
      allow_combine: rule.allow_combine || false,
      is_active: rule.is_active,
      effective_from: rule.effective_from ? rule.effective_from.split('T')[0] : '',
      effective_until: rule.effective_until ? rule.effective_until.split('T')[0] : ''
    };
    selectedProductName = rule.product_id ? `Product #${rule.product_id}` : '';
    showModal = true;
  }

  function openDelete(rule) {
    selectedRule = rule;
    showDeleteModal = true;
  }

  function handleProductSearch() {
    clearTimeout(productSearchTimeout);
    productSearchTimeout = setTimeout(async () => {
      if (productSearchQuery.length < 2) { productSearchResults = []; return; }
      productSearchResults = await searchProducts(productSearchQuery, 10);
    }, 300);
  }

  function selectProduct(product) {
    form.product_id = product.id;
    selectedProductName = `${product.name} (${product.sku})`;
    productSearchResults = [];
    productSearchQuery = '';
  }

  function clearProduct() {
    form.product_id = null;
    selectedProductName = '';
  }

  function toggleDay(day) {
    if (form.recurrence_days.includes(day)) {
      form.recurrence_days = form.recurrence_days.filter(d => d !== day);
    } else {
      form.recurrence_days = [...form.recurrence_days, day];
    }
  }

  async function saveRule(e) {
    e.preventDefault();
    if (!form.name) {
      toast.error('Nama rule wajib diisi');
      return;
    }
    if (!form.product_id && !form.category_id && !form.brand_id) {
      toast.error('Pilih minimal satu target (product, kategori, atau brand)');
      return;
    }
    saving = true;
    const payload = { ...form };
    if (payload.effective_from) {
      payload.effective_from = payload.effective_from + 'T00:00:00Z';
    } else {
      delete payload.effective_from;
    }
    if (payload.effective_until) {
      payload.effective_until = payload.effective_until + 'T23:59:59Z';
    } else {
      delete payload.effective_until;
    }
    if (payload.maximum_quantity === null || payload.maximum_quantity === '') delete payload.maximum_quantity;
    if (payload.customer_group_id === null) delete payload.customer_group_id;
    if (payload.store_id === null) delete payload.store_id;
    if (!payload.time_from) delete payload.time_from;
    if (!payload.time_to) delete payload.time_to;
    if (payload.recurrence_days.length === 0) delete payload.recurrence_days;

    let ok;
    if (modalMode === 'add') {
      ok = await createPricingRule(payload);
    } else {
      ok = await updatePricingRule(selectedRule.id, payload);
    }
    saving = false;

    if (ok) {
      toast.success(modalMode === 'add' ? 'Rule berhasil dibuat' : 'Rule berhasil diupdate');
      showModal = false;
      fetchRules();
    } else {
      toast.error('Gagal menyimpan rule');
    }
  }

  async function confirmDelete() {
    if (!selectedRule) return;
    const ok = await deletePricingRule(selectedRule.id);
    if (ok) {
      toast.success('Rule berhasil dihapus');
      showDeleteModal = false;
      fetchRules();
    } else {
      toast.error('Gagal menghapus rule');
    }
  }

  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
    fetchRules();
  }

  let searchTimeout;
  function handleSearch() {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => { offset = 0; fetchRules(); }, 300);
  }

  function handleFilterChange() {
    offset = 0;
    fetchRules();
  }

  function formatPrice(v) {
    return v?.toLocaleString('id-ID') || '0';
  }

  onMount(async () => {
    fetchRules();
    const [cg, st] = await Promise.all([getCustomerGroups(), getStores()]);
    customerGroups = cg;
    stores = st;
  });
</script>

<div class="space-y-5">
  <div class="card p-4">
    <div class="flex items-center gap-3">
      <div class="flex-1">
        <SearchBar bind:value={searchQuery} placeholder="Cari rule..." oninput={handleSearch} inputClass="h-10" />
      </div>
      <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border-default">
        <button
          class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'all'; handleFilterChange(); }}
        >Semua</button>
        <button
          class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'active' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'active'; handleFilterChange(); }}
        >Aktif</button>
        <button
          class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'inactive' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'inactive'; handleFilterChange(); }}
        >Nonaktif</button>
      </div>
      <Dropdown placement="bottom-start" items={[
        { label: 'Semua Tipe', checked: typeFilter === 'all', onclick: () => { typeFilter = 'all'; handleFilterChange(); } },
        ...pricingTypes.map(pt => ({
          label: pt.label,
          checked: typeFilter === pt.value,
          onclick: () => { typeFilter = pt.value; handleFilterChange(); }
        }))
      ]}>
        {#snippet trigger({ toggle })}
          <button
            class="flex items-center gap-2 px-3 h-10 rounded-xl border border-border bg-surface-default text-sm hover:border-border-strong hover:bg-surface-hover transition-colors {typeFilter !== 'all' ? 'text-text-primary' : 'text-text-secondary'}"
            style="min-width: 130px;"
            onclick={toggle}
          >
            <span class="flex-1 text-left truncate">{typeLabel}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
        {/snippet}
      </Dropdown>
      {#if canCreate}
        <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={openAdd}>
          <Plus size={18} /> Tambah Rule
        </Button>
      {/if}
    </div>
  </div>

  <div class="card overflow-hidden">
    {#if loading}
      <div class="overflow-x-auto">
      <table class="w-full" style="table-layout: fixed;">
        <colgroup>
          <col style="width: 20%;" />
          <col style="width: 10%;" />
          <col style="width: 12%;" />
          <col style="width: 15%;" />
          <col style="width: 10%;" />
          <col style="width: 8%;" />
          <col style="width: 8%;" />
          <col style="width: 17%;" />
        </colgroup>
        <thead><tr><th>Nama</th><th>Tipe</th><th>Target</th><th>Metode & Nilai</th><th>Min Qty</th><th>Prioritas</th><th>Status</th><th>Aksi</th></tr></thead>
        <tbody>{#each Array(5) as _}<tr>{#each Array(8) as _}<td><Skeleton class="h-4 w-20" /></td>{/each}</tr>{/each}</tbody>
      </table>
      </div>
    {:else if rules.length === 0}
      <div class="flex flex-col items-center justify-center py-12 text-gray-400">
        <DollarSign class="w-12 h-12 mb-3" />
        <p>Belum ada pricing rules</p>
      </div>
    {:else}
      <div class="overflow-x-auto">
      <table class="w-full min-w-[900px]" style="table-layout: fixed;">
        <colgroup>
          <col style="width: 20%;" />
          <col style="width: 10%;" />
          <col style="width: 12%;" />
          <col style="width: 15%;" />
          <col style="width: 10%;" />
          <col style="width: 8%;" />
          <col style="width: 8%;" />
          <col style="width: 17%;" />
        </colgroup>
        <thead class="bg-muted/50">
          <tr class="border-b text-left text-sm text-text-muted">
            <th class="px-4 py-3 font-semibold">
              <SortableHeader label="NAMA" column="name" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="px-4 py-3 font-semibold">
              <SortableHeader label="TIPE" column="pricing_type" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="px-4 py-3 font-semibold">TARGET</th>
            <th class="px-4 py-3 font-semibold">METODE & NILAI</th>
            <th class="px-4 py-3 font-semibold text-right">
              <SortableHeader label="MIN QTY" column="minimum_quantity" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} align="right" />
            </th>
            <th class="px-4 py-3 font-semibold text-right">
              <SortableHeader label="PRIORITAS" column="priority" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} align="right" />
            </th>
            <th class="px-4 py-3 font-semibold">
              <SortableHeader label="STATUS" column="is_active" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="px-4 py-3 font-semibold text-right">AKSI</th>
          </tr>
        </thead>
        <tbody>
          {#each sortedRules as rule (rule.id)}
            <tr class="border-b border-border hover:bg-surface-hover/50 transition-colors">
              <td class="px-4 py-3 font-medium truncate">{rule.name}</td>
              <td class="px-4 py-3"><span class="inline-flex items-center rounded-md bg-blue-50 px-2 py-1 text-xs font-medium text-blue-700 truncate">{pricingTypes.find(t => t.value === rule.pricing_type)?.label || rule.pricing_type}</span></td>
              <td class="px-4 py-3 text-xs truncate">{targetLabel(rule)}</td>
              <td class="px-4 py-3 text-xs">
                <span class="text-text-muted">{methodLabel(rule.pricing_method)}</span>
                <span class="font-medium tabular-nums">{valueLabel(rule)}</span>
              </td>
              <td class="px-4 py-3 text-right tabular-nums">{rule.minimum_quantity}{rule.maximum_quantity ? `-${rule.maximum_quantity}` : ''}</td>
              <td class="px-4 py-3 text-right tabular-nums">{rule.priority}</td>
              <td class="px-4 py-3">
                {#if rule.is_active}
                  <span class="inline-flex items-center rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700">Aktif</span>
                {:else}
                  <span class="inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600">Nonaktif</span>
                {/if}
              </td>
              <td class="px-4 py-3 text-right">
                <div class="flex items-center justify-end gap-1">
                  {#if canEdit}
                    <Button variant="ghost" size="icon" class="text-text-muted hover:text-primary-light" onclick={() => openEdit(rule)}><Pencil class="w-4 h-4" /></Button>
                  {/if}
                  {#if canDelete}
                    <Button variant="ghost" size="icon" class="text-text-muted hover:text-danger hover:bg-danger-subtle" onclick={() => openDelete(rule)}><Trash2 class="w-4 h-4" /></Button>
                  {/if}
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
      </div>

      {#if !loading && rules.length > 0}
        <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
          <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
        </div>
      {/if}
    {/if}
  </div>
</div>

<Modal bind:open={showModal} title={modalMode === 'add' ? 'Tambah Pricing Rule' : 'Edit Pricing Rule'} size="lg">
  <form onsubmit={saveRule} class="space-y-4">
    <div>
      <label for="rule-name" class="block text-xs font-medium text-text-secondary mb-1">Nama Rule <span class="text-danger">*</span></label>
      <Input id="rule-name" bind:value={form.name} required placeholder="Diskon Member VIP" class="h-9 text-sm" />
    </div>

    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="pricing-type" class="block text-xs font-medium text-text-secondary mb-1">Tipe Harga <span class="text-danger">*</span></label>
        <select id="pricing-type" bind:value={form.pricing_type} class="w-full rounded-xl border border-border px-3 py-2 text-sm bg-surface text-text-primary focus:outline-none focus:ring-2 focus:ring-primary h-9">
          {#each pricingTypes as pt}<option value={pt.value}>{pt.label}</option>{/each}
        </select>
      </div>
      <div>
        <label for="pricing-method" class="block text-xs font-medium text-text-secondary mb-1">Metode Harga <span class="text-danger">*</span></label>
        <select id="pricing-method" bind:value={form.pricing_method} class="w-full rounded-xl border border-border px-3 py-2 text-sm bg-surface text-text-primary focus:outline-none focus:ring-2 focus:ring-primary h-9">
          {#each pricingMethods as pm}<option value={pm.value}>{pm.label}</option>{/each}
        </select>
      </div>
    </div>

    <div class="grid grid-cols-3 gap-3">
      <div>
        <label for="pricing-value" class="block text-xs font-medium text-text-secondary mb-1">Nilai Harga <span class="text-danger">*</span></label>
        <Input id="pricing-value" type="number" bind:value={form.pricing_value} required min="0" step="0.01" placeholder="0" class="h-9 text-sm" />
        <p class="mt-0.5 text-[11px] leading-tight text-text-muted">
          {#if form.pricing_method === 'fixed_price'}Harga tetap per unit
          {:else if form.pricing_method === 'discount_percent'}Persentase diskon (0-100)
          {:else if form.pricing_method === 'discount_amount'}Nominal diskon (Rp)
          {:else if form.pricing_method === 'markup_percent'}Persentase markup (0-500){/if}
        </p>
      </div>
      <div>
        <label for="min-qty" class="block text-xs font-medium text-text-secondary mb-1">Min Qty</label>
        <Input id="min-qty" type="number" bind:value={form.minimum_quantity} min="1" placeholder="1" class="h-9 text-sm" />
      </div>
      <div>
        <label for="max-qty" class="block text-xs font-medium text-text-secondary mb-1">Max Qty</label>
        <Input id="max-qty" type="number" bind:value={form.maximum_quantity} min="1" placeholder="Tanpa batas" class="h-9 text-sm" />
        <p class="mt-0.5 text-[11px] leading-tight text-text-muted">Kosongkan = tanpa batas</p>
      </div>
    </div>

    <div class="border-t pt-3">
      <p class="text-xs font-medium text-text-secondary mb-2">Target</p>
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label for="product-search" class="block text-xs text-text-muted mb-1">Produk</label>
          {#if selectedProductName}
            <div class="flex items-center gap-2 h-9 px-3 rounded-xl border border-border bg-surface text-sm">
              <span class="flex-1 truncate">{selectedProductName}</span>
              <button type="button" onclick={clearProduct} class="text-text-muted hover:text-danger text-xs">x</button>
            </div>
          {:else}
            <div class="relative">
              <Input id="product-search" bind:value={productSearchQuery} oninput={handleProductSearch} placeholder="Cari produk..." class="h-9 text-sm" />
              {#if productSearchResults.length > 0}
                <div class="absolute z-50 mt-1 w-full bg-surface border border-border rounded-xl shadow-lg max-h-48 overflow-auto">
                  {#each productSearchResults as p}
                    <button type="button" onclick={() => selectProduct(p)} class="w-full text-left px-3 py-2 text-sm hover:bg-surface-hover truncate">
                      {p.name} <span class="text-text-muted">({p.sku})</span>
                    </button>
                  {/each}
                </div>
              {/if}
            </div>
          {/if}
        </div>
        <div>
          <label for="category-id" class="block text-xs text-text-muted mb-1">Atau pilih kategori / brand</label>
          <div class="grid grid-cols-2 gap-2">
            <Input id="category-id" type="number" bind:value={form.category_id} placeholder="Category ID" class="h-9 text-sm" />
            <Input id="brand-id" type="number" bind:value={form.brand_id} placeholder="Brand ID" class="h-9 text-sm" />
          </div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-2 gap-3 border-t pt-3">
      <div>
        <label for="customer-group" class="block text-xs text-text-muted mb-1">Customer Group</label>
        <select id="customer-group" bind:value={form.customer_group_id} class="w-full rounded-xl border border-border px-3 py-2 text-sm bg-surface text-text-primary focus:outline-none focus:ring-2 focus:ring-primary h-9">
          <option value={null}>Semua Group</option>
          {#each customerGroups as cg}<option value={cg.id}>{cg.name}</option>{/each}
        </select>
      </div>
      <div>
        <label for="store-id" class="block text-xs text-text-muted mb-1">Toko / Outlet</label>
        <select id="store-id" bind:value={form.store_id} class="w-full rounded-xl border border-border px-3 py-2 text-sm bg-surface text-text-primary focus:outline-none focus:ring-2 focus:ring-primary h-9">
          <option value={null}>Semua Toko</option>
          {#each stores as s}<option value={s.id}>{s.name}</option>{/each}
        </select>
      </div>
    </div>

    <div class="border-t pt-3">
      <p class="text-xs font-medium text-text-secondary mb-2">Jadwal</p>
      <div class="flex flex-wrap gap-1 mb-2">
        {#each dayOptions as day}
          <button type="button" onclick={() => toggleDay(day.value)}
            class="h-7 px-2 rounded-lg text-xs font-medium transition-all {form.recurrence_days.includes(day.value) ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'bg-bg-secondary text-text-muted border border-border-default hover:bg-surface-hover'}">
            {day.label}
          </button>
        {/each}
      </div>
      <p class="text-[11px] text-text-muted mb-2">Kosongkan = setiap hari</p>
      <div class="grid grid-cols-2 gap-3">
        <div>
          <label for="time-from" class="block text-xs text-text-muted mb-1">Dari Jam</label>
          <Input id="time-from" type="time" bind:value={form.time_from} class="h-9 text-sm" />
        </div>
        <div>
          <label for="time-to" class="block text-xs text-text-muted mb-1">Sampai Jam</label>
          <Input id="time-to" type="time" bind:value={form.time_to} class="h-9 text-sm" />
        </div>
      </div>
    </div>

    <div class="grid grid-cols-2 gap-3 border-t pt-3">
      <div>
        <label for="effective-from" class="block text-xs text-text-muted mb-1">Berlaku Dari</label>
        <Input id="effective-from" type="date" bind:value={form.effective_from} class="h-9 text-sm" />
      </div>
      <div>
        <label for="effective-until" class="block text-xs text-text-muted mb-1">Berlaku Sampai</label>
        <Input id="effective-until" type="date" bind:value={form.effective_until} class="h-9 text-sm" />
      </div>
    </div>

    <div class="flex items-center gap-4">
      <label class="flex items-center gap-2">
        <input type="checkbox" bind:checked={form.allow_combine} class="rounded" />
        <span class="text-sm text-text-secondary">Boleh digabung (stacking)</span>
      </label>
      {#if modalMode === 'edit'}
        <label class="flex items-center gap-2">
          <input type="checkbox" bind:checked={form.is_active} class="rounded" />
          <span class="text-sm text-text-secondary">Aktif</span>
        </label>
      {/if}
    </div>
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => showModal = false} disabled={saving}>Batal</Button>
    <Button variant="primary" class="min-w-32" onclick={saveRule} disabled={saving}>
      {#if saving}<Loader2 class="w-4 h-4 mr-2 animate-spin" />{/if}
      {modalMode === 'add' ? 'Buat Rule' : 'Update Rule'}
    </Button>
  {/snippet}
</Modal>

<ConfirmDeleteModal bind:open={showDeleteModal} onconfirm={confirmDelete} />
