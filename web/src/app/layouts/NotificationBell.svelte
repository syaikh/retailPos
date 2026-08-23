<script lang="ts">
  import { Button } from '$shared/ui';
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { Bell } from 'lucide-svelte';
  import { useWebSocket } from '$shared/api/websocket';
  import { goto } from '$app/router';
  import { toast } from '$shared/stores/toast.svelte';
  import {
    notifications,
    formatRelativeTime,
    getNotificationIcon,
    canReceiveStockOpnameNotifications,
    type Notification,
  } from '$shared/stores/notifications.svelte';
  import { useAuthStore } from '$modules/auth';
  import { labels, t } from '$shared/i18n';
  import { routePermissions } from '$app/config/permissions';
  import { useRBAC } from '$shared/composables/useRBAC.svelte';

  const ws = useWebSocket();
  const authStore = useAuthStore();
  const rbac = useRBAC();

  const canSeeStockOpname = $derived(canReceiveStockOpnameNotifications(authStore.user?.permissions));

  let open = $state(false);
  let container = $state<HTMLElement>();

  // Workaround: subscribe to store manually to use value in template
  let items = $state<Notification[]>([]);
  let unread = $state(0);

  notifications.subscribe(n => items = n);
  notifications.unreadCount.subscribe(n => unread = n);

  function handleClick() {
    open = !open;
  }

  // Close on outside click / Escape — only when open
  $effect(() => {
    if (!open) return;

    function handleWindowClick(e: MouseEvent) {
      if (container && !container.contains(e.target as Node)) {
        open = false;
      }
    }
    function handleKeydown(e: KeyboardEvent) {
      if (e.key === 'Escape') open = false;
    }

    // Delay so the click that opened the dropdown is not captured
    const raf = requestAnimationFrame(() => {
      window.addEventListener('click', handleWindowClick);
      window.addEventListener('keydown', handleKeydown);
    });

    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener('click', handleWindowClick);
      window.removeEventListener('keydown', handleKeydown);
    };
  });

  function handleNotificationClick(n: Notification) {
    if (!n.read) notifications.markAsRead(n.id);
    open = false;
    if (!n.navigateTo) return;
    const targetPath = n.navigateTo.split('?')[0];
    // Detail deep-links (e.g. /stock-opnames/<id>) have no exact entry, so fall
    // back to the base resource path to gate them with the same permission.
    const requiredPerms = routePermissions[targetPath] ?? routePermissions[basePath(targetPath)];
    if (requiredPerms && !rbac.canAny(requiredPerms)) return;
    goto(n.navigateTo);
  }

  function basePath(path: string): string {
    const parts = path.split('/').filter(Boolean);
    parts.pop();
    return '/' + parts.join('/');
  }

  function handleMarkAllRead(e: Event) {
    e.stopPropagation();
    notifications.markAllRead();
  }

  function handleDropdownKeydown(e: KeyboardEvent) {
    if (e.key !== 'Tab') return;
    const dropdown = container?.querySelector('[role="menu"]');
    if (!dropdown) return;
    const focusable = dropdown.querySelectorAll<HTMLElement>('button, [tabindex]:not([tabindex="-1"])');
    if (focusable.length === 0) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  }

  // Register WebSocket listeners once on mount
  onMount(() => {
    const unsubs = [
      ws.on('low_stock_alert', (data: any) => {
        notifications.push({
          type: 'low_stock',
          title: labels.lowStockAlert,
          description: t('lowStockAlertDesc', { name: data.name, stock: data.stock }),
          navigateTo: '/inventory/products?low_stock=true',
        });
        toast.error(t('lowStockAlertToast', { name: data.name, stock: data.stock }));
      }),
      ws.on('sale_created', (data: any) => {
        notifications.push({
          type: 'sale_created',
          title: labels.newTransaction,
          description: t('newTransactionDesc', { invoice: data.invoice, amount: (data.total || 0).toLocaleString('id-ID') }),
          navigateTo: data.id ? `/transactions?txn=${data.id}` : '/transactions',
        });
      }),
      ws.on('stock_update', (data: any) => {
        const status = data.low_stock ? `⚠️ ${labels.stockUpdateStatusLow}` : t('stockUpdateUnits', { count: data.stock });
        notifications.push({
          type: 'stock_update',
          title: labels.stockUpdated,
          description: t('stockUpdatedDesc', { sku: data.sku, status }),
          navigateTo: `/inventory/products?product_id=${data.id}`,
        });
      }),
      ws.on('product_updated', (data: any) => {
        notifications.push({
          type: 'product_updated',
          title: labels.productUpdated,
          description: t('productUpdatedDesc', { sku: data.sku, price: (data.price || 0).toLocaleString('id-ID') }),
          navigateTo: `/inventory/products?product_id=${data.id}`,
        });
      }),
      ws.on('po_received', (data: any) => {
        notifications.push({
          type: 'po_received',
          title: labels.poReceived,
          description: t('poReceivedDesc', { po_number: data.po_number, gr_number: data.gr_number }),
          navigateTo: `/purchase-orders?po_id=${data.po_id}`,
        });
      }),
      ws.on('so_created', (data: any) => {
        if (!canSeeStockOpname) return;
        notifications.push({
          type: 'so_created',
          title: labels.soCreatedTitle,
          description: t('soSessionCreatedDesc', { number: data.session_number }),
          navigateTo: `/stock-opnames/${data.session_id}`,
        });
      }),
      ws.on('so_submitted', (data: any) => {
        if (!canSeeStockOpname) return;
        notifications.push({
          type: 'so_submitted',
          title: labels.soSubmittedTitle,
          description: t('soSessionAwaitingDesc', { number: data.session_number }),
          navigateTo: `/stock-opnames/${data.session_id}`,
        });
      }),
      ws.on('so_approved', (data: any) => {
        if (!canSeeStockOpname) return;
        notifications.push({
          type: 'so_approved',
          title: labels.soApprovedTitle,
          description: t('soSessionApprovedDesc', { number: data.session_number }),
          navigateTo: `/stock-opnames/${data.session_id}`,
        });
      }),
      ws.on('so_rejected', (data: any) => {
        if (!canSeeStockOpname) return;
        notifications.push({
          type: 'so_rejected',
          title: labels.soRejectedTitle,
          description: t('soSessionRejectedDesc', { number: data.session_number }),
          navigateTo: `/stock-opnames/${data.session_id}`,
        });
      }),
      ws.on('so_needs_recount', (data: any) => {
        if (!canSeeStockOpname) return;
        notifications.push({
          type: 'so_needs_recount',
          title: labels.soNeedsRecountTitle,
          description: t('soSessionNeedsRecountDesc', { number: data.session_number }),
          navigateTo: `/stock-opnames/${data.session_id}`,
        });
      }),
      ws.on('so_cancelled', (data: any) => {
        if (!canSeeStockOpname) return;
        notifications.push({
          type: 'so_cancelled',
          title: labels.soCancelledTitle,
          description: t('soSessionCancelledDesc', { number: data.session_number }),
          navigateTo: `/stock-opnames/${data.session_id}`,
        });
      }),
    ];
    return () => unsubs.forEach(fn => fn());
  });
