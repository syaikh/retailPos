<script lang="ts">
  import { onMount } from 'svelte';
  import { getPath, goto } from '$app/router';
  import { useStockOpnameStore } from '../stores/stock-opname-store.svelte';
  import { useAuthStore } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { Badge, Button, Card, EmptyState, Input, Modal, PageHeader, Skeleton } from '$shared/ui';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
  import { ArrowLeft, CheckCircle2, ClipboardCheck, RotateCcw, Send, XCircle } from 'lucide-svelte';
  import { STOCK_OPNAME_STATUS_LABELS } from '../types';
  import type { StockOpnameSession } from '../types';

  const store = useStockOpnameStore();
  const authStore = useAuthStore();
  const userPermissions = $derived(authStore.user?.permissions || []);
  const canApprove = $derived(userPermissions.includes('stock_opname.approve'));
  const canReject = $derived(userPermissions.includes('stock_opname.reject'));
  const canRecount = $derived(userPermissions.includes('stock_opname.recount'));
  const canCancel = $derived(userPermissions.includes('stock_opname.cancel'));
  const canCount = $derived(userPermissions.includes('stock_opname.count'));
  const canSubmit = $derived(userPermissions.includes('stock_opname.submit'));
  const canAssign = $derived(userPermissions.includes('stock_opname.assign'));
  const canExport = $derived(userPermissions.includes('stock_opname.export'));

  const sessionId = Number((getPath().split('/').filter(Boolean).pop()) || '0');
  let session = $state<StockOpnameSession | null>(null);
  let loading = $state(true);
  let searchQuery = $state('');

  let showAssignModal = $state(false);
  let assignUserId = $state('');
  let assignRole = $state('counter');
  let assigning = $state(false);

  let showCountModal = $state(false);
  let countTarget = $state<{ id: number; product_name: string } | null>(null);
  let countValue = $state('');
  let countRemarks = $state('');
  let counting = $state(false);

  let showApproveModal = $state(false);
  let approveComment = $state('');
  let actionInProgress = $state(false);

  onMount(async () => {
    await reload();
  });

  async function reload() {
    loading = true;
    try {
      await store.loadSession(sessionId);
      session = store.current;
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || 'Failed to load session');
      goto('/stock-opnames');
    } finally {
      loading = false;
    }
  }

  const filteredItems = $derived(
    session?.items?.filter(it =>
      !searchQuery ||
      it.product_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      it.sku.toLowerCase().includes(searchQuery.toLowerCase())
    ) ?? []
  );

  const isCounter = $derived(
    !!authStore.user?.id && !!session?.assignments?.some(a => a.user_id === authStore.user.id && a.role === 'counter')
  );
  const canEnterCount = $derived(canCount && isCounter && (session?.status === 'counting' || session?.status === 'needs_recount'));

  function statusBadge(status: string) {
    switch (status) {
      case 'draft': return 'muted';
      case 'counting': return 'primary';
      case 'pending_approval': return 'warning';
      case 'needs_recount': return 'warning';
      case 'approved': return 'success';
      case 'cancelled': return 'danger';
      default: return 'default';
    }
  }

  async function handleAssign() {
    assigning = true;
    try {
      await store.assign(sessionId, { user_id: Number(assignUserId), role: assignRole as any });
      toast.success('Counter assigned');
      showAssignModal = false;
      assignUserId = '';
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || 'Failed to assign counter');
    } finally {
      assigning = false;
    }
  }

  function openCount(itemId: number, productName: string) {
    countTarget = { id: itemId, product_name: productName };
    countValue = '';
    countRemarks = '';
    showCountModal = true;
  }

  async function handleSaveCount() {
    const qty = parseFloat(countValue);
    if (isNaN(qty) || qty < 0) {
      toast.error('Please enter a valid quantity');
      return;
    }
    counting = true;
    try {
      await store.saveCount(countTarget!.id, { physical_qty: qty, remarks: countRemarks || undefined });
      toast.success('Count saved');
      showCountModal = false;
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || 'Failed to save count');
    } finally {
      counting = false;
    }
  }

  async function handleStart() {
    actionInProgress = true;
    try {
      await store.start(sessionId);
      toast.success('Counting started');
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || 'Failed to start counting');
    } finally {
      actionInProgress = false;
    }
  }

  async function handleSubmit() {
    if (!confirm(`Submit stock opname ${session?.session_number} for approval?`)) return;
    actionInProgress = true;
    try {
      await store.submit(sessionId);
      toast.success('Submitted for approval');
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || 'Failed to submit');
    } finally {
      actionInProgress = false;
    }
  }

  async function handleApprove() {
    actionInProgress = true;
    try {
      await store.approve(sessionId, approveComment);
      toast.success('Stock opname approved — inventory adjusted');
      showApproveModal = false;
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || 'Failed to approve');
    } finally {
      actionInProgress = false;
    }
  }

  async function handleReject() {
    if (!confirm(`Reject stock opname ${session?.session_number} and request recount?`)) return;
    actionInProgress = true;
    try {
      await store.reject(sessionId, approveComment);
      toast.success('Rejected — recount requested');
      showApproveModal = false;
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || 'Failed to reject');
    } finally {
      actionInProgress = false;
    }
  }

  async function handleRecount() {
    if (!confirm(`Request recount for stock opname ${session?.session_number}?`)) return;
    actionInProgress = true;
    try {
      await store.recount(sessionId, 'Recount requested');
      toast.success('Recount requested');
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || 'Failed to request recount');
    } finally {
      actionInProgress = false;
    }
  }

  async function handleResume() {
    actionInProgress = true;
    try {
      await store.resume(sessionId);
      toast.success('Counting resumed');
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || 'Failed to resume');
    } finally {
      actionInProgress = false;
    }
  }

  async function handleCancel() {
    if (!confirm(`Cancel stock opname ${session?.session_number}? This cannot be undone.`)) return;
    actionInProgress = true;
    try {
      await store.cancelSession(sessionId);
      toast.success('Stock opname cancelled');
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || 'Failed to cancel');
    } finally {
      actionInProgress = false;
    }
  }

  async function doExport() {
    try {
      const blob = await store.exportCSV(sessionId);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `stock-opname-${session?.session_number}.csv`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch {
      toast.error('Failed to export');
    }
  }
</script>

{#if loading && !session}
  <div class="space-y-4">
    <Skeleton class="h-12 w-64" />
    <Skeleton class="h-64 w-full" />
  </div>
{:else if session}
  <div class="space-y-5">
    <PageHeader title={session.session_number} subtitle={`${session.scope_type} scope #${session.scope_id}`}>
      {#snippet actions()}
        <Button variant="ghost" onclick={() => goto('/stock-opnames')}>
          <ArrowLeft class="w-4 h-4" /> Back
        </Button>
        {#if canExport}
          <Button variant="secondary" onclick={doExport}>Export CSV</Button>
        {/if}
        {#if canAssign && (session.status === 'draft' || session.status === 'counting' || session.status === 'needs_recount')}
          <Button variant="secondary" onclick={() => (showAssignModal = true)}>Assign Counter</Button>
        {/if}
        {#if canCount && isCounter && session.status === 'draft'}
          <Button onclick={handleStart} disabled={actionInProgress}>Start Counting</Button>
        {/if}
        {#if canSubmit && isCounter && session.status === 'counting'}
          <Button onclick={handleSubmit} disabled={actionInProgress}>
            <Send class="w-4 h-4" /> Submit for Approval
          </Button>
        {/if}
        {#if canRecount && session.status === 'pending_approval'}
          <Button variant="secondary" onclick={handleRecount} disabled={actionInProgress}>
            <RotateCcw class="w-4 h-4" /> Request Recount
          </Button>
        {/if}
        {#if canApprove && session.status === 'pending_approval'}
          <Button variant="success" onclick={() => { approveComment = ''; showApproveModal = true; }}>
            <CheckCircle2 class="w-4 h-4" /> Approve
          </Button>
        {/if}
        {#if canReject && session.status === 'pending_approval'}
          <Button variant="danger" onclick={() => { approveComment = ''; showApproveModal = true; }}>
            <XCircle class="w-4 h-4" /> Reject
          </Button>
        {/if}
        {#if canCount && isCounter && session.status === 'needs_recount'}
          <Button onclick={handleResume} disabled={actionInProgress}>
            <ClipboardCheck class="w-4 h-4" /> Resume Counting
          </Button>
        {/if}
        {#if canCancel && (session.status === 'draft' || session.status === 'counting' || session.status === 'needs_recount')}
          <Button variant="danger" onclick={handleCancel} disabled={actionInProgress}>Cancel</Button>
        {/if}
      {/snippet}
    </PageHeader>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <Card class="p-4">
        <div class="text-sm text-text-muted">Status</div>
        <div class="mt-1"><Badge variant={statusBadge(session.status)}>{STOCK_OPNAME_STATUS_LABELS[session.status] || session.status}</Badge></div>
      </Card>
      <Card class="p-4">
        <div class="text-sm text-text-muted">Blind Count</div>
        <div class="mt-1 font-semibold text-text-primary">{session.blind_count ? 'Enabled' : 'Disabled'}</div>
      </Card>
      <Card class="p-4">
        <div class="text-sm text-text-muted">Total Items</div>
        <div class="mt-1 font-semibold text-text-primary">{store.currentSummary?.total_items ?? session.items?.length ?? 0}</div>
      </Card>
      <Card class="p-4">
        <div class="text-sm text-text-muted">Counted</div>
        <div class="mt-1 font-semibold text-text-primary">
          {store.currentSummary?.counted_items ?? 0}
          <span class="text-text-muted font-normal">/ {store.currentSummary?.total_items ?? 0}</span>
        </div>
      </Card>
    </div>

    {#if session.assignments && session.assignments.length > 0}
      <Card class="p-4">
        <h3 class="text-sm font-semibold text-text-primary mb-2">Assignments</h3>
        <div class="flex flex-wrap gap-2">
          {#each session.assignments as a}
            <Badge variant={a.role === 'counter' ? 'primary' : 'muted'}>
              {a.username} ({a.role})
            </Badge>
          {/each}
        </div>
      </Card>
    {/if}

    <Card class="p-4">
      <div class="flex flex-wrap items-center justify-between gap-3 mb-3">
        <h3 class="text-sm font-semibold text-text-primary">Items</h3>
        <div class="w-64 max-w-full">
          <Input type="text" placeholder="Search product / SKU..." bind:value={searchQuery} />
        </div>
      </div>

      {#if filteredItems.length === 0}
        <EmptyState title="No items" description="No products match the current filter." />
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-left whitespace-nowrap">
            <thead class="bg-surface-subtle border-b border-border">
              <tr class="text-[11px] font-semibold text-text-muted uppercase tracking-wider h-10">
                <th class="px-3">Product</th>
                <th class="px-3">SKU</th>
                <th class="px-3">Opening</th>
                <th class="px-3">Physical</th>
                <th class="px-3">Diff</th>
                <th class="px-3">Status</th>
                {#if canEnterCount}
                  <th class="px-3">Action</th>
                {/if}
              </tr>
            </thead>
            <tbody class="divide-y divide-border/60">
              {#each filteredItems as item}
                <tr class="hover:bg-surface-subtle">
                  <td class="px-3 py-2.5 font-medium text-text-primary">{item.product_name}</td>
                  <td class="px-3 py-2.5 text-text-secondary">{item.sku}</td>
                  <td class="px-3 py-2.5 text-text-secondary">{item.opening_qty} {item.uom_name}</td>
                  <td class="px-3 py-2.5 text-text-primary font-semibold">{item.physical_qty}</td>
                  <td class="px-3 py-2.5 {item.difference_qty === 0 ? 'text-text-secondary' : item.difference_qty > 0 ? 'text-success-light' : 'text-danger-light'}">
                    {item.difference_qty === 0 ? '—' : item.difference_qty}
                  </td>
                  <td class="px-3 py-2.5">
                    <Badge variant={item.status === 'counted' ? 'success' : 'muted'}>{item.status}</Badge>
                  </td>
                  {#if canEnterCount}
                    <td class="px-3 py-2.5">
                      <Button variant="ghost" size="xs" onclick={() => openCount(item.id, item.product_name)}>
                        {item.status === 'counted' ? 'Recount' : 'Count'}
                      </Button>
                    </td>
                  {/if}
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </Card>
  </div>
{:else}
  <EmptyState title="Session not found" description="This stock opname session could not be loaded." />
{/if}

<Modal bind:open={showAssignModal} title="Assign Counter" size="sm">
  {#snippet children()}
    <div class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-text-secondary mb-1">User ID</label>
        <Input type="number" bind:value={assignUserId} min={1} placeholder="e.g. 3" />
      </div>
      <div>
        <label class="block text-sm font-medium text-text-secondary mb-1">Role</label>
        <Input tag="select" bind:value={assignRole}>
          {#snippet children()}
            <option value="counter">Counter</option>
            <option value="supervisor">Supervisor</option>
          {/snippet}
        </Input>
      </div>
    </div>
  {/snippet}
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showAssignModal = false)}>Cancel</Button>
      <Button onclick={handleAssign} disabled={assigning || !assignUserId}>Assign</Button>
    </div>
  {/snippet}
</Modal>

<Modal bind:open={showCountModal} title={countTarget ? `Count: ${countTarget.product_name}` : ''} size="sm">
  {#snippet children()}
    <div class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-text-secondary mb-1">Physical Quantity</label>
        <Input type="number" bind:value={countValue} min={0} step="any" placeholder="0" autofocus />
      </div>
      <div>
        <label class="block text-sm font-medium text-text-secondary mb-1">Remarks</label>
        <Input tag="textarea" bind:value={countRemarks} placeholder="Optional notes" />
      </div>
    </div>
  {/snippet}
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showCountModal = false)}>Cancel</Button>
      <Button onclick={handleSaveCount} disabled={counting || countValue === ''}>Save Count</Button>
    </div>
  {/snippet}
</Modal>

<Modal bind:open={showApproveModal} title="Approval Action" size="sm">
  {#snippet children()}
    <div>
      <label class="block text-sm font-medium text-text-secondary mb-1">Comment (required)</label>
      <Input tag="textarea" bind:value={approveComment} placeholder="Approval / rejection notes" />
    </div>
  {/snippet}
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showApproveModal = false)}>Cancel</Button>
      <Button variant="danger" onclick={handleReject} disabled={actionInProgress || !approveComment}>Reject</Button>
      <Button variant="success" onclick={handleApprove} disabled={actionInProgress || !approveComment}>Approve</Button>
    </div>
  {/snippet}
</Modal>
