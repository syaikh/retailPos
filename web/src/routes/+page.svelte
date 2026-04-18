<script>
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { checkAuth } from '$lib/api/auth';

  onMount(async () => {
    const isAuth = await checkAuth();
    if (isAuth) {
      goto('/dashboard');
    } else {
      goto('/login');
    }
  });
</script>

<div class="loading">
  <p>Memuat...</p>
</div>

<style>
  .loading {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100vh;
    background: var(--bg-main);
  }
</style>
