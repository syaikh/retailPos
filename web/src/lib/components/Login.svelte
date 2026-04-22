<script lang="ts">
  import { auth } from '$lib/stores/auth';
  import api from '$lib/api.js';
  import { goto } from '$app/navigation';
  import { invalidateAll } from '$app/navigation';
  import { LogIn, Key, User } from 'lucide-svelte';

  let username = $state('');
  let password = $state('');
  let error = $state('');
  let loading = $state(false);

   async function handleLogin(event: Event) {
    event.preventDefault();
    error = '';
    loading = true;
    try {
      const resp = await api.post('/login', { username, password });
      auth.setUser(resp.data.user);
      await invalidateAll();
      goto('/', { replaceState: true });
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        error = (e as any).response?.data?.error || 'Login failed';
      } else {
        error = 'Login failed';
      }
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-container">
  <div class="login-card premium-card glass">
    <div class="header">
      <LogIn size={40} color="var(--primary)" />
      <h1>RetailPOS</h1>
      <p>Masuk ke sistem kasir & stok</p>
    </div>

    <form onsubmit={handleLogin}>
      <div class="field">
        <label for="username">Username</label>
        <div class="input-wrapper">
          <span class="icon"><User size={18} /></span>
          <input type="text" id="username" bind:value={username} placeholder="Username admin/cashier" required />
        </div>
      </div>

      <div class="field">
        <label for="password">Password</label>
        <div class="input-wrapper">
          <span class="icon"><Key size={18} /></span>
          <input type="password" id="password" bind:value={password} placeholder="••••••••" required />
        </div>
      </div>

      {#if error}
        <div class="error-msg">{error}</div>
      {/if}

      <button type="submit" class="login-btn" disabled={loading}>
        {loading ? 'Memproses...' : 'Login'}
      </button>
    </form>
  </div>
</div>

<style>
  .login-container {
    height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: radial-gradient(circle at top right, #1e1b4b, #0f172a);
  }

  .login-card {
    width: 100%;
    max-width: 400px;
    padding: 40px;
  }

  .header {
    text-align: center;
    margin-bottom: 32px;
  }

  .header h1 {
    margin-top: 16px;
    font-size: 1.75rem;
    font-weight: 800;
  }

  .header p {
    color: var(--text-secondary);
    font-size: 0.875rem;
  }

  .field {
    margin-bottom: 20px;
  }

  label {
    display: block;
    margin-bottom: 8px;
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .input-wrapper {
    position: relative;
  }

  .icon {
    position: absolute;
    left: 12px;
    top: 50%;
    transform: translateY(-50%);
    color: var(--text-secondary);
  }

  input {
    width: 100%;
    padding-left: 40px;
    height: 48px;
  }

  .login-btn {
    width: 100%;
    height: 48px;
    background: var(--primary);
    color: white;
    font-size: 1rem;
    margin-top: 12px;
  }

  .login-btn:hover {
    background: var(--primary-hover);
  }

  .error-msg {
    color: var(--danger);
    background: rgba(239, 68, 68, 0.1);
    padding: 10px;
    border-radius: 6px;
    font-size: 0.875rem;
    margin-bottom: 16px;
    text-align: center;
  }
</style>
