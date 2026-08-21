<script lang="ts">
  import { onMount } from 'svelte';
  import { usePurchaseOrderStore } from '../stores/po-store.svelte';
  import { useAuthStore } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
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
  const canCancel = $derived(userPermissions.includes('purchase_order.cancel'));

  let showForm = $state(false);
  let selectedPOForDetail = $state<number | null>(null);
  let showDetail = $state(false);
  let detailReloadKey = $state(0);
  let selectedPOForReceipt = $state<number | null>(null);
  let showReceiptModal = $state(false);

  let firstRun = true;
  let loadTimer: ReturnType<typeof setTimeout>;

  onMount(() => {
    store.load(store.currentFilters);
  });

  $effect(() => {
    // Track only filter inputs — pagination/sorting manage their own loads.
    // Reading currentFilters synchronously here would also capture `page`,
    // which would re-trigger this effect on every page change and reset to page 0.
    void store.searchQuery;
    void store.statusFilter;
    void store.supplierFilter;
    void store.startDate;
    void store.endDate;
    if (firstRun) {
      firstRun = false;
      return;
    }
    clearTimeout(loadTimer);
    loadTimer = setTimeout(() => {
      store.page = 0;
      store.load(store.currentFilters);
    }, 300);
    return () => clearTimeout(loadTimer);
  });

  $effect(() => {
    const unsubWS = store.subscribeToWS();
    return () => unsubWS();
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
    detailReloadKey += 1;
    showDetail = true;
  }

  async function handleConfirm(po: PurchaseOrder) {
    if (!confirm(`Confirm purchase order ${po.po_number}?`)) return;
    try {
      await store.confirm(po.id);
      toast.success(`${po.po_number} confirmed`);
      store.load(store.currentFilters);
    } catch (e: any) {
      toast.error(e.message || 'Failed to confirm');
    }
  }

  function handleReceipt(po: PurchaseOrder) {
    selectedPOForReceipt = po.id;
    showReceiptModal = true;
  }

  async function handleCancel(po: PurchaseOrder) {
    if (!confirm(`Cancel purchase order ${po.po_number}?`)) return;
    try {
      await store.cancel(po.id);
      toast.success(`${po.po_number} cancelled`);
      store.load(store.currentFilters);
    } catch (e: any) {
      toast.error(e.message || 'Failed to cancel purchase order');
    }
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
    store.page = 0;
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
      {canCancel}
      sortBy={store.sortBy}
      sortDir={store.sortDir}
      onsort={handleSort}
      onview={handleView}
      onedit={handleEdit}
      onconfirm={handleConfirm}
      onreceive={handleReceipt}
      oncancel={handleCancel}
    />

    {#if !store.loading && store.purchaseOrdersData.length > 0}
      <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
        <Pagination total={store.total} limit={store.pageSize} offset={store.offset} onPageChange={handlePageChange} />
      </div>
    {/if}
  </div>
</div>

  <PurchaseOrderForm bind:open={showForm} />

  <PurchaseOrderDetail
    bind:poId={selectedPOForDetail}
    bind:open={showDetail}
    reloadKey={detailReloadKey}
    {canEdit}
    {canConfirm}
    {canCancel}
    {canReceive}
    onedit={(po) => { showDetail = false; handleEdit(po); }}
    onconfirm={(po) => { showDetail = false; handleConfirm(po); }}
    oncancel={(po) => { showDetail = false; handleCancel(po); }}
    onreceive={(po) => { showDetail = false; handleReceipt(po); }}
  />

  <GoodsReceiptModal poId={selectedPOForReceipt} bind:open={showReceiptModal} onReceiptCreated={() => store.load(store.currentFilters)} />
