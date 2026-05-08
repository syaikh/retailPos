<script>
  import { goto, getPath, subscribe } from '$lib/router';
  import LoginPage from '$lib/pages/LoginPage.svelte';
  import Home from '$lib/pages/Home.svelte';
  import PosPage from '$lib/pages/PosPage.svelte';
  import InventoryPage from '$lib/pages/InventoryPage.svelte';
  import ReportsPage from '$lib/pages/ReportsPage.svelte';
  import AdminUsers from '$lib/pages/admin/Users.svelte';
  import AdminRoles from '$lib/pages/admin/Roles.svelte';
  import AdminAuditLogs from '$lib/pages/admin/AuditLogs.svelte';
  import Layout from '$lib/components/Layout.svelte';
  import Toast from '$lib/components/ui/Toast.svelte';
  import { auth } from '$lib/stores/auth';
  import { checkAuth, restoreSession } from '$lib/api/auth';
  import { fade } from 'svelte/transition';

  let Component = $state(LoginPage);
  let currentPath = $state(getPath());
  let isInitializing = $state(true);

  function getComponent(path) {
    switch (path) {
      case '/login':            return LoginPage;
      case '/pos':              return PosPage;
      case '/inventory':        return InventoryPage;
      case '/reports':          return ReportsPage;
      case '/admin':            return AdminUsers;
      case '/admin/users':      return AdminUsers;
      case '/admin/roles':      return AdminRoles;
      case '/admin/audit-logs': return AdminAuditLogs;
      default:                  return Home;
    }
  }

  function handleRoute(path) {
    const token = sessionStorage.getItem('access_token');
    const hasValidToken = token && token !== 'null' && token !== 'undefined' && token.length > 10;

    if (path !== '/login' && !hasValidToken) {
      Component = LoginPage;
      currentPath = '/login';
      window.history.replaceState({}, '', '/login');
      isInitializing = false;
      return;
    }

    if (path === '/login' && hasValidToken) {
      goto('/');
      return;
    }

    currentPath = path;
    isInitializing = false;
    Component = getComponent(path);
  }

  async function initializeRoute(path) {
    const token = sessionStorage.getItem('access_token');
    const hasToken = token && token !== 'null' && token !== 'undefined' && token.length > 10;
    let isAuthenticated = hasToken;

    if (!hasToken) {
      // Try to restore session using HttpOnly refresh_token cookie
      const restoreResult = await restoreSession();
      isAuthenticated = restoreResult.success;
      if (isAuthenticated && restoreResult.user) {
        auth.setUser(restoreResult.user);
      }
    }

    if (!isAuthenticated) {
      if (path !== '/login') {
        Component = LoginPage;
        currentPath = '/login';
        window.history.replaceState({}, '', '/login');
      } else {
        Component = LoginPage;
        currentPath = '/login';
      }
      isInitializing = false;
      subscribe(handleRoute);
      return;
    }

    // Token is valid, proceed to route
    if (path === '/login') {
      Component = Home;
      currentPath = '/';
      window.history.replaceState({}, '', '/');
    } else {
      currentPath = path;
      Component = getComponent(path);
    }

    isInitializing = false;
    subscribe(handleRoute);
  }

  // Initial route resolution
  const initialPath = getPath();
  initializeRoute(initialPath);
</script>

{#if isInitializing}
  <!-- Boot splash -->
  <div class="min-h-screen bg-bg flex items-center justify-center absolute inset-0 z-50" out:fade={{ duration: 300 }}>
    <div class="flex flex-col items-center gap-4">
      <div class="w-12 h-12 rounded-2xl gradient-bg-primary flex items-center justify-center shadow-glow-primary animate-pulse">
        <span class="text-white text-xl font-bold">R</span>
      </div>
      <div class="flex gap-1.5">
        <span class="w-2 h-2 bg-primary rounded-full animate-bounce" style="animation-delay:0ms"></span>
        <span class="w-2 h-2 bg-primary rounded-full animate-bounce" style="animation-delay:150ms"></span>
        <span class="w-2 h-2 bg-primary rounded-full animate-bounce" style="animation-delay:300ms"></span>
      </div>
    </div>
  </div>
{:else if currentPath === '/login'}
  <Component />
{:else}
  <Layout {currentPath}>
    <Component />
  </Layout>
{/if}

<!-- Global toast notifications -->
<Toast />
