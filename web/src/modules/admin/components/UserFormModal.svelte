<script lang="ts">
  import { Button, Input, Modal } from '$shared/ui';
  import { User, Shield, ChevronDown, Loader2 } from 'lucide-svelte';

  let {
    open = $bindable(false),
    modalMode = 'add',
    form = $bindable({ username: '', email: '', password: '', role_id: 0, is_active: true }),
    roles = [],
    saving = $bindable(false),
    usernameHasInvalidChars = false,
    onsave = () => {},
  }: {
    open: boolean;
    modalMode?: string;
    form?: { username: string; email: string; password: string; role_id: number; is_active: boolean };
    roles?: any[];
    saving?: boolean;
    usernameHasInvalidChars?: boolean;
    onsave?: () => void;
  } = $props();

  let showFormRoleDropdown = $state(false);
  let dropdownStyle = $state('');

  let selectedRoleName = $derived(roles.find((r: any) => r.id === form.role_id)?.name || 'Select Role');

  function toggleFormRoleDropdown(e: any) {
    if (showFormRoleDropdown) {
      showFormRoleDropdown = false;
      dropdownStyle = '';
      return;
    }
    const btn = e.target.closest('button');
    if (btn) {
      const rect = btn.getBoundingClientRect();
      dropdownStyle = `position:fixed; top:${rect.bottom + 4}px; left:${rect.left}px; width:${rect.width}px;`;
    }
    showFormRoleDropdown = true;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (showFormRoleDropdown && e.key === 'Escape') {
      showFormRoleDropdown = false;
      dropdownStyle = '';
      e.stopPropagation();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<Modal bind:open={open} title={modalMode === 'add' ? 'Add New User' : 'Edit User'} size="md">
  <form onsubmit={(e) => { e.preventDefault(); onsave(); }} class="space-y-6">
    <div class="space-y-4">
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="usr-username" class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
            <User size={14} class="text-text-muted" />
            Username
          </label>
          <Input id="usr-username" type="text" placeholder="johndoe" bind:value={form.username} required minlength="3" maxlength="50" pattern="[a-zA-Z0-9]+" title="3-50 alphanumeric characters only (will be converted to lowercase)" />
        </div>
        <div>
          <label for="usr-email" class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
            Email Address
          </label>
          <Input id="usr-email" type="email" placeholder="john@example.com" bind:value={form.email} required />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div class="relative form-role-dropdown-container">
          <label class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
            <Shield size={14} class="text-text-muted" />
            Role
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
          <label class="flex items-center gap-3 cursor-pointer select-none group">
            <div class="relative">
              <input type="checkbox" class="sr-only peer" bind:checked={form.is_active} />
              <div class="w-10 h-5 bg-surface-default border border-border rounded-full peer peer-checked:bg-primary-subtle peer-checked:border-primary/50 transition-colors"></div>
              <div class="absolute left-1 top-1 w-3 h-3 bg-text-muted rounded-full peer-checked:translate-x-5 peer-checked:bg-primary-light transition-transform shadow-sm"></div>
            </div>
            <span class="text-sm font-medium text-text-secondary group-hover:text-text-primary transition-colors">Active Account</span>
          </label>
        </div>
      </div>

      <div class="pt-2">
        <label for="usr-password" class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-text-muted"><rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          {modalMode === 'add' ? 'Password' : 'New Password (optional)'}
        </label>
        <Input id="usr-password" type="password" placeholder="••••••••" bind:value={form.password} required={modalMode === 'add'} minlength="6" />
        {#if modalMode === 'edit'}
          <p class="text-xs text-text-muted mt-1.5">Leave blank to keep current password</p>
        {/if}
      </div>
    </div>
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => open = false} disabled={saving}>Cancel</Button>
    <Button variant="primary" class="min-w-32" onclick={onsave} disabled={saving || (modalMode === 'add' && usernameHasInvalidChars)}>
      {#if saving}
        <Loader2 size={16} class="animate-spin" /> Saving...
      {:else}
        {modalMode === 'add' ? 'Create User' : 'Save Changes'}
      {/if}
    </Button>
  {/snippet}
</Modal>
