<script>
  import '../../lib/app.css';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import Navbar from '$lib/components/Navbar.svelte';
  let { children } = $props();
</script>

<div class="layout">
  <Sidebar />
  <div class="main-content">
    <Navbar />
    <main>
      {@render children()}
    </main>
  </div>
</div>

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