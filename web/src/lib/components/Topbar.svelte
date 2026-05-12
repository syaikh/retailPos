<script lang="ts">
  import { Bell, Search } from 'lucide-svelte';
  import { useWebSocket } from '$lib/composables/useWebSocket';

  let { currentPath = '/' }: { currentPath?: string } = $props();
  let ws = useWebSocket();
  let wsStatus = $derived($ws.status);

  // Build breadcrumb from path
  const breadcrumb = $derived(() => {
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

    if (currentPath !== '/') {
      parts.push({ label: 'Home', href: '/' });

      if (currentPath.startsWith('/admin/')) {
        parts.push({ label: 'Administration', href: '/admin' });
      }
    }

    const label = map[currentPath] || currentPath;
    parts.push({ label, href: currentPath });
    return parts;
  });
</script>

<header class="topbar px-6 py-4">
  <!-- Breadcrumb -->
  <nav class="flex items-center gap-2" aria-label="Breadcrumb">
    {#each breadcrumb() as crumb, i}
      {#if i > 0}
        <span class="text-text-muted text-xs">/</span>
      {/if}
      {#if i === breadcrumb().length - 1}
        <span class="text-lg font-bold tracking-tight text-white">
          {crumb.label}
        </span>
      {:else}
        <span class="text-xs text-text-muted hover:text-text-secondary transition-colors cursor-pointer">
          {crumb.label}
        </span>
      {/if}
    {/each}
  </nav>

  <div class="ml-auto flex items-center gap-2">
    <!-- WebSocket connection status -->
    <div class="flex items-center gap-1.5 text-xs" title="Real-time connection">
      <span class="w-2 h-2 rounded-full {wsStatus === 'connected' ? 'bg-success animate-pulse-dot' : wsStatus === 'connecting' ? 'bg-warning animate-pulse' : 'bg-text-muted'}"></span>
      <span class="text-text-muted hidden lg:inline">{wsStatus === 'connected' ? 'Online' : wsStatus === 'connecting' ? 'Connecting...' : 'Offline'}</span>
    </div>
    
    <!-- Notification bell -->
    <button class="btn-icon btn-ghost relative text-text-muted hover:text-text-primary">
      <Bell size={18} />
      <span class="absolute top-1.5 right-1.5 w-1.5 h-1.5 bg-danger rounded-full animate-pulse-dot"></span>
    </button>

    <div class="w-px h-6 bg-border mx-1"></div>

    <!-- Date/time -->
    <span class="text-xs text-text-muted hidden lg:block">
      {new Date().toLocaleDateString('id-ID', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}
    </span>
  </div>
</header>
