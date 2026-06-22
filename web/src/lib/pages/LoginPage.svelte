<script>
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import { goto } from '$lib/router';
  import { login } from '$lib/api/auth';
  import { auth } from '$lib/stores/auth';
  import { Eye, EyeOff, Store, ShieldCheck } from 'lucide-svelte';
  import { fade, fly } from 'svelte/transition';

  let username = $state('');
  let password = $state('');
  let loading = $state(false);
  let errorMsg = $state('');
  let showPassword = $state(false);

  async function handleLogin(e) {
    e.preventDefault();
    errorMsg = '';
    if (!username.trim() || !password.trim()) {
      errorMsg = 'Username and password are required';
      return;
    }
    loading = true;
    const result = await login(username.trim(), password);
    loading = false;
    if (result && result !== false) {
      auth.setUser(result.user);
      goto('/');
    } else {
      errorMsg = 'Invalid username or password';
    }
  }
</script>

<div class="h-dvh w-full flex bg-bg overflow-hidden" in:fade={{ duration: 400 }}>

  <!-- Left panel — animated brand -->
  <div class="hidden md:flex flex-col flex-1 relative bg-bg-secondary overflow-hidden">
    <!-- Mesh gradient blobs -->
    <div class="absolute inset-0 overflow-hidden pointer-events-none">
      <div class="absolute -top-32 -left-32 w-[500px] h-[500px] rounded-full bg-primary/20 blur-[120px]"></div>
      <div class="absolute top-1/2 left-1/3 w-[400px] h-[400px] rounded-full bg-accent/15 blur-[100px]"></div>
      <div class="absolute -bottom-48 right-0 w-[600px] h-[600px] rounded-full bg-primary/10 blur-[140px]"></div>
    </div>

    <!-- Grid overlay -->
    <div class="absolute inset-0 opacity-5"
      style="background-image: linear-gradient(#7c3aed 1px, transparent 1px), linear-gradient(90deg, #7c3aed 1px, transparent 1px); background-size: 40px 40px;">
    </div>

    <!-- Content -->
    <div class="relative z-10 flex flex-col justify-center h-full px-16">
      <!-- Logo -->
      <div class="flex items-center gap-4 mb-12">
        <div class="w-14 h-14 rounded-2xl gradient-bg-primary shadow-glow-primary flex items-center justify-center">
          <Store size={28} class="text-white" />
        </div>
        <div>
          <h1 class="text-2xl font-bold text-text-primary">RetailPOS</h1>
          <p class="text-sm text-text-muted">Management System</p>
        </div>
      </div>

      <!-- Feature list -->
      <h2 class="text-4xl font-bold text-text-primary leading-tight mb-4">
        Powerful POS<br/>
        <span class="gradient-text">Made Simple</span>
      </h2>
      <p class="text-text-secondary text-lg mb-10 max-w-md">
        Streamline sales, inventory, and reporting in one unified platform built for modern retail.
      </p>

      <div class="space-y-4">
        {#each ['Real-time inventory tracking', 'Multi-role access control', 'Sales analytics & reports', 'WebSocket live updates'] as feat}
          <div class="flex items-center gap-3">
            <div class="w-6 h-6 rounded-full bg-primary-subtle border border-primary/30 flex items-center justify-center shrink-0">
              <ShieldCheck size={12} class="text-primary-light" />
            </div>
            <span class="text-sm text-text-secondary">{feat}</span>
          </div>
        {/each}
      </div>
    </div>
  </div>

  <!-- Right panel — login form -->
  <div class="flex flex-col justify-center w-full md:w-[480px] px-8 md:px-16 bg-surface/30 backdrop-blur-2xl border-l border-border/30 relative z-20" in:fly={{ x: 20, duration: 500, delay: 200 }}>
    <div class="max-w-sm w-full mx-auto">

      <!-- Mobile logo -->
      <div class="flex items-center gap-3 mb-10 lg:hidden">
        <div class="w-10 h-10 rounded-xl gradient-bg-primary shadow-glow-primary-sm flex items-center justify-center">
          <Store size={20} class="text-white" />
        </div>
        <p class="text-lg font-bold text-text-primary">RetailPOS</p>
      </div>

      <h2 class="text-2xl font-bold text-text-primary mb-1">Welcome back</h2>
      <p class="text-text-muted text-sm mb-8">Sign in to your account to continue</p>

      <form onsubmit={handleLogin} class="space-y-5">
        <div>
          <label for="username" class="block text-sm font-medium text-text-secondary mb-2">Username</label>
          <Input
            id="username"
            type="text"
            placeholder="Enter your username"
            class="bg-surface-subtle border-transparent focus:bg-bg focus:border-primary-light focus:ring-1 focus:ring-primary-light/50 transition-all"
            bind:value={username}
            disabled={loading}
            autocomplete="username"
          />
        </div>

        <div>
          <label for="password" class="block text-sm font-medium text-text-secondary mb-2">Password</label>
          <div class="relative">
            <Input
              id="password"
              type={showPassword ? 'text' : 'password'}
              placeholder="Enter your password"
              class="pr-11 bg-surface-subtle border-transparent focus:bg-bg focus:border-primary-light focus:ring-1 focus:ring-primary-light/50 transition-all"
              bind:value={password}
              disabled={loading}
              autocomplete="current-password"
            />
            <button
              type="button"
              class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
              onclick={() => showPassword = !showPassword}
              aria-label={showPassword ? 'Hide password' : 'Show password'}
            >
              {#if showPassword}
                <EyeOff size={16} />
              {:else}
                <Eye size={16} />
              {/if}
            </button>
          </div>
        </div>

        {#if errorMsg}
          <div class="flex items-center gap-2 p-3 rounded-xl bg-danger-subtle border border-danger/25 text-danger-light text-sm" role="alert">
            <span class="w-1.5 h-1.5 rounded-full bg-danger shrink-0"></span>
            {errorMsg}
          </div>
        {/if}

        <Button variant="primary" type="submit" class="w-full py-3.5 text-base mt-2 shadow-glow-primary hover:-translate-y-0.5 active:scale-95 transition-all" disabled={loading}>
          {#if loading}
            <span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
            Signing in…
          {:else}
            Sign In
          {/if}
        </Button>
      </form>

      <p class="text-xs text-text-muted text-center mt-8">
        © {new Date().getFullYear()} RetailPOS Management System
      </p>
    </div>
  </div>
</div>
