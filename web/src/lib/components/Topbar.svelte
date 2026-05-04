<script lang="ts">
  import { Bell, Search } from 'lucide-svelte';

  let { currentPath = '/' }: { currentPath?: string } = $props();

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
