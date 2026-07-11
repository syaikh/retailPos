<script lang="ts">
  import { Badge, Button, Skeleton, SortableHeader } from '$shared/ui';
  import { User, Pencil, Trash2, Users, Search } from 'lucide-svelte';
  import { formatDateInJakarta, formatTimeInJakarta } from '$shared/utils/jakartaTime';

  let {
    users = [],
    loading = false,
    searchQuery = '',
    canEdit = false,
    canDelete = false,
    canEditSuperadmin = false,
    currentUserID = 0,
    sortBy = $bindable('username'),
    sortDir = $bindable('asc'),
    onsort = (key: string) => {},
    onedit = (user: any) => {},
    ondelete = (user: any) => {},
  }: {
    users: any[];
    loading: boolean;
    searchQuery: string;
    canEdit: boolean;
    canDelete: boolean;
    canEditSuperadmin: boolean;
    currentUserID: number;
    sortBy: string;
    sortDir: string;
    onsort?: (key: string) => void;
    onedit?: (user: any) => void;
    ondelete?: (user: any) => void;
  } = $props();

  function roleVariant(r: any): string {
    const roleName = typeof r === 'object' ? r.name : r;
    if (roleName === 'superadmin') return 'primary';
    if (roleName === 'admin') return 'warning';
    return 'muted';
  }

  function roleName(user: any): string {
    return user.role?.name || (user.role_id === 1 ? 'superadmin' : user.role_id === 2 ? 'admin' : user.role_id === 3 ? 'cashier' : user.role_id === 4 ? 'manager' : user.role_id === 5 ? 'staff' : 'unknown');
  }
</script>

<div class="overflow-x-auto" style="min-width: 0;">
  <table class="w-full table-fixed border-collapse" style="min-width: 680px;">
    <thead class="bg-muted/50">
      <tr>
        <th class="text-left p-4 font-semibold" style="width: 30%;">
          <SortableHeader label="USER" column="username" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="text-left p-4 font-semibold w-40">
          <SortableHeader label="ROLE" column="role_id" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="text-left p-4 font-semibold w-28">STATUS</th>
        <th class="text-left p-4 font-semibold w-44">
          <SortableHeader label="LAST LOGIN" column="last_login" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="text-center p-4 font-semibold w-20">ACTIONS</th>
      </tr>
    </thead>
    <tbody>
      {#if loading}
        <div aria-busy="true" aria-label="Loading users">
        {#each { length: 5 } as _}
          <tr class="border-t border-border">
            <td class="p-4">
              <div class="flex items-center gap-3">
                <Skeleton width="w-8" height="h-8" rounded="rounded-full" />
                <div class="flex flex-col gap-1.5">
                  <Skeleton width="w-32" height="h-3.5" />
                  <Skeleton width="w-44" height="h-3" />
                </div>
              </div>
            </td>
            <td class="p-4">
              <Skeleton width="w-16" height="h-6" rounded="rounded-full" />
            </td>
            <td class="p-4">
              <div class="flex items-center gap-2">
                <Skeleton width="w-1.5" height="h-1.5" rounded="rounded-full" />
                <Skeleton width="w-12" height="h-3.5" />
              </div>
            </td>
            <td class="p-4">
              <Skeleton width="w-36" height="h-3.5" />
            </td>
            <td class="p-4">
              <div class="flex items-center justify-center gap-2">
                <Skeleton width="w-8" height="h-8" rounded="rounded-xl" />
                <Skeleton width="w-8" height="h-8" rounded="rounded-xl" />
              </div>
            </td>
          </tr>
        {/each}
        </div>
      {:else if users.length === 0}
        <tr>
          <td colspan="5" class="px-4 py-12 text-center" role="status">
            <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
              <Users size={32} class="text-text-muted" />
            </div>
            <p class="text-text-primary font-semibold mt-4">No users found</p>
            <p class="text-text-muted text-sm mt-1">
              {searchQuery ? `No match for "${searchQuery}"` : 'Start by adding a user'}
            </p>
          </td>
        </tr>
      {:else}
        {#each users as user (user.id)}
          <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
            <td class="p-4 min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">
              <div class="flex items-center gap-3">
                <div class="w-8 h-8 rounded-full gradient-bg-primary flex items-center justify-center shrink-0">
                  <User size={14} class="text-white" />
                </div>
                <div>
                  <p class="font-medium text-text-primary">{user.username}</p>
                  <p class="text-xs text-text-muted">{user.email || '—'}</p>
                </div>
              </div>
            </td>
            <td class="p-4">
              <Badge variant={roleVariant(user.role)}>{roleName(user)}</Badge>
            </td>
            <td class="p-4">
              <div class="flex items-center gap-2">
                <span class="w-1.5 h-1.5 rounded-full {user.is_active !== false ? 'bg-success animate-pulse-dot' : 'bg-text-muted'}"></span>
                <span class="text-sm text-text-secondary">{user.is_active !== false ? 'Active' : 'Inactive'}</span>
              </div>
            </td>
            <td class="p-4 text-text-muted text-sm leading-relaxed">
              {#if user.last_login}
                <span class="block">{formatDateInJakarta(user.last_login)}</span>
                <span class="block text-[10px] text-text-muted">{formatTimeInJakarta(user.last_login)}</span>
              {:else}
                Never
              {/if}
            </td>
            <td class="p-4 text-center">
              <div class="flex items-center justify-center gap-2">
                <Button
                  variant="ghost"
                  size="icon"
                  class="text-text-muted hover:text-primary-light"
                  title="Edit"
                  aria-label="Edit"
                  onclick={() => onedit(user)}
                  disabled={user.role_id === 1 && !canEditSuperadmin}
                >
                  <Pencil size={14} />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  class="text-text-muted hover:text-danger hover:bg-danger-subtle"
                  onclick={() => ondelete(user)}
                  title="Delete"
                  aria-label="Delete"
                  disabled={user.id === currentUserID || user.role_id === 1}
                >
                  <Trash2 size={14} />
                </Button>
              </div>
            </td>
          </tr>
        {/each}
      {/if}
    </tbody>
  </table>
</div>
