<script>
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte.ts';
  import { useAuthStore } from '$modules/auth';
  import { getPricingRules, createPricingRule, updatePricingRule, deletePricingRule } from '../services/pricing-service.ts';
  import { Button, Input, Modal, Skeleton, SearchBar, Pagination, ConfirmDeleteModal, SortableHeader } from '$shared/ui';
  import { Plus, Pencil, Trash2, DollarSign, Loader2 } from 'lucide-svelte';

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

  let form = $state({
    product_id: 0,
    pricing_type: 'discount',
    name: '',
    price: 0,
    minimum_quantity: 1,
    priority: 0,
    is_active: true,
    effective_from: '',
    effective_until: ''
  });

  let canCreate = $derived((authStore.user?.permissions || []).includes('pricing:create'));
  let canEdit = $derived((authStore.user?.permissions || []).includes('pricing:update'));
  let canDelete = $derived((authStore.user?.permissions || []).includes('pricing:delete'));

  const pricingTypes = [
    { value: 'discount', label: 'Discount' },
    { value: 'wholesale', label: 'Wholesale' },
    { value: 'member', label: 'Member' },
    { value: 'promotion', label: 'Promotion' }
  ];

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

  async function fetchRules() {
    loading = true;
    const result = await getPricingRules({ limit, offset, search: searchQuery });
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

  function openAdd() {
    modalMode = 'add';
    form = { product_id: 0, pricing_type: 'discount', name: '', price: 0, minimum_quantity: 1, priority: 0, is_active: true, effective_from: '', effective_until: '' };
    showModal = true;
  }

  function openEdit(rule) {
    modalMode = 'edit';
    selectedRule = rule;
    form = {
      product_id: rule.product_id,
      pricing_type: rule.pricing_type,
      name: rule.name,
      price: rule.price,
      minimum_quantity: rule.minimum_quantity,
      priority: rule.priority,
      is_active: rule.is_active,
      effective_from: rule.effective_from ? rule.effective_from.split('T')[0] : '',
      effective_until: rule.effective_until ? rule.effective_until.split('T')[0] : ''
    };
    showModal = true;
  }

  function openDelete(rule) {
    selectedRule = rule;
    showDeleteModal = true;
  }

  async function saveRule(e) {
    e.preventDefault();
    if (!form.name || !form.product_id) {
      toast.error('Name and Product ID are required');
      return;
    }
    saving = true;
    const payload = { ...form };
    if (!payload.effective_from) delete payload.effective_from;
    if (!payload.effective_until) delete payload.effective_until;

    let ok;
    if (modalMode === 'add') {
      ok = await createPricingRule(payload);
    } else {
      ok = await updatePricingRule(selectedRule.id, payload);
    }
    saving = false;

    if (ok) {
      toast.success(modalMode === 'add' ? 'Rule created' : 'Rule updated');
      showModal = false;
      fetchRules();
    } else {
      toast.error('Failed to save rule');
    }
  }

  async function confirmDelete() {
    if (!selectedRule) return;
    const ok = await deletePricingRule(selectedRule.id);
    if (ok) {
      toast.success('Rule deleted');
      showDeleteModal = false;
      fetchRules();
    } else {
      toast.error('Failed to delete rule');
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

  function formatPrice(v) {
    return v.toLocaleString('id-ID');
  }

  onMount(() => fetchRules());
</script>

<div class="space-y-5">
  <div class="card p-4">
    <div class="flex items-center gap-4">
      <div class="flex-2">
        <SearchBar bind:value={searchQuery} placeholder="Search rules..." oninput={handleSearch} inputClass="h-10" />
      </div>
      {#if canCreate}
        <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={openAdd}>
          <Plus size={18} /> Add Rule
        </Button>
      {/if}
    </div>
  </div>

  <div class="card overflow-hidden">
    {#if loading}
      <table class="w-full">
        <thead><tr><th>Name</th><th>Type</th><th>Price</th><th>Min Qty</th><th>Status</th></tr></thead>
        <tbody>{#each Array(5) as _}<tr>{#each Array(5) as _}<td><Skeleton class="h-4 w-20" /></td>{/each}</tr>{/each}</tbody>
      </table>
    {:else if rules.length === 0}
      <div class="flex flex-col items-center justify-center py-12 text-gray-400">
        <DollarSign class="w-12 h-12 mb-3" />
        <p>No pricing rules found</p>
      </div>
    {:else}
      <div class="overflow-x-auto">
      <table class="w-full min-w-[800px]">
        <thead class="bg-muted/50">
          <tr class="border-b text-left text-sm text-text-muted">
            <th class="px-4 py-3 font-semibold">
              <SortableHeader label="NAME" column="name" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="px-4 py-3 font-semibold">
              <SortableHeader label="TYPE" column="pricing_type" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="px-4 py-3 font-semibold text-right">
              <SortableHeader label="RP PRICE" column="price" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} align="right" />
            </th>
            <th class="px-4 py-3 font-semibold text-right">
              <SortableHeader label="MIN QTY" column="minimum_quantity" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} align="right" />
            </th>
            <th class="px-4 py-3 font-semibold text-right">
              <SortableHeader label="PRIORITY" column="priority" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} align="right" />
            </th>
            <th class="px-4 py-3 font-semibold">
              <SortableHeader label="STATUS" column="is_active" sortColumn={sortBy} sortDirection={sortDir} onsort={handleSort} />
            </th>
            <th class="px-4 py-3 font-semibold text-right">ACTIONS</th>
          </tr>
        </thead>
        <tbody>
          {#each sortedRules as rule (rule.id)}
            <tr class="border-b border-border hover:bg-surface-hover/50 transition-colors">
              <td class="px-4 py-3 font-medium">{rule.name}</td>
              <td class="px-4 py-3"><span class="inline-flex items-center rounded-md bg-blue-50 px-2 py-1 text-xs font-medium text-blue-700">{rule.pricing_type}</span></td>
              <td class="px-4 py-3 text-right tabular-nums">{formatPrice(rule.price)}</td>
              <td class="px-4 py-3 text-right tabular-nums">{rule.minimum_quantity}</td>
              <td class="px-4 py-3 text-right tabular-nums">{rule.priority}</td>
              <td class="px-4 py-3">
                {#if rule.is_active}
                  <span class="inline-flex items-center rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700">Active</span>
                {:else}
                  <span class="inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600">Inactive</span>
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

<Modal bind:open={showModal} title={modalMode === 'add' ? 'Add Pricing Rule' : 'Edit Pricing Rule'} size="md">
  <form onsubmit={saveRule} class="space-y-4">
    <div>
      <label for="product_id" class="block text-sm font-medium text-text-secondary mb-1">Product ID <span class="text-danger">*</span></label>
      <Input id="product_id" type="number" bind:value={form.product_id} required min="1" />
    </div>
    <div>
      <label for="pricing_type" class="block text-sm font-medium text-text-secondary mb-1">Pricing Type <span class="text-danger">*</span></label>
      <select id="pricing_type" bind:value={form.pricing_type} class="w-full rounded-xl border border-border px-3 py-2 text-sm bg-surface text-text-primary focus:outline-none focus:ring-2 focus:ring-primary">
        {#each pricingTypes as pt}<option value={pt.value}>{pt.label}</option>{/each}
      </select>
    </div>
    <div>
      <label for="rule_name" class="block text-sm font-medium text-text-secondary mb-1">Name <span class="text-danger">*</span></label>
      <Input id="rule_name" bind:value={form.name} required placeholder="e.g. Wholesale min 5" />
    </div>
    <div>
      <label for="rule_price" class="block text-sm font-medium text-text-secondary mb-1">Price (Rp) <span class="text-danger">*</span></label>
      <Input id="rule_price" type="number" bind:value={form.price} required min="0" />
    </div>
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label for="min_qty" class="block text-sm font-medium text-text-secondary mb-1">Minimum Quantity <span class="text-danger">*</span></label>
        <Input id="min_qty" type="number" bind:value={form.minimum_quantity} required min="1" />
      </div>
      <div>
        <label for="priority" class="block text-sm font-medium text-text-secondary mb-1">Priority <span class="text-text-muted text-xs">(opsional)</span></label>
        <Input id="priority" type="number" bind:value={form.priority} min="0" />
      </div>
    </div>
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label for="effective_from" class="block text-sm font-medium text-text-secondary mb-1">Effective From <span class="text-text-muted text-xs">(opsional)</span></label>
        <Input id="effective_from" type="date" bind:value={form.effective_from} />
      </div>
      <div>
        <label for="effective_until" class="block text-sm font-medium text-text-secondary mb-1">Effective Until <span class="text-text-muted text-xs">(opsional)</span></label>
        <Input id="effective_until" type="date" bind:value={form.effective_until} />
      </div>
    </div>
    {#if modalMode === 'edit'}
      <div class="flex items-center gap-3">
        <input type="checkbox" bind:checked={form.is_active} id="is_active" class="rounded" />
        <label for="is_active" class="text-sm text-text-secondary">Active</label>
      </div>
    {/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => showModal = false} disabled={saving}>Cancel</Button>
    <Button variant="primary" class="min-w-32" onclick={saveRule} disabled={saving}>
      {#if saving}<Loader2 class="w-4 h-4 mr-2 animate-spin" />{/if}
      {modalMode === 'add' ? 'Create' : 'Update'}
    </Button>
  {/snippet}
</Modal>

<ConfirmDeleteModal bind:open={showDeleteModal} onconfirm={confirmDelete} />
