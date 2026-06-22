<script lang="ts">
  import Button from '$lib/components/ui/Button.svelte';
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { Bell } from 'lucide-svelte';
  import { useWebSocket } from '$lib/composables/useWebSocket';
  import { goto } from '$lib/router';
  import { toast } from '$lib/stores/toast';
  import {
    notifications,
    formatRelativeTime,
    getNotificationIcon,
    type Notification,
  } from '$lib/stores/notifications';

  const ws = useWebSocket();

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
    if (n.navigateTo) goto(n.navigateTo);
  }

  function handleMarkAllRead(e: Event) {
    e.stopPropagation();
    notifications.markAllRead();
  }

  // Register WebSocket listeners once on mount
  onMount(() => {
    const unsubs = [
      ws.on('low_stock_alert', (data: any) => {
        notifications.push({
          type: 'low_stock',
          title: 'Stok Menipis',
          description: `${data.name} — sisa ${data.stock}`,
          navigateTo: '/inventory/products?low_stock=true',
        });
        toast.error(`Low stock alert: ${data.name} (stock: ${data.stock})`);
      }),
      ws.on('sale_created', (data: any) => {
        notifications.push({
          type: 'sale_created',
          title: 'Transaksi Baru',
          description: `${data.invoice} — Rp ${(data.total || 0).toLocaleString('id-ID')}`,
          navigateTo: `/transactions/${data.id}`,
        });
      }),
      ws.on('stock_update', (data: any) => {
        const status = data.low_stock ? '⚠️ low stock' : `${data.stock} units`;
        notifications.push({
          type: 'stock_update',
          title: 'Stok Diubah',
          description: `${data.sku} — ${status}`,
          navigateTo: `/inventory/products/${data.id}`,
        });
      }),
      ws.on('product_updated', (data: any) => {
        notifications.push({
          type: 'product_updated',
          title: 'Produk Diubah',
          description: `${data.sku} — harga: Rp ${(data.price || 0).toLocaleString('id-ID')}`,
          navigateTo: `/inventory/products/${data.id}`,
        });
      }),
    ];
    return () => unsubs.forEach(fn => fn());
  });
</script>

<div bind:this={container} class="relative">
  <Button variant="ghost" size="icon" class="relative text-text-muted hover:text-text-primary" onclick={handleClick} aria-label="Notifications">
    <Bell size={18} />
    {#if unread > 0}
      <span class="absolute -top-0.5 -right-0.5 min-w-[16px] h-4 flex items-center justify-center bg-danger text-white text-[10px] font-bold rounded-full px-1 leading-none shadow-glow-danger">
        {unread > 99 ? '99+' : unread}
      </span>
    {/if}
  </Button>

  {#if open}
    <div class="absolute right-0 top-full mt-2 z-50" onclick={(e) => e.stopPropagation()} role="none" onkeydown={(e) => { if (e.key !== 'Escape') e.stopPropagation(); }} transition:fly={{ y: -8, duration: 200 }}>
      <div class="card-glass p-2 w-80 max-h-96 flex flex-col">
        <!-- Header -->
        <div class="flex items-center justify-between px-2 py-1.5 border-b border-border/50">
          <span class="text-xs font-semibold text-text-primary uppercase tracking-wide">
            Notifications {unread > 0 ? `(${unread})` : ''}
          </span>
          {#if items.length > 0 && unread > 0}
            <button
              class="text-[10px] font-medium text-primary-light hover:text-primary transition-colors"
              onclick={handleMarkAllRead}
            >
              Mark all read
            </button>
          {/if}
        </div>

        <!-- Body -->
        <div class="overflow-y-auto flex-1 max-h-80">
          {#if items.length === 0}
            <div class="flex flex-col items-center justify-center py-8 text-text-muted">
              <Bell size={24} class="opacity-30 mb-2" />
              <span class="text-xs">No notifications yet</span>
            </div>
          {:else}
            {#each items as n (n.id)}
              <button
                class="w-full text-left flex items-start gap-2.5 px-2 py-2.5 rounded-lg transition-colors hover:bg-surface-hover/50 {n.read ? 'opacity-60' : ''}"
                onclick={() => handleNotificationClick(n)}
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
