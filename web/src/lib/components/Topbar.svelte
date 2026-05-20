<script lang="ts">
  import { Bell } from 'lucide-svelte';
  import { useWebSocket } from '$lib/composables/useWebSocket';

  let { currentPath = '/' }: { currentPath?: string } = $props();
  const ws = useWebSocket();
  
  // Get the status store reference
  const status = ws.status;

  // Breadcrumb computed as a derived value
  const breadcrumb = $derived(getBreadcrumb(currentPath));

  function getBreadcrumb(path: string): { label: string; href: string }[] {
    const map: Record<string, string> = {
      '/':                  'Dashboard',
      '/pos':               'Point of Sale',
      '/inventory':         'Inventory',
      '/reports':           'Reports',
      '/admin':             'Administration',
      '/admin/users':       'Users',
      '/admin/roles':       'Roles',
      '/admin/audit-logs':  'Audit Logs',
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

  const dateString = $derived(new Date().toLocaleDateString('id-ID', { 
    weekday: 'long', 
    year: 'numeric', 
    month: 'long', 
    day: 'numeric' 
  }));

  function handleNotifications() {
    // TODO: Open notifications panel
    console.log('Notifications clicked');
  }
</script>

<header class="flex items-center px-6 py-4 border-b border-border-default bg-surface-default">
  <!-- Breadcrumb -->
  <nav class="flex items-center gap-2" aria-label="Breadcrumb">
    {#each breadcrumb as crumb, i}
      {#if i > 0}
        <span class="text-text-muted text-xs">/</span>
      {/if}
      {#if i === breadcrumb.length - 1}
        <span class="text-lg font-bold tracking-tight text-white">
          {crumb.label}
        </span>
      {:else}
        <a href={crumb.href} class="text-xs text-text-muted hover:text-text-secondary transition-colors cursor-pointer">
          {crumb.label}
        </a>
      {/if}
    {/each}
  </nav>

  <div class="ml-auto flex items-center gap-2">
    <!-- WebSocket connection status -->
    <div class="flex items-center gap-1.5 text-xs" title="Real-time connection">
      <span class="w-2 h-2 rounded-full {$status === 'connected' ? 'bg-success animate-pulse-dot' : $status === 'connecting' ? 'bg-warning animate-pulse' : 'bg-text-muted'}"></span>
      <span class="text-text-muted hidden lg:inline">{$status === 'connected' ? 'Online' : $status === 'connecting' ? 'Connecting...' : 'Offline'}</span>
    </div>
    
    <!-- Notification bell -->
    <button class="btn-icon btn-ghost relative text-text-muted hover:text-text-primary" onclick={handleNotifications}>
      <Bell size={18} />
      <span class="absolute top-1.5 right-1.5 w-1.5 h-1.5 bg-danger rounded-full animate-pulse-dot"></span>
    </button>

    <div class="w-px h-6 border-l border-border-default mx-1"></div>

    <!-- Date/time -->
    <span class="text-xs text-text-muted hidden lg:block">
      {dateString}
    </span>
  </div>
</header>
