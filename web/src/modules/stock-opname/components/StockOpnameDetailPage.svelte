<script lang="ts">
  import { onMount } from 'svelte';
  import { getPath, goto } from '$app/router';
  import { useStockOpnameStore } from '../stores/stock-opname-store.svelte';
  import { useAuthStore } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { Badge, Button, Card, Dropdown, EmptyState, Input, Modal, PageHeader, Pagination, SelectSearch, Skeleton } from '$shared/ui';
  import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime';
  import { ArrowLeft, CheckCircle2, ChevronDown, ClipboardCheck, RotateCcw, Send, XCircle } from 'lucide-svelte';
  import { labels, t } from '$shared/i18n';
  import { STOCK_OPNAME_STATUS_LABELS, STOCK_OPNAME_SCOPE_LABELS } from '../types';
  import type { StockOpnameSession } from '../types';

  const store = useStockOpnameStore();
  const authStore = useAuthStore();
  const userPermissions = $derived(authStore.user?.permissions || []);
  const canOpen = $derived(userPermissions.includes('stock_opname.create'));
  const canVerify = $derived(userPermissions.includes('stock_opname.verify'));
  const canPost = $derived(userPermissions.includes('stock_opname.post'));
  const canClose = $derived(userPermissions.includes('stock_opname.close'));
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
  let statusFilter = $state('');

  let showAssignModal = $state(false);
  let assignUserId = $state(0);
  let assignRole = $state('counter');
  let assigning = $state(false);

  let showCountModal = $state(false);
  let countTarget = $state<{ id: number; product_name: string } | null>(null);
  let countValue = $state('');
  let countRemarks = $state('');
  let counting = $state(false);

  let actionModal = $state<'open' | 'verify' | 'reject' | 'recount' | 'post' | null>(null);
  let actionComment = $state('');
  let postNotes = $state('');
  let actionInProgress = $state(false);

  function getOpenModal() { return actionModal === 'open'; }
  function setOpenModal(v: boolean) { actionModal = v ? 'open' : null; }

  function getVerifyModal() { return actionModal === 'verify' || actionModal === 'reject' || actionModal === 'recount'; }
  function setVerifyModal(v: boolean) { if (!v) actionModal = null; }

  function getPostModal() { return actionModal === 'post'; }
  function setPostModal(v: boolean) { actionModal = v ? 'post' : null; }

  let pageLimit = $state(20);
  let pageOffset = $state(0);

  onMount(async () => {
    await Promise.all([reload(), store.loadAssignableUsers()]);
  });

  const assignableUserOptions = $derived(
    store.assignableUsers.map(u => ({
      value: u.id,
      label: `${u.username} — ${u.role_name}`,
    }))
  );

  async function reload() {
    loading = true;
    try {
      await store.loadSession(sessionId);
      session = store.current;
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || labels.toastFailedLoadSession);
      goto('/stock-opnames');
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    const unsubWS = store.subscribeToWS();
    return () => unsubWS();
  });

  // Keep the page session in sync when the store refreshes the current
  // session after a WebSocket status event arrives.
  $effect(() => {
    if (store.current?.id === sessionId) {
      session = store.current;
    }
  });

  const filteredItems = $derived(
    session?.items?.filter(it =>
      (!statusFilter || it.status === statusFilter) &&
      (!searchQuery ||
        it.product_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        it.sku.toLowerCase().includes(searchQuery.toLowerCase()))
    ) ?? []
  );

  $effect(() => {
    searchQuery;
    statusFilter;
    pageOffset = 0;
  });

  const paginatedItems = $derived(filteredItems.slice(pageOffset, pageOffset + pageLimit));
  const totalFiltered = $derived(filteredItems.length);

  const itemStatusOptions = $derived([
    { value: 'pending', label: labels.statusPending },
    { value: 'counted', label: labels.statusCounted },
  ]);

  const itemStatusLabel = $derived(
    itemStatusOptions.find(s => s.value === statusFilter)?.label || labels.allStatus
  );

  const itemStatusItems = $derived([
    { label: labels.allStatus, checked: statusFilter === '', onclick: () => { statusFilter = ''; } },
    ...itemStatusOptions.map(opt => ({
      label: opt.label,
      checked: statusFilter === opt.value,
      onclick: () => { statusFilter = opt.value; },
    })),
  ]);

  function handlePageChange(newOffset: number, newLimit: number) {
    if (newLimit && newLimit !== pageLimit) pageLimit = newLimit;
    pageOffset = newOffset;
  }

  // @ownership-only — data-scope: hanya counter assignee dari session ini yang bisa masuk counting.
  const isCounter = $derived(
    (() => {
      const uid = authStore.user?.id;
      return !!uid && !!session?.assignments?.some(a => a.user_id === uid && a.role === 'counter');
    })()
  );
  const canEnterCount = $derived(canCount && isCounter && (session?.status === 'counting' || session?.status === 'needs_recount'));

  function statusBadge(status: string) {
    switch (status) {
      case 'draft': return 'muted';
      case 'open': return 'primary';
      case 'counting': return 'primary';
      case 'verification': return 'warning';
      case 'needs_recount': return 'warning';
      case 'approved': return 'success';
      case 'posted': return 'primary';
      case 'closed': return 'muted';
      case 'cancelled': return 'danger';
      default: return 'default';
    }
  }

  function scopeSummary(s: StockOpnameSession): string {
    const scopes = s.scopes?.length ? s.scopes : [{ scope_type: s.scope_type, scope_name: s.scope_name }];
    const label = (sc: any) => {
      const base = STOCK_OPNAME_SCOPE_LABELS[sc.scope_type as keyof typeof STOCK_OPNAME_SCOPE_LABELS] || sc.scope_type;
      if (sc.scope_type === 'manual') return base;
      return sc.scope_name ? `${base}: ${sc.scope_name}` : `${base} #${sc.scope_id}`;
    };
    return scopes.map(label).join(' + ');
  }

  async function handleAssign() {
    assigning = true;
    try {
      await store.assign(sessionId, { user_id: assignUserId, role: assignRole as any });
      toast.success(labels.toastCounterAssigned);
      showAssignModal = false;
      assignUserId = 0;
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || labels.toastFailedAssignCounter);
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
      toast.error(labels.toastInvalidQuantity);
      return;
    }
    counting = true;
    try {
      await store.saveCount(countTarget!.id, { physical_qty: qty, remarks: countRemarks || undefined });
      toast.success(labels.toastCountSaved);
      showCountModal = false;
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || labels.toastFailedSaveCount);
    } finally {
      counting = false;
    }
  }

  async function handleStart() {
    actionInProgress = true;
    try {
      await store.start(sessionId);
      toast.success(labels.toastCountingStarted);
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || labels.toastFailedStartCounting);
    } finally {
      actionInProgress = false;
    }
  }

  async function handleSubmit() {
    if (!confirm(t('confirmSubmitStockOpname', { number: session?.session_number || '' }))) return;
    actionInProgress = true;
    try {
      await store.submit(sessionId);
      toast.success(labels.toastSubmittedForVerification);
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || labels.toastFailedSubmit);
    } finally {
      actionInProgress = false;
    }
  }

  async function handleOpen() {
    actionInProgress = true;
    try {
      await store.open(sessionId, actionComment);
      toast.success(labels.toastSessionOpened);
      actionModal = null;
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || labels.toastFailedOpenSession);
    } finally {
      actionInProgress = false;
    }
  }

  async function handleVerify() {
    actionInProgress = true;
    try {
      await store.verify(sessionId, actionComment);
      toast.success(labels.toastStockOpnameVerified);
      actionModal = null;
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || labels.toastFailedVerify);
    } finally {
      actionInProgress = false;
    }
  }

  async function handleReject() {
    actionInProgress = true;
    try {
      await store.reject(sessionId, actionComment);
      toast.success(labels.toastRejectedRecountRequested);
      actionModal = null;
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || labels.toastFailedReject);
    } finally {
      actionInProgress = false;
    }
  }

  async function handleRecount() {
    actionInProgress = true;
    try {
      await store.recount(sessionId, actionComment);
      toast.success(labels.toastRecountRequested);
      actionModal = null;
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || labels.toastFailedRequestRecount);
    } finally {
      actionInProgress = false;
    }
  }

  async function handleResume() {
    actionInProgress = true;
    try {
      await store.resume(sessionId);
      toast.success(labels.toastCountingResumed);
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || labels.toastFailedResumeCounting);
    } finally {
      actionInProgress = false;
    }
  }

  async function handlePost() {
    actionInProgress = true;
    try {
      const adjustment = await store.post(sessionId, { comment: actionComment || undefined, notes: postNotes || undefined });
      toast.success(t('toastAdjustmentPosted', { number: adjustment.adjustment_number }));
      actionModal = null;
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || labels.toastFailedPostAdjustment);
    } finally {
      actionInProgress = false;
    }
  }

  async function handleClose() {
    if (!confirm(t('confirmCloseStockOpname', { number: session?.session_number || '' }))) return;
    actionInProgress = true;
    try {
      await store.close(sessionId);
      toast.success(labels.toastStockOpnameClosed);
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || labels.toastFailedClose);
    } finally {
      actionInProgress = false;
    }
  }

  async function handleCancel() {
    if (!confirm(t('confirmCancelStockOpname', { number: session?.session_number || '' }))) return;
    actionInProgress = true;
    try {
      await store.cancelSession(sessionId);
      toast.success(labels.toastStockOpnameCancelled);
      await reload();
    } catch (e: any) {
      toast.error(e?.response?.data?.error?.message || e.message || labels.toastFailedCancel);
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
      toast.error(labels.toastExportFailed);
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
    <PageHeader title={session.title || session.session_number} subtitle={scopeSummary(session)}>
      {#snippet actions()}
        <Button variant="ghost" onclick={() => goto('/stock-opnames')}>
          <ArrowLeft class="w-4 h-4" /> {labels.back}
        </Button>
        {#if canExport}
          <Button variant="secondary" onclick={doExport}>{labels.exportCSV}</Button>
        {/if}
        {#if canAssign && (session?.status === 'draft' || session?.status === 'open' || session?.status === 'counting' || session?.status === 'needs_recount')}
          <Button variant="secondary" onclick={() => (showAssignModal = true)}>{labels.assignCounter}</Button>
        {/if}
        {#if canOpen && session?.status === 'draft'}
          <Button onclick={() => { actionComment = ''; actionModal = 'open'; }} disabled={actionInProgress}>{labels.openSession}</Button>
        {/if}
        {#if canCount && isCounter && (session?.status === 'draft' || session?.status === 'open')}
          <Button onclick={handleStart} disabled={actionInProgress}>{labels.startCounting}</Button>
        {/if}
        {#if canSubmit && isCounter && session?.status === 'counting'}
          <Button onclick={handleSubmit} disabled={actionInProgress}>
            <Send class="w-4 h-4" /> {labels.submitForVerification}
          </Button>
        {/if}
        {#if canRecount && session?.status === 'verification'}
          <Button variant="secondary" onclick={() => { actionComment = ''; actionModal = 'recount'; }} disabled={actionInProgress}>
            <RotateCcw class="w-4 h-4" /> {labels.requestRecount}
          </Button>
        {/if}
        {#if canVerify && session?.status === 'verification'}
          <Button variant="success" onclick={() => { actionComment = ''; actionModal = 'verify'; }} disabled={actionInProgress}>
            <CheckCircle2 class="w-4 h-4" /> {labels.verify}
          </Button>
          <Button variant="danger" onclick={() => { actionComment = ''; actionModal = 'reject'; }} disabled={actionInProgress}>
            <XCircle class="w-4 h-4" /> {labels.reject}
          </Button>
        {/if}
        {#if canCount && isCounter && session?.status === 'needs_recount'}
          <Button onclick={handleResume} disabled={actionInProgress}>
            <ClipboardCheck class="w-4 h-4" /> {labels.resumeCounting}
          </Button>
        {/if}
        {#if canPost && session?.status === 'approved'}
          <Button variant="success" onclick={() => { actionComment = ''; postNotes = ''; actionModal = 'post'; }} disabled={actionInProgress}>
            {labels.postAdjustment}
          </Button>
        {/if}
        {#if canClose && session?.status === 'posted'}
          <Button onclick={handleClose} disabled={actionInProgress}>{labels.closeSession}</Button>
        {/if}
        {#if canCancel && (session?.status === 'draft' || session?.status === 'open' || session?.status === 'counting' || session?.status === 'needs_recount')}
          <Button variant="danger" onclick={handleCancel} disabled={actionInProgress}>{labels.cancel}</Button>
        {/if}
      {/snippet}
    </PageHeader>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <Card class="p-4">
        <div class="text-sm text-text-muted">{labels.status}</div>
        <div class="mt-1"><Badge variant={statusBadge(session.status)}>{STOCK_OPNAME_STATUS_LABELS[session.status] || session.status}</Badge></div>
      </Card>
      <Card class="p-4">
        <div class="text-sm text-text-muted">{labels.blindCount}</div>
        <div class="mt-1 font-semibold text-text-primary">{session.blind_count ? labels.enabled : labels.disabled}</div>
      </Card>
      <Card class="p-4">
        <div class="text-sm text-text-muted">{labels.totalDifference}</div>
        <div class="mt-1 font-semibold tabular-nums {session.total_difference === 0 ? 'text-text-primary' : session.total_difference > 0 ? 'text-success-light' : 'text-danger-light'}">
          {session.total_difference || '—'}
        </div>
      </Card>
      <Card class="p-4">
        <div class="text-sm text-text-muted">{labels.totalAdjustment}</div>
        <div class="mt-1 font-semibold tabular-nums text-text-primary">{session.total_adjustment || '—'}</div>
      </Card>
    </div>

    {#if session.notes}
      <Card class="p-4">
        <div class="text-sm text-text-muted">{labels.notes}</div>
        <div class="mt-1 text-sm text-text-primary">{session.notes}</div>
      </Card>
    {/if}

    {#if session.assignments && session.assignments.length > 0}
      <Card class="p-4">
        <h3 class="text-sm font-semibold text-text-primary mb-2">{labels.assignments}</h3>
        <div class="flex flex-wrap gap-2">
          <!-- @display-only — badge tugas assignment (counter/verifier), bukan authz. -->
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
        <h3 class="text-sm font-semibold text-text-primary">{labels.items}</h3>
        <div class="flex flex-wrap items-center gap-2">
          <Dropdown placement="bottom-start" items={itemStatusItems}>
            {#snippet trigger({ toggle })}
              <button
                type="button"
                class="flex items-center gap-2 px-3 h-10 rounded-xl border transition-all duration-200 text-[13px] font-medium whitespace-nowrap {statusFilter !== '' ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
                onclick={toggle}
              >
                <span>{itemStatusLabel}</span>
                <ChevronDown size={14} class="text-text-muted shrink-0" />
              </button>
            {/snippet}
          </Dropdown>
          <div class="w-64 max-w-full">
            <Input type="text" placeholder={labels.searchProductOrSku} bind:value={searchQuery} />
          </div>
        </div>
      </div>

      {#if filteredItems.length === 0}
        <EmptyState title={labels.noItems} subtitle={labels.noProductsMatchFilter} />
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-left whitespace-nowrap">
            <thead class="bg-surface-subtle border-b border-border">
              <tr class="text-[11px] font-semibold text-text-muted uppercase tracking-wider h-10">
                <th class="px-3">{labels.product}</th>
                <th class="px-3">{labels.sku}</th>
                <th class="px-3 text-right">{labels.opening}</th>
                <th class="px-3 text-right">{labels.physical}</th>
                <th class="px-3 text-right">{labels.diff}</th>
                <th class="px-3">{labels.status}</th>
                {#if canEnterCount}
                  <th class="px-3">{labels.action}</th>
                {/if}
              </tr>
            </thead>
            <tbody class="divide-y divide-border/60">
              {#each paginatedItems as item}
                <tr class="hover:bg-surface-subtle">
                  <td class="px-3 py-2.5 font-medium text-text-primary">{item.product_name}</td>
                  <td class="px-3 py-2.5 text-text-secondary">{item.sku}</td>
                  <td class="px-3 py-2.5 text-right text-text-secondary">{item.opening_qty} {item.uom_name}</td>
                  <td class="px-3 py-2.5 text-right text-text-primary font-semibold">{item.physical_qty}</td>
                  <td class="px-3 py-2.5 text-right {item.difference_qty === 0 ? 'text-text-secondary' : item.difference_qty > 0 ? 'text-success-light' : 'text-danger-light'}">
                    {item.difference_qty === 0 ? '—' : item.difference_qty}
                  </td>
                  <td class="px-3 py-2.5">
                    <Badge variant={item.status === 'counted' ? 'success' : 'muted'}>{item.status}</Badge>
                  </td>
                  {#if canEnterCount}
                    <td class="px-3 py-2.5">
                      <Button variant="ghost" size="xs" onclick={() => openCount(item.id, item.product_name)}>
                        {item.status === 'counted' ? labels.recount : labels.count}
                      </Button>
                    </td>
                  {/if}
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
        {#if totalFiltered > 0}
          <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
            <Pagination total={totalFiltered} limit={pageLimit} offset={pageOffset} onPageChange={handlePageChange} />
          </div>
        {/if}
      {/if}
    </Card>
  </div>
{:else}
  <EmptyState title={labels.sessionNotFound} subtitle={labels.stockOpnameSessionCouldNotBeLoaded} />
{/if}

<Modal bind:open={showAssignModal} title={labels.assignCounter} size="sm">
  {#snippet children()}
    <div class="space-y-4">
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.user}</span>
        <SelectSearch
          bind:value={assignUserId}
          options={assignableUserOptions}
          placeholder={store.assignableLoading ? labels.loadingUsers : labels.selectUser}
          searchPlaceholder={labels.searchByUsernameOrEmail}
          notFoundText={labels.noMatchingUsers}
        />
      </label>
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.role}</span>
        <Input tag="select" bind:value={assignRole}>
          {#snippet children()}
            <option value="counter">{labels.counter}</option>
            <option value="supervisor">{labels.supervisor}</option>
          {/snippet}
        </Input>
      </label>
    </div>
  {/snippet}
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showAssignModal = false)}>{labels.cancel}</Button>
      <Button onclick={handleAssign} disabled={assigning || !assignUserId}>{labels.assign}</Button>
    </div>
  {/snippet}
</Modal>

<Modal bind:open={showCountModal} title={countTarget ? t('countWithProduct', { name: countTarget.product_name }) : ''} size="sm">
  {#snippet children()}
    <div class="space-y-4">
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.physicalQuantity}</span>
        <Input type="number" bind:value={countValue} min={0} step="any" placeholder="0" autofocus />
      </label>
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.remarks}</span>
        <Input tag="textarea" bind:value={countRemarks} placeholder={labels.optionalNotes} />
      </label>
    </div>
  {/snippet}
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (showCountModal = false)}>{labels.cancel}</Button>
      <Button onclick={handleSaveCount} disabled={counting || countValue === ''}>{labels.saveCount}</Button>
    </div>
  {/snippet}
</Modal>

{#if actionModal === 'open'}
  <Modal bind:open={getOpenModal, setOpenModal} title={labels.openSession} size="sm">
    {#snippet children()}
      <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
        <span>{labels.commentRequired}</span>
        <Input tag="textarea" bind:value={actionComment} placeholder={labels.whyIsThisSessionBeingOpened} />
      </label>
    {/snippet}
    {#snippet footer()}
      <div class="flex justify-end gap-3 w-full">
        <Button variant="secondary" onclick={() => (actionModal = null)}>{labels.cancel}</Button>
        <Button onclick={handleOpen} disabled={actionInProgress || !actionComment.trim()}>{labels.open}</Button>
      </div>
    {/snippet}
  </Modal>
{:else if actionModal === 'verify' || actionModal === 'reject' || actionModal === 'recount'}
  <Modal bind:open={getVerifyModal, setVerifyModal} title={actionModal === 'verify' ? labels.verifyStockOpname : actionModal === 'reject' ? labels.rejectStockOpname : labels.requestRecount} size="sm">
    {#snippet children()}
      <div class="space-y-2">
        <p class="text-sm text-text-secondary">
          {#if actionModal === 'verify'}
            {labels.verifyModalHint}
          {:else if actionModal === 'reject'}
            {labels.rejectModalHint}
          {:else}
            {labels.recountModalHint}
          {/if}
        </p>
        <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
          <span>{labels.commentRequired}</span>
          <Input tag="textarea" bind:value={actionComment} placeholder={labels.verificationNotes} />
        </label>
      </div>
    {/snippet}
    {#snippet footer()}
      <div class="flex justify-end gap-3 w-full">
        <Button variant="secondary" onclick={() => (actionModal = null)}>{labels.cancel}</Button>
        <Button variant={actionModal === 'verify' ? 'success' : 'danger'} onclick={actionModal === 'verify' ? handleVerify : actionModal === 'reject' ? handleReject : handleRecount} disabled={actionInProgress || !actionComment.trim()}>
          {actionModal === 'verify' ? labels.verify : actionModal === 'reject' ? labels.reject : labels.requestRecount}
        </Button>
      </div>
    {/snippet}
  </Modal>
{:else if actionModal === 'post'}
  <Modal bind:open={getPostModal, setPostModal} title={labels.postAdjustment} size="sm">
    {#snippet children()}
      <div class="space-y-2">
        <p class="text-sm text-text-secondary">
          {labels.postModalHint}
        </p>
        <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
          <span>{labels.notes}</span>
          <Input tag="textarea" bind:value={postNotes} placeholder={labels.optionalAdjustmentNotes} />
        </label>
        <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
          <span>{labels.optionalComment}</span>
          <Input tag="textarea" bind:value={actionComment} placeholder={labels.optionalComment} />
        </label>
      </div>
    {/snippet}
    {#snippet footer()}
      <div class="flex justify-end gap-3 w-full">
        <Button variant="secondary" onclick={() => (actionModal = null)}>{labels.cancel}</Button>
        <Button variant="success" onclick={handlePost} disabled={actionInProgress}>{labels.postAdjustment}</Button>
      </div>
    {/snippet}
  </Modal>
{/if}
