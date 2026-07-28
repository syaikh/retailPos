<script lang="ts">
  import { onMount } from 'svelte';
  import { usePurchaseOrderStore } from '../stores/po-store.svelte';
  import PurchaseOrderForm from './PurchaseOrderForm.svelte';
  import GoodsReceiptModal from './GoodsReceiptModal.svelte';

  const store = usePurchaseOrderStore();
  let showForm = $state(false);
  let selectedPOForReceipt = $state<number | null>(null);

  onMount(() => {
    store.load(store.currentFilters);
  });

  let loadTimer: ReturnType<typeof setTimeout>;
  $effect(() => {
    clearTimeout(loadTimer);
    loadTimer = setTimeout(() => {
      store.page = 0;
      store.load(store.currentFilters);
    }, 300);
    return () => clearTimeout(loadTimer);
  });

  function handleCreate() {
    store.selectedPO = null;
    showForm = true;
  }

  function handleEdit(po: any) {
    store.selectedPO = po;
    showForm = true;
  }

  function handleView(po: any) {
    store.selectedPO = po;
    showForm = true;
  }

  function handleReceipt(poId: number) {
    selectedPOForReceipt = poId;
  }

  function getStatusBadge(status: string) {
    const colors: Record<string, string> = {
      draft: 'bg-gray-100 text-gray-700',
      confirmed: 'bg-blue-100 text-blue-700',
      partial_received: 'bg-yellow-100 text-yellow-700',
      fully_received: 'bg-green-100 text-green-700',
      cancelled: 'bg-red-100 text-red-700',
      waiting_approval: 'bg-orange-100 text-orange-700',
      rejected: 'bg-red-100 text-red-700',
    };
    return colors[status] || 'bg-gray-100 text-gray-700';
  }
</script>

<div class="p-6">
  <div class="flex justify-between items-center mb-6">
    <h1 class="text-2xl font-bold text-gray-900">Purchase Orders</h1>
    <button onclick={handleCreate} class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
      Create PO
    </button>
  </div>

  <div class="bg-white rounded-lg shadow">
    <div class="p-4 border-b flex gap-4">
      <input
        type="text"
        placeholder="Search PO number or supplier..."
        bind:value={store.searchQuery}
        class="px-3 py-2 border rounded-lg flex-1"
      />
      <select bind:value={store.statusFilter} class="px-3 py-2 border rounded-lg">
        <option value="">All Status</option>
        <option value="draft">Draft</option>
        <option value="confirmed">Confirmed</option>
        <option value="partial_received">Partial Received</option>
        <option value="fully_received">Fully Received</option>
        <option value="cancelled">Cancelled</option>
      </select>
      <input
        type="date"
        bind:value={store.startDate}
        class="px-3 py-2 border rounded-lg"
      />
      <input
        type="date"
        bind:value={store.endDate}
        class="px-3 py-2 border rounded-lg"
      />
    </div>

    {#if store.loading}
      <div class="p-8 text-center text-gray-500">Loading...</div>
    {:else if store.purchaseOrdersData.length === 0}
      <div class="p-8 text-center text-gray-500">No purchase orders yet</div>
    {:else}
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">PO Number</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Supplier</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Expected Date</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Grand Total</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created At</th>
            <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          {#each store.purchaseOrdersData as po}
            <tr>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{po.po_number}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{po.supplier_name || 'N/A'}</td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span class="px-2 py-1 text-xs font-medium rounded-full {getStatusBadge(po.status)}">
                  {po.status}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{po.expected_date || '-'}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{po.grand_total.toLocaleString('id-ID')}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{new Date(po.created_at).toLocaleDateString('id-ID')}</td>
              <td class="px-6 py-4 whitespace-nowrap text-right text-sm">
                <button onclick={() => handleView(po)} class="text-blue-600 hover:text-blue-800 mr-2">View</button>
                {#if po.status === 'draft'}
                  <button onclick={() => handleEdit(po)} class="text-green-600 hover:text-green-800 mr-2">Edit</button>
                {/if}
                {#if po.status === 'confirmed' || po.status === 'partial_received'}
                  <button onclick={() => handleReceipt(po.id)} class="text-purple-600 hover:text-purple-800 mr-2">Receive</button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>

  {#if showForm}
    <PurchaseOrderForm on:close={() => { showForm = false; store.selectedPO = null; }} />
  {/if}

  {#if selectedPOForReceipt}
    <GoodsReceiptModal poId={selectedPOForReceipt} on:close={() => { selectedPOForReceipt = null; }} />
  {/if}
</div>
