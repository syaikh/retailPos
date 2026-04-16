<script>
  import { user } from '$lib/stores.js';
  import { 
    Bell, 
    User,
    ChevronDown
  } from 'lucide-svelte';

  let username = $derived($user?.username || 'Guest');
  let role = $derived($user?.role || 'cashier');
</script>

<header class="navbar glass">
  <div class="search-context">
    <!-- Context-aware search could go here -->
  </div>

  <div class="user-profile">
    <button class="notif-btn">
      <Bell size={20} />
      <span class="badge"></span>
    </button>

    <div class="divider"></div>

    <div class="profile-info">
      <div class="avatar">
        <User size={20} />
      </div>
      <div class="text">
        <span class="name">{username}</span>
        <span class="role">{role}</span>
      </div>
      <ChevronDown size={16} class="chevron" />
    </div>
  </div>
</header>

<style>
  .navbar {
    height: 72px;
    padding: 0 32px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid var(--border);
    background: rgba(15, 23, 42, 0.8) !important;
  }

  .user-profile {
    display: flex;
    align-items: center;
    gap: 20px;
  }

  .notif-btn {
    background: transparent;
    color: var(--text-secondary);
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .notif-btn:hover {
    color: var(--text-primary);
  }

  .badge {
    position: absolute;
    top: -2px;
    right: -2px;
    width: 8px;
    height: 8px;
    background: var(--danger);
    border-radius: 50%;
    border: 2px solid var(--bg-surface);
  }

  .divider {
    width: 1px;
    height: 32px;
    background: var(--border);
  }

  .profile-info {
    display: flex;
    align-items: center;
    gap: 12px;
    cursor: pointer;
    padding: 6px 12px;
    border-radius: 99px;
    transition: background 0.2s;
  }

  .profile-info:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  .avatar {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    background: var(--primary);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
  }

  .text {
    display: flex;
    flex-direction: column;
  }

  .name {
    font-size: 0.875rem;
    font-weight: 600;
  }

  .role {
    font-size: 0.75rem;
    color: var(--text-secondary);
    text-transform: capitalize;
  }
</style>
