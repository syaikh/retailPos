<script>
  import { onMount } from 'svelte';
  import { protectRoute, authReady } from '$lib/auth.js';
  import '../lib/app.css';
  import { user } from '$lib/stores.js';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import Navbar from '$lib/components/Navbar.svelte';
  import Login from '$lib/components/Login.svelte';

  let { children } = $props();

  // Run once on mount: read the hash and decide where to go.
  // We use onMount (not $effect) to avoid re-running on every reactive update.
  onMount(() => {
    protectRoute();

    // Also guard on hash changes (e.g. user manually edits the URL bar).
    const onHashChange = () => protectRoute();
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  });
</script>

{#if !$authReady}
  <!-- Blank screen while auth state is being determined to prevent flash -->
  <div class="auth-loading"></div>
{:else if $user}
  <div class="layout">
    <Sidebar />
    <div class="main-content">
      <Navbar />
      <main>
        {@render children()}
      </main>
    </div>
  </div>
{:else}
  <Login />
{/if}

<style>
  .auth-loading {
    width: 100vw;
    height: 100vh;
    background: var(--bg-main, #0f172a);
  }

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
    position: relative;
  }

  main {
    padding: 24px;
    flex: 1;
    overflow: visible;
  }
</style>
