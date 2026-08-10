<script lang="ts">
  import { Button, Input, Modal, ToggleSwitch } from '$shared/ui';
  import { User, Shield, ChevronDown, Loader2, UserRoundCog } from 'lucide-svelte';
  import { getUsers } from '$modules/admin';
  import { labels } from '$shared/i18n';

  let {
    open = $bindable(false),
    modalMode = 'add',
    form = $bindable({ username: '', email: '', password: '', role_id: 0, is_active: true, reports_to: null }),
    roles = [],
    saving = $bindable(false),
    usernameHasInvalidChars = false,
    canAssignManager = true,
    onsave = () => {},
  }: {
    open: boolean;
    modalMode?: string;
    form?: { username: string; email: string; password: string; role_id: number; is_active: boolean; reports_to: number | null };
    roles?: any[];
    saving?: boolean;
    usernameHasInvalidChars?: boolean;
    canAssignManager?: boolean;
    onsave?: () => void;
  } = $props();

  let showFormRoleDropdown = $state(false);
  let showReportsToDropdown = $state(false);
  let dropdownStyle = $state('');
  let reportsToDropdownStyle = $state('');
  let allUsers = $state<any[]>([]);

  let selectedRoleName = $derived(roles.find((r: any) => r.id === form.role_id)?.name || labels.role);

  let selectedReportsToName = $derived(
    form.reports_to ? allUsers.find((u: any) => u.id === form.reports_to)?.username || labels.unknown : labels.none
  );

  function toggleFormRoleDropdown(e: any) {
    if (showFormRoleDropdown) {
      showFormRoleDropdown = false;
      dropdownStyle = '';
      return;
    }
    showReportsToDropdown = false;
    const btn = e.target.closest('button');
    if (btn) {
      const rect = btn.getBoundingClientRect();
      dropdownStyle = `position:fixed; top:${rect.bottom + 4}px; left:${rect.left}px; width:${rect.width}px;`;
    }
    showFormRoleDropdown = true;
  }

  function toggleReportsToDropdown(e: any) {
    if (showReportsToDropdown) {
      showReportsToDropdown = false;
      reportsToDropdownStyle = '';
      return;
    }
    showFormRoleDropdown = false;
    loadAllUsers();
    const btn = e.target.closest('button');
    if (btn) {
      const rect = btn.getBoundingClientRect();
      reportsToDropdownStyle = `position:fixed; top:${rect.bottom + 4}px; left:${rect.left}px; width:${rect.width}px;`;
    }
    showReportsToDropdown = true;
  }

  async function loadAllUsers() {
    if (allUsers.length > 0) return;
    try {
      const result = await getUsers({ limit: 1000, offset: 0 });
      allUsers = result.data;
    } catch {
      // silently fail
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (showFormRoleDropdown && e.key === 'Escape') {
      showFormRoleDropdown = false;
      dropdownStyle = '';
      e.stopPropagation();
    }
    if (showReportsToDropdown && e.key === 'Escape') {
      showReportsToDropdown = false;
      reportsToDropdownStyle = '';
      e.stopPropagation();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<Modal bind:open={open} title={modalMode === 'add' ? `${labels.add} ${labels.user}` : `${labels.edit} ${labels.user}`} size="md">
  <form onsubmit={(e) => { e.preventDefault(); onsave(); }} class="space-y-6">
    <div class="space-y-4">
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="usr-username" class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
            <User size={14} class="text-text-muted" />
            {labels.username}
          </label>
          <Input id="usr-username" type="text" placeholder="johndoe" bind:value={form.username} required minlength="3" maxlength="50" pattern="[a-zA-Z0-9]+" title="3-50 alphanumeric characters only (will be converted to lowercase)" />
        </div>
        <div>
          <label for="usr-email" class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
            {labels.email}
          </label>
          <Input id="usr-email" type="email" placeholder="john@example.com" bind:value={form.email} required />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div class="relative form-role-dropdown-container">
          <label class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
            <Shield size={14} class="text-text-muted" />
            {labels.role}
          </label>
          <div class="form-role-dropdown-container">
            <button
              type="button"
              class="flex items-center gap-2 px-3 h-10 w-full rounded-xl border border-border bg-surface-default text-sm hover:border-border-strong hover:bg-surface-hover transition-colors {form.role_id ? 'text-text-primary' : 'text-text-muted'}"
              onclick={toggleFormRoleDropdown}
            >
              <span class="flex-1 text-left truncate">{selectedRoleName}</span>
              <ChevronDown size={14} class="text-text-muted shrink-0" />
            </button>
            {#if showFormRoleDropdown}
              <div style={dropdownStyle} class="z-[100] bg-surface-default border border-border rounded-lg shadow-xl">
                <div class="p-1.5 max-h-64 overflow-y-auto">
                  <div class="grid grid-cols-2 gap-1">
                    {#each roles as role}
                      <button
                        type="button"
                        class="w-full text-left px-3 py-2 text-sm rounded-lg transition-colors truncate {form.role_id === role.id ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                        onclick={() => { form.role_id = role.id; showFormRoleDropdown = false; }}
                      >{role.name}</button>
                    {/each}
                  </div>
                </div>
              </div>
            {/if}
          </div>
        </div>
        <div class="flex items-end pb-2">
          <ToggleSwitch bind:checked={form.is_active} label={labels.active} />
        </div>
      </div>

      <div>
        <label class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
          <UserRoundCog size={14} class="text-text-muted" />
          {labels.reportsTo}
        </label>
        <div class="relative">
          <button
            type="button"
            class="flex items-center gap-2 px-3 h-10 w-full rounded-xl border border-border bg-surface-default text-sm hover:border-border-strong hover:bg-surface-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            onclick={toggleReportsToDropdown}
            disabled={!canAssignManager}
          >
            <span class="flex-1 text-left truncate">{selectedReportsToName}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
          {#if showReportsToDropdown}
            <div style={reportsToDropdownStyle} class="z-[100] bg-surface-default border border-border rounded-lg shadow-xl">
              <div class="p-1.5 max-h-64 overflow-y-auto">
                <button
                  type="button"
                  class="w-full text-left px-3 py-2 text-sm rounded-lg transition-colors truncate {!form.reports_to ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                  onclick={() => { form.reports_to = null; showReportsToDropdown = false; }}
                >{labels.none}</button>
                {#each allUsers as u}
                  <button
                    type="button"
                    class="w-full text-left px-3 py-2 text-sm rounded-lg transition-colors truncate {form.reports_to === u.id ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                    onclick={() => { form.reports_to = u.id; showReportsToDropdown = false; }}
                  >{u.username}</button>
                {/each}
              </div>
            </div>
          {/if}
        </div>
      </div>

      <div class="pt-2">
        <label for="usr-password" class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-text-muted"><rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          {labels.password}
        </label>
        <Input id="usr-password" type="password" placeholder="••••••••" bind:value={form.password} required={modalMode === 'add'} minlength="6" />
        {#if modalMode === 'edit'}
          <p class="text-xs text-text-muted mt-1.5">{labels.leaveBlankToKeepPassword}</p>
        {/if}
      </div>
    </div>
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => open = false} disabled={saving}>{labels.cancel}</Button>
    <Button variant="primary" class="min-w-32" onclick={onsave} disabled={saving || (modalMode === 'add' && usernameHasInvalidChars)}>
      {#if saving}
        <Loader2 size={16} class="animate-spin" /> {labels.saving}
      {:else}
        {modalMode === 'add' ? `${labels.create} ${labels.user}` : labels.save}
      {/if}
    </Button>
  {/snippet}
</Modal>
