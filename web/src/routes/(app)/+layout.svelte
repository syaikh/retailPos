<script lang="ts">
  import '../../lib/app.css';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import Navbar from '$lib/components/Navbar.svelte';
  import { onMount } from 'svelte';
  import { beforeNavigate, goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { auth } from '$lib/stores/auth';
  import { checkAuth } from '$lib/api/auth';

  let { children } = $props();
  let initialized = $state(false);

  // Route permission requirements
  const routePermissions: Record<string, string[]> = {
    '/': ['dashboard:read'],
    '/pos': ['pos:access'],
    '/inventory': ['inventory:read'],
    '/inventory/groups': ['inventory:group:read'],
    '/reports': ['reports:read'],
    '/admin/users': ['users:read'],
    '/admin/roles': ['users:roles:manage'],
  };

  function hasAllPermissions(userPerms: string[], required: string[]): boolean {
    return required.every(p => userPerms.includes(p));
  }

  function getRequiredPermissions(path: string): string[] | null {
    // Check exact match first
    if (routePermissions[path]) {
      return routePermissions[path];
    }
    // Check prefix matches
    for (const route in routePermissions) {
      if (path.startsWith(route)) {
        return routePermissions[route];
      }
    }
    return null;
  }

  // Initial auth check on mount
  onMount(async () => {
    await checkAuth();
    const state = $auth;
    const currentPath = $page.url.pathname;

    // Not authenticated -> redirect to login
    if (!state.user) {
      goto('/login');
      return;
    }

    // Check permission for current route
    const requiredPerms = getRequiredPermissions(currentPath);
    const userPerms = state.user.permissions || [];
    
    if (requiredPerms && !hasAllPermissions(userPerms, requiredPerms)) {
      // Redirect to fallback page based on available permissions
      if (userPerms.includes('pos:access')) {
        goto('/pos');
      } else if (userPerms.includes('dashboard:read')) {
        goto('/');
      } else {
        goto('/pos');
      }
      return;
    }

    // Authorized -> show content
    initialized = true;
  });

  // Client-side navigation guard for subsequent navigations
  beforeNavigate(async (nav) => {
    const state = $auth;
    const path = nav.to?.url?.pathname || '';

    // Allow navigation to login page
    if (path === '/login') {
      return;
    }

    // Not authenticated -> cancel and redirect to login
    if (!state.user) {
      nav.cancel();
      goto('/login');
      return;
    }

    // Check permissions
    const requiredPerms = getRequiredPermissions(path);
    const userPerms = state.user.permissions || [];
    
    if (requiredPerms && !hasAllPermissions(userPerms, requiredPerms)) {
      nav.cancel();
      // Redirect to fallback
      if (userPerms.includes('pos:access')) {
        goto('/pos');
      } else if (userPerms.includes('dashboard:read')) {
        goto('/');
      } else {
        goto('/pos');
      }
    }
  });
</script>

{#if initialized}
  <div class="layout">
    <Sidebar />
    <div class="main-content">
      <Navbar />
      <main>
        {@render children()}
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