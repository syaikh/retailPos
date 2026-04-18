<script>
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

    // Cashier restricted to /pos only
    if (state.user.role === 'cashier' && !currentPath.startsWith('/pos')) {
      goto('/pos');
      return;
    }

    // Admin or authorized cashier -> show content
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

    // Admin has full access
    if (state.user.role === 'admin') {
      return;
    }

    // Cashier restricted to /pos only
    if (state.user.role === 'cashier' && !path.startsWith('/pos')) {
      nav.cancel();
      goto('/pos');
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