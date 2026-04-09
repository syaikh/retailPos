<script>
  import { page } from '$app/stores';
  import { protectRoute } from '$lib/auth.js';
  import '../lib/app.css';
  import { user } from '$lib/stores.js';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import Navbar from '$lib/components/Navbar.svelte';
  import Login from '$lib/components/Login.svelte';

  let { children } = $props();

  $effect(() => {
    protectRoute($page.url.pathname);
  });
</script>

{#if $user}
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
