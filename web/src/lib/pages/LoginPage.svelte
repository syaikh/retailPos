<script>
  import { goto } from '$lib/router';
  import { login } from '$lib/api/auth';

  let username = $state('');
  let password = $state('');
  let loading = $state(false);
  let errorMsg = $state('');
  let errorVisible = $state(false);
</script>

<div id="login-section">
  <header class="text-center mb-8">
    <h1 class="text-3xl font-bold bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent mb-2">
      Retail POS System
    </h1>
    <p class="text-slate-400">Modern Point of Sale Management</p>
  </header>

  <div class="max-w-md mx-auto bg-slate-800/50 backdrop-blur-sm p-8 rounded-xl border border-slate-700">
    <h2 class="text-xl font-bold text-center mb-6 text-white">Login to Retail POS</h2>

    <form onsubmit={async (e) => {
      e.preventDefault();
      errorVisible = false;
      
      if (!username.trim() || !password.trim()) {
        errorMsg = 'Username and password are required';
        errorVisible = true;
        return;
      }

      loading = true;
      const success = await login(username.trim(), password);
      loading = false;

      if (success) {
        goto('/');
      } else {
        errorMsg = 'Invalid username or password';
        errorVisible = true;
      }
    }}>
      <div class="mb-4">
        <label for="username" class="block text-sm font-medium text-slate-300 mb-2">Username</label>
        <input 
          id="username" 
          type="text" 
          placeholder="Enter username" 
          class="input"
          bind:value={username}
          disabled={loading}
        />
      </div>

      <div class="mb-4">
        <label for="password" class="block text-sm font-medium text-slate-300 mb-2">Password</label>
        <input 
          id="password" 
          type="password" 
          placeholder="Enter password" 
          class="input"
          bind:value={password}
          disabled={loading}
        />
      </div>

      {#if errorVisible}
        <div class="mb-4 p-3 bg-red-500/10 border border-red-500/20 text-red-300 rounded-lg text-sm text-center">
          {errorMsg}
        </div>
      {/if}

      <button 
        type="submit" 
        class="btn btn-success w-full"
        disabled={loading}
      >
        {loading ? 'Logging in...' : 'Login'}
      </button>
    </form>
  </div>
</div>

