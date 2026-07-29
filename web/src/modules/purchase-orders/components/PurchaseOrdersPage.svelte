<script lang="ts">
  import { onMount } from 'svelte';
  import { usePurchaseOrderStore } from '../stores/po-store.svelte';
  import { useAuthStore } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { debounce } from '$shared/utils/debounce';
  import { Pagination } from '$shared/ui';
  import type { PurchaseOrder } from '../types';
  import PurchaseOrderForm from './PurchaseOrderForm.svelte';
  import GoodsReceiptModal from './GoodsReceiptModal.svelte';
  import PurchaseOrderDetail from './PurchaseOrderDetail.svelte';
  import PurchaseOrdersToolbar from './PurchaseOrdersToolbar.svelte';
  import PurchaseOrdersTable from './PurchaseOrdersTable.svelte';

  const store = usePurchaseOrderStore();
  const authStore = useAuthStore();

  const userPermissions = $derived(authStore.user?.permissions || []);
  const canCreate = $derived(userPermissions.includes('purchase_order.create'));
  const canView = $derived(userPermissions.includes('purchase_order.view'));
  const canEdit = $derived(userPermissions.includes('purchase_order.update'));
  const canConfirm = $derived(userPermissions.includes('purchase_order.confirm'));
  const canReceive = $derived(userPermissions.includes('purchase_order.receive'));

  let showForm = $state(false);
  let selectedPOForDetail = $state<number | null>(null);
  let showDetail = $state(false);
  let selectedPOForReceipt = $state<number | null>(null);
  let showReceiptModal = $state(false);

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
    selectedPOForDetail = po.id;
    showDetail = true;
  }

  async function handleConfirm(po: PurchaseOrder) {
    if (!confirm(`Confirm purchase order ${po.po_number}?`)) return;
    try {
      await store.confirm(po.id);
      toast.success(`PO ${po.po_number} confirmed`);
      store.load(store.currentFilters);
    } catch (e: any) {
      toast.error(e.message || 'Failed to confirm');
    }
  }

  function handleReceipt(po: PurchaseOrder) {
    selectedPOForReceipt = po.id;
    showReceiptModal = true;
  }

  function handlePageChange(newOffset: number, newLimit: number) {
    store.pageSize = newLimit;
    store.page = Math.floor(newOffset / newLimit);
    store.load(store.currentFilters);
  }

  function handleSort(column: string) {
    if (store.sortBy === column) {
      store.sortDir = store.sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      store.sortBy = column;
      store.sortDir = 'asc';
    }
    store.load(store.currentFilters);
  }
</script>

<div class="space-y-5">
  <PurchaseOrdersToolbar
    bind:searchQuery={store.searchQuery}
    bind:statusFilter={store.statusFilter}
    bind:startDate={store.startDate}
    bind:endDate={store.endDate}
    {canCreate}
    onsearch={() => { store.page = 0; store.load(store.currentFilters); }}
    onstatuschange={() => { store.page = 0; store.load(store.currentFilters); }}
    onstartdatechange={() => { store.page = 0; store.load(store.currentFilters); }}
    onenddatechange={() => { store.page = 0; store.load(store.currentFilters); }}
    oncreate={handleCreate}
  />

  <div class="card overflow-hidden">
    <PurchaseOrdersTable
      purchaseOrders={store.purchaseOrdersData}
      loading={store.loading}
      searchQuery={store.searchQuery}
      {canView}
      {canEdit}
      {canConfirm}
      {canReceive}
      sortBy={store.sortBy}
      sortDir={store.sortDir}
      onsort={handleSort}
      onview={handleView}
      onedit={handleEdit}
      onconfirm={handleConfirm}
      onreceive={handleReceipt}
    />

    {#if !store.loading && store.purchaseOrdersData.length > 0}
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
        <Pagination total={store.total} limit={store.pageSize} offset={store.offset} onPageChange={handlePageChange} />
      </div>
    {/if}
  </div>
</div>

  <PurchaseOrderForm bind:open={showForm} />

  <PurchaseOrderDetail bind:poId={selectedPOForDetail} bind:open={showDetail} />

  <GoodsReceiptModal poId={selectedPOForReceipt} bind:open={showReceiptModal} />
