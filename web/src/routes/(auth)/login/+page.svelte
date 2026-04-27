<script lang="ts">
  import { auth } from '$lib/stores/auth';
  import client from '$lib/api/client';
  import { goto } from '$app/navigation';
  import { LogIn, Key, User } from 'lucide-svelte';

  let username = $state('');
  let password = $state('');
  let error = $state('');
  let loading = $state(false);

  async function handleLogin(event: SubmitEvent) {
    event.preventDefault();
    error = '';
    loading = true;
    try {
      const { data: loginData } = await client.post('/login', { username, password });
      
      if (loginData.refresh_token) {
        sessionStorage.setItem('refresh_token', loginData.refresh_token);
      }
      
      const { data } = await client.get('/auth/validate');
      
      if (data.user) {
        auth.setUser(data.user);
        
        const perms = data.user.permissions || [];
        if (perms.includes('dashboard:read')) {
          goto('/');
        } else if (perms.includes('pos:access')) {
          goto('/pos');
        } else {
          goto('/pos');
        }
      }
    } catch (e: any) {
      error = e.response?.data?.error || 'Login failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="min-h-screen flex items-center justify-center bg-[radial-gradient(circle_at_top_right,#1e1b4b,#0f172a)]">
  <div class="w-full max-w-[400px] p-10 bg-[var(--color-glass)] backdrop-blur-md border border-[var(--color-border)] rounded-xl">
    <div class="text-center mb-8">
      <LogIn size={40} color="var(--color-primary)" />
      <h1 class="mt-4 text-2xl font-bold">RetailPOS</h1>
      <p class="text-sm text-[var(--color-text-secondary)]">Masuk ke sistem kasir & stok</p>
    </div>

    <form onsubmit={handleLogin} class="space-y-5">
      <div class="mb-5">
        <label for="username" class="block mb-2 text-sm text-[var(--color-text-secondary)]">Username</label>
        <div class="relative">
          <span class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-text-secondary)]">
            <User size={18} />
          </span>
          <input
            type="text"
            id="username"
            bind:value={username}
            placeholder="Username admin/cashier"
            required
            class="w-full pl-10 h-12 bg-transparent border border-[var(--color-border)] rounded-lg text-white placeholder-[var(--color-text-secondary)] focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-colors"
          />
        </div>
      </div>

      <div class="mb-5">
        <label for="password" class="block mb-2 text-sm text-[var(--color-text-secondary)]">Password</label>
        <div class="relative">
          <span class="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-text-secondary)]">
            <Key size={18} />
          </span>
          <input
            type="password"
            id="password"
            bind:value={password}
            placeholder="••••••••"
            required
            class="w-full pl-10 h-12 bg-transparent border border-[var(--color-border)] rounded-lg text-white placeholder-[var(--color-text-secondary)] focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-colors"
          />
        </div>
      </div>

      {#if error}
        <div class="p-2.5 mb-4 rounded-md text-sm text-center text-[var(--color-danger)] bg-[var(--color-danger)]/10">{error}</div>
      {/if}

      <button
        type="submit"
        disabled={loading}
        class="w-full h-12 bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white font-medium mt-3 rounded-lg transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {#if loading}
          <span class="flex items-center justify-center gap-2">
            <svg class="animate-spin h-5 w-5" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            Memproses...
          </span>
        {:else}
          Masuk
        {/if}
      </button>
    </form>
  </div>
</div>