</script>

<div bind:this={container} class="relative">
  <Button variant="ghost" size="icon" class="relative text-text-muted hover:text-text-primary" onclick={handleClick} aria-label={labels.notifications}>
    <Bell size={18} />
    {#if unread > 0}
      <span class="absolute -top-0.5 -right-0.5 min-w-[16px] h-4 flex items-center justify-center bg-danger text-white text-[10px] font-bold rounded-full px-1 leading-none shadow-glow-danger">
        {unread > 99 ? '99+' : unread}
      </span>
    {/if}
  </Button>

  {#if open}
    <div class="absolute right-0 top-full mt-2 z-50" onclick={(e) => e.stopPropagation()} role="none" onkeydown={handleDropdownKeydown} transition:fly={{ y: -8, duration: 200 }}>
      <div class="card-glass p-2 w-80 max-h-96 flex flex-col">
        <!-- Header -->
        <div class="flex items-center justify-between px-2 py-1.5 border-b border-border/50">
          <span class="text-xs font-semibold text-text-primary uppercase tracking-wide">
            {labels.notifications} {unread > 0 ? `(${unread})` : ''}
          </span>
          {#if items.length > 0 && unread > 0}
<button type="button" 
              class="text-[10px] font-medium text-primary-light hover:text-primary transition-colors"
              onclick={handleMarkAllRead}
            >
              {labels.markAllRead}
            </button>
          {/if}
        </div>

        <!-- Body -->
        <div class="overflow-y-auto flex-1 max-h-80" role="menu">
          {#if items.length === 0}
            <div class="flex flex-col items-center justify-center py-8 text-text-muted">
              <Bell size={24} class="opacity-30 mb-2" />
              <span class="text-xs">{labels.noNotificationsYet}</span>
            </div>
          {:else}
            {#each items as n (n.id)}
  <button type="button" 
                class="w-full text-left flex items-start gap-2.5 px-2 py-2.5 rounded-lg transition-colors hover:bg-surface-hover/50 {n.read ? 'opacity-60' : ''}"
                onclick={() => handleNotificationClick(n)}
                role="menuitem"
              >
                <span class="text-base leading-none mt-0.5 shrink-0">{getNotificationIcon(n.type)}</span>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center justify-between gap-2">
                    <span class="text-xs font-medium text-text-primary truncate">{n.title}</span>
                    <span class="text-[10px] text-text-muted shrink-0">{formatRelativeTime(n.timestamp)}</span>
                  </div>
                  <p class="text-[11px] text-text-secondary mt-0.5 leading-snug line-clamp-2">{n.description}</p>
                </div>
              </button>
            {/each}
          {/if}
        </div>
      </div>
    </div>
  {/if}
</div>
