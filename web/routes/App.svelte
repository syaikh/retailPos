<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { auth } from '$lib/stores/auth';
  import { ui } from '$lib/stores/ui';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import Navbar from '$lib/components/Navbar.svelte';

  // Pages
  import LoginPage from '$lib/pages/LoginPage.svelte';
  import DashboardPage from '$lib/pages/DashboardPage.svelte';
  import PosPage from '$lib/pages/PosPage.svelte';
  import InventoryPage from '$lib/pages/InventoryPage.svelte';
  import SalesPage from '$lib/pages/SalesPage.svelte';
  import AdminUsers from '$lib/pages/admin/Users.svelte';
  import AdminRoles from '$lib/pages/admin/Roles.svelte';
  import AdminAuditLogs from '$lib/pages/admin/AuditLogs.svelte';

  import * as router from '$lib/router';

  interface RouteConfig {
    path: string;
    component: any;
    permissions?: string[];
  }

  const routes: RouteConfig[] = [
    { path: '/login', component: LoginPage },
    { path: '/', component: DashboardPage, permissions: ['dashboard:read'] },
    { path: '/dashboard', component: DashboardPage, permissions: ['dashboard:read'] },
    { path: '/pos', component: PosPage, permissions: ['pos:access'] },
    { path: '/inventory', component: InventoryPage, permissions: ['inventory:read'] },
    { path: '/sales', component: SalesPage, permissions: ['sale:read'] },
    { path: '/admin/users', component: AdminUsers, permissions: ['user:read'] },
    { path: '/admin/roles', component: AdminRoles, permissions: ['user:read'] },
    { path: '/admin/audit-logs', component: AdminAuditLogs, permissions: ['audit:read'] },
  ];

  let user: any = null;
  let currentPath = router.getPath();
  let unsubscribe: (() => void) | null = null;
  let initialized = false;

  function getRouteForPath(path: string): RouteConfig | undefined {
    // The order matters: longer prefixes first
    const sorted = [...routes].sort((a,b) => b.path.length - a.path.length);
    for (const r of sorted) {
      if (path === r.path || path.startsWith(r.path + '/') || (r.path !== '/' && path.startsWith(r.path))) {
        return r;
      }
    }
    return undefined;
  }

  function hasPermission(userPerms: string[], required: string[]): boolean {
    return required.every(p => userPerms.includes(p));
  }

  function fallbackRoute(perms: string[]): string {
    if (perms.includes('pos:access')) return '/pos';
    if (perms.includes('dashboard:read')) return '/';
    return '/pos';
  }

  onMount(async () => {
    // Subscribe to router
    unsubscribe = router.subscribe(() => {
      currentPath = router.getPath();
    });

    // Load auth
    user = await auth.checkAuth();

    // Handle unauthenticated
    if (!user && currentPath !== '/login') {
      router.goto('/login');
      return;
    }

    if (user && currentPath === '/login') {
      router.goto('/');
      return;
    }

    if (user) {
      // Check permissions for the matched route
      const route = getRouteForPath(currentPath);
      if (route && route.permissions) {
        const userPerms = user.permissions || [];
        if (!hasPermission(userPerms, route.permissions)) {
          const fb = fallbackRoute(userPerms);
          ui.warn('Anda tidak memiliki akses ke halaman ini');
          router.goto(fb);
          return;
        }
      }
    }

    initialized = true;
  });

  onDestroy(() => {
    if (unsubscribe) unsubscribe();
  });

  // Reactively re-check when path or user changes
  $: if (initialized && user) {
    const route = getRouteForPath(currentPath);
    if (route && route.permissions && !hasPermission(user.permissions || [], route.permissions)) {
      const fb = fallbackRoute(user.permissions || []);
      router.goto(fb);
    }
  }

  // Determine page component
  let PageComponent: any = null;
  $: {
    const match = getRouteForPath(currentPath);
    PageComponent = match ? match.component : LoginPage;
  }

  // Show loading auth state
  $: loading = !initialized || !user;
</script>

{#if loading}
  <div class="min-h-screen flex items-center justify-center bg-slate-900">
    <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
  </div>
{:else}
  <div class="layout">
    <Sidebar />
    <div class="main-content">
      <Navbar />
      <main>
        <svelte:component this={PageComponent} />
      </main>
    </div>
  </div>
{/if}

<style>
  .layout {
    display: flex;
    height: 100vh;
    overflow: hidden;
  }
  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
  }
  main {
    padding: 24px;
    flex: 1;
  }
</style>
