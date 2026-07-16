<script>
  import { onMount } from 'svelte';
  import { toast } from '$shared/stores/toast.svelte.ts';
  import { useAuthStore } from '$modules/auth';
  import { getPricingRules, createPricingRule, updatePricingRule, deletePricingRule } from '../services/pricing-service.ts';
  import { Button, Input, Modal, Skeleton, SearchBar, Pagination, ConfirmDeleteModal } from '$shared/ui';
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

  async function fetchRules() {
    loading = true;
    const result = await getPricingRules({ limit, offset, search: searchQuery });
    rules = result.data;
    total = result.total;
    loading = false;
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

  function handlePageChange(e) {
    offset = e.detail.offset;
    fetchRules();
  }

  let searchTimeout;
  function handleSearch() {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => { offset = 0; fetchRules(); }, 300);
  }

  function formatCurrency(v) {
    return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(v);
  }

  onMount(() => fetchRules());
</script>

<div class="space-y-6">
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-2xl font-bold text-gray-900">Pricing Rules</h1>
      <p class="text-sm text-gray-500 mt-1">Manage product pricing rules (discounts, wholesale, promotions)</p>
    </div>
    {#if canCreate}
      <Button variant="primary" onclick={openAdd}>
        <Plus class="w-4 h-4 mr-2" /> Add Rule
      </Button>
    {/if}
  </div>

  <div class="card">
    <SearchBar bind:value={searchQuery} placeholder="Search rules..." oninput={handleSearch} />
  </div>

  <div class="card">
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
      <table class="w-full">
        <thead>
          <tr class="border-b text-left text-sm text-gray-500">
            <th class="px-4 py-3">Name</th>
            <th class="px-4 py-3">Type</th>
            <th class="px-4 py-3">Price</th>
            <th class="px-4 py-3">Min Qty</th>
            <th class="px-4 py-3">Priority</th>
            <th class="px-4 py-3">Status</th>
            <th class="px-4 py-3 text-right">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each rules as rule (rule.id)}
            <tr class="border-b hover:bg-gray-50">
              <td class="px-4 py-3 font-medium">{rule.name}</td>
              <td class="px-4 py-3"><span class="inline-flex items-center rounded-md bg-blue-50 px-2 py-1 text-xs font-medium text-blue-700">{rule.pricing_type}</span></td>
              <td class="px-4 py-3">{formatCurrency(rule.price)}</td>
              <td class="px-4 py-3">{rule.minimum_quantity}</td>
              <td class="px-4 py-3">{rule.priority}</td>
              <td class="px-4 py-3">
                {#if rule.is_active}
                  <span class="inline-flex items-center rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700">Active</span>
                {:else}
                  <span class="inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600">Inactive</span>
                {/if}
              </td>
              <td class="px-4 py-3 text-right">
                {#if canEdit}
                  <Button variant="ghost" size="icon" onclick={() => openEdit(rule)}><Pencil class="w-4 h-4" /></Button>
                {/if}
                {#if canDelete}
                  <Button variant="ghost" size="icon" onclick={() => openDelete(rule)}><Trash2 class="w-4 h-4 text-red-500" /></Button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>

  {#if total > limit}
    <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
  {/if}
</div>

<Modal bind:open={showModal} title={modalMode === 'add' ? 'Add Pricing Rule' : 'Edit Pricing Rule'} size="md">
  <form onsubmit={saveRule} class="space-y-4">
    <Input type="number" label="Product ID" bind:value={form.product_id} required />
    <div>
      <label for="pricing_type" class="block text-sm font-medium text-gray-700 mb-1">Pricing Type</label>
      <select id="pricing_type" bind:value={form.pricing_type} class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm">
        {#each pricingTypes as pt}<option value={pt.value}>{pt.label}</option>{/each}
      </select>
    </div>
    <Input label="Name" bind:value={form.name} required />
    <Input type="number" label="Price" bind:value={form.price} required min="0" />
    <Input type="number" label="Minimum Quantity" bind:value={form.minimum_quantity} required min="1" />
    <Input type="number" label="Priority" bind:value={form.priority} min="0" />
    <Input type="date" label="Effective From" bind:value={form.effective_from} />
    <Input type="date" label="Effective Until" bind:value={form.effective_until} />
    {#if modalMode === 'edit'}
      <div class="flex items-center gap-2">
        <input type="checkbox" bind:checked={form.is_active} id="is_active" class="rounded" />
        <label for="is_active" class="text-sm">Active</label>
      </div>
    {/if}
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => showModal = false}>Cancel</Button>
    <Button variant="primary" onclick={saveRule} disabled={saving}>
      {#if saving}<Loader2 class="w-4 h-4 mr-2 animate-spin" />{/if}
      {modalMode === 'add' ? 'Create' : 'Update'}
    </Button>
  {/snippet}
</Modal>

<ConfirmDeleteModal bind:open={showDeleteModal} onconfirm={confirmDelete} />
