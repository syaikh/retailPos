<script lang="ts">
  import { useWebSocket } from '$shared/api/websocket';
  import { getCurrentJakartaDateDisplay, getCurrentJakartaClock } from '$shared/utils/jakartaTime';
  import NotificationBell from '$app/layouts/NotificationBell.svelte';
  import { Menu } from 'lucide-svelte';

  let {
    currentPath = '/',
    ontogglemenu = () => {},
    isMobileMenuOpen = false,
  }: {
    currentPath?: string;
    ontogglemenu?: () => void;
    isMobileMenuOpen?: boolean;
  } = $props();
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
		'/stores':             'Stores',
		'/stores/import-history': 'Import History',
		'/brands':             'Brands',
		'/units-of-measure':   'Units',
		'/pricing-rules':      'Pricing Rules',
		'/customer-groups':    'Customer Groups',
		'/suppliers':          'Suppliers',
		'/shifts':             'Shifts',
		'/purchase-orders':    'Purchase Orders',
		'/categories/import-history': 'Import History',
		'/brands/import-history':     'Import History',
		'/units-of-measure/import-history': 'Import History',
		'/customers/import-history':  'Import History',
		'/products/import-history':   'Import History',
	};

	const importHistoryParents: Record<string, { label: string; href: string }> = {
		'/categories/import-history':        { label: 'Categories', href: '/categories' },
		'/brands/import-history':            { label: 'Brands', href: '/brands' },
		'/units-of-measure/import-history':  { label: 'Units', href: '/units-of-measure' },
		'/customers/import-history':         { label: 'Customers', href: '/customers' },
		'/products/import-history':          { label: 'Products', href: '/inventory/products' },
		'/stores/import-history':            { label: 'Stores', href: '/stores' },
	};

    const parts: { label: string; href: string }[] = [];

    if (path !== '/') {
      parts.push({ label: 'Home', href: '/' });

      const adminPaths = ['/admin/users', '/admin/roles', '/admin/audit-logs', '/stores'];
      if (adminPaths.includes(path)) {
        parts.push({ label: 'Administration', href: '/admin' });
      }

      const parent = importHistoryParents[path];
      if (parent) {
        parts.push(parent);
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

<header class="flex items-center h-14 md:h-16 px-4 md:px-6 border-b border-border-default bg-surface-default">
  <!-- Hamburger (mobile only) -->
  <button
    type="button"
    class="md:hidden p-2 -ml-2 mr-2 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
    onclick={ontogglemenu}
    aria-label="Toggle navigation menu"
    aria-expanded={isMobileMenuOpen}
  >
    <Menu size={20} />
  </button>

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
