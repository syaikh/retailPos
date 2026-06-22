<script>
  import Button from '$lib/components/ui/Button.svelte';
  import { auth } from '$lib/stores/auth';
  import { logout } from '$lib/api/auth';
  import { 
    Bell, 
    User,
    ChevronDown
  } from 'lucide-svelte';
  import { goto } from '$lib/router';

  let username = $derived($auth.user?.username || 'Guest');
  let role = $derived($auth.user?.role || 'cashier');

  async function handleLogout() {
    await logout();
    await goto('/login');
  }
</script>

<header class="navbar min-h-18 px-8 flex items-center justify-between border-b border-slate-700 bg-slate-900/80 backdrop-blur-sm">
  <div class="search-context">
    <span class="placeholder"></span>
  </div>

    <div class="user-profile flex items-center gap-4">
      <button type="button" class="notif-btn relative bg-transparent text-slate-400 hover:text-white p-2 rounded-lg transition-colors" aria-label="Notifications">
        <Bell size={20} />
        <span class="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full border-2 border-slate-800"></span>
      </button>

      <div class="divider w-px h-8 bg-slate-700"></div>

      <div class="profile-info flex items-center gap-3 cursor-pointer rounded-lg px-3 py-2 hover:bg-slate-700/50 transition-colors">
        <div class="avatar w-9 h-9 rounded-full bg-blue-500 flex items-center justify-center text-white">
          <User size={20} />
        </div>
        <div class="text hidden sm:block">
          <span class="name text-sm font-semibold text-white">{username}</span>
          <span class="role block text-xs text-slate-400 capitalize">{role}</span>
        </div>
        <ChevronDown size={16} class="chevron text-slate-400" />
      </div>

      <Button variant="ghost" size="sm" onclick={handleLogout} class="ml-3">
        Logout
      </Button>
    </div>
</header>
