<script lang="ts">
  import { useWebSocket } from '$lib/composables/useWebSocket';
  import { getCurrentJakartaDateDisplay, getCurrentJakartaClock } from '$lib/utils/jakartaTime';
  import NotificationBell from '$lib/components/NotificationBell.svelte';

  let { currentPath = '/' }: { currentPath?: string } = $props();
  const ws = useWebSocket();
  
  // Get the status store reference
  const status = ws.status;

  // Breadcrumb computed as a derived value
  const breadcrumb = $derived(getBreadcrumb(currentPath));

  function getBreadcrumb(path: string): { label: string; href: string }[] {
    const map: Record<string, string> = {
      '/':                   'Dashboard',
      '/pos':                'Point of Sale',
      '/inventory/products': 'Products',
      '/reports':            'Reports',
      '/transactions':       'Transactions',
      '/categories':         'Categories',
      '/customers':          'Customers',
      '/admin':              'Administration',
      '/admin/users':        'Users',
      '/admin/roles':        'Roles',
      '/admin/audit-logs':   'Audit Logs',
    };

    const parts: { label: string; href: string }[] = [];

    if (path !== '/') {
      parts.push({ label: 'Home', href: '/' });

      if (path.startsWith('/admin/')) {
        parts.push({ label: 'Administration', href: '/admin' });
      }
    }

    const label = map[path] || path;
    parts.push({ label, href: path });
    return parts;
  }

  // Live clock - updates every minute using Asia/Jakarta timezone
  let jakartaClock = $state(getCurrentJakartaClock());
  let jakartaDate = $state(getCurrentJakartaDateDisplay());

  $effect(() => {
    const timer = setInterval(() => {
      jakartaClock = getCurrentJakartaClock();
      jakartaDate = getCurrentJakartaDateDisplay();
    }, 60000);
    return () => clearInterval(timer);
  });

  const dateTimeString = $derived(
    `${jakartaDate.weekday}, ${jakartaDate.day} ${jakartaDate.month} ${jakartaDate.year} • ${jakartaClock.hours}:${jakartaClock.minutes}`
  );

</script>

<header class="flex items-center h-16 px-6 border-b border-border-default bg-surface-default">
  <!-- Breadcrumb -->
  <nav class="flex items-center gap-2" aria-label="Breadcrumb">
    {#each breadcrumb as crumb, i}
      {#if i > 0}
        <span class="text-text-muted text-xs">/</span>
      {/if}
      {#if i === breadcrumb.length - 1}
        <h1 class="text-lg font-bold tracking-tight text-white">
          {crumb.label}
        </h1>
      {:else}
        <a href={crumb.href} class="text-xs text-text-muted hover:text-text-secondary transition-colors cursor-pointer">
          {crumb.label}
        </a>
      {/if}
    {/each}
  </nav>

  <div class="ml-auto flex items-center gap-4">
    <!-- WebSocket connection status -->
    <div class="flex items-center gap-2 text-xs" title="Real-time connection">
      <span class="w-2 h-2 rounded-full {$status === 'connected' ? 'bg-success animate-pulse-dot' : $status === 'connecting' ? 'bg-warning animate-pulse' : 'bg-text-muted'}"></span>
      <span class="text-text-muted hidden lg:inline">{$status === 'connected' ? 'Online' : $status === 'connecting' ? 'Connecting...' : 'Offline'}</span>
    </div>
    
    <!-- Notification bell -->
    <NotificationBell />

    <!-- Subtle divider -->
    <div class="w-px h-4 bg-border-subtle"></div>

    <!-- Date & Time -->
    <span class="text-xs text-text-muted font-medium tracking-wide">
      {dateTimeString}
    </span>
  </div>
</header>
