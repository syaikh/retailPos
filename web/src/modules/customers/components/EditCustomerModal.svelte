<script lang="ts">
  import { Button, Input, Modal } from '$shared/ui';
  import { Save, Loader2, X } from 'lucide-svelte';

  let {
    open = $bindable(false),
    customer = $bindable(null as any),
    saving = $bindable(false),
    groups = [] as { id: number; name: string }[],
    onsave = (data: any) => {},
    oncancel = () => {},
  }: {
    open: boolean;
    customer: any;
    saving: boolean;
    groups?: { id: number; name: string }[];
    onsave?: (data: any) => void;
    oncancel?: () => void;
  } = $props();

  let name = $state('');
  let phone = $state('');
  let email = $state('');
  let address = $state('');
  let note = $state('');
  let groupId = $state<number | null>(null);
  let isActive = $state(true);
  let fieldErrors = $state({ name: '', phone: '', email: '' });

  let origName = $state('');
  let origPhone = $state('');
  let origEmail = $state('');
  let origAddress = $state<string | undefined>();
  let origNote = $state<string | undefined>();
  let origGroupId = $state<number | null>(null);

  $effect(() => {
    if (open && customer) {
      name = customer.name || '';
      phone = customer.phone || '';
      email = customer.email || '';
      address = customer.address || '';
      note = customer.note || '';
      groupId = customer.customer_group_id ?? null;
      isActive = customer.is_active !== false;
      origName = customer.name || '';
      origPhone = customer.phone || '';
      origEmail = customer.email || '';
      origAddress = customer.address;
      origNote = customer.note;
      origGroupId = customer.customer_group_id ?? null;
      fieldErrors = { name: '', phone: '', email: '' };
    }
  });

  function validateEmail(v: string): boolean {
    if (!v) return true;
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v);
  }

  function validatePhone(v: string): boolean {
    if (!v) return true;
    return /^[0-9+\-() ]{7,20}$/.test(v);
  }

  function handleSave() {
    const errors = { name: '', phone: '', email: '' };
    let valid = true;

    if (!name.trim()) {
      errors.name = 'Name is required';
      valid = false;
    } else if (name.trim().length > 200) {
      errors.name = 'Name must be at most 200 characters';
      valid = false;
    }

    if (!phone.trim()) {
      errors.phone = 'Phone is required';
      valid = false;
    } else if (!validatePhone(phone.trim())) {
      errors.phone = 'Invalid phone format';
      valid = false;
    }

    if (!email.trim()) {
      errors.email = 'Email is required';
      valid = false;
    } else if (!validateEmail(email.trim())) {
      errors.email = 'Invalid email format';
      valid = false;
    }

    fieldErrors = errors;
    if (!valid) return;

    const payload: Record<string, any> = {
      id: customer.id,
      name: name.trim(),
      phone: phone.trim(),
      email: email.trim(),
      is_active: isActive,
    };
    if (address.trim() !== (origAddress ?? '')) payload.address = address.trim();
    if (note.trim() !== (origNote ?? '')) payload.note = note.trim();
    if (groupId !== origGroupId) payload.customer_group_id = groupId;
    onsave(payload);
  }

  function handleCancel() {
    fieldErrors = { name: '', phone: '', email: '' };
    oncancel();
  }
</script>

<Modal bind:open={open} title="Edit Customer" size="md">
  <div class="space-y-4">
    <div class="space-y-1">
      <label for="edit-name" class="text-xs font-semibold text-text-secondary">Name <span class="text-danger">*</span></label>
      <Input
        id="edit-name"
        class={fieldErrors.name ? 'border-danger' : ''}
        placeholder="e.g. John Doe"
        bind:value={name}
      />
      {#if fieldErrors.name}
        <p class="text-danger text-xs mt-0.5">{fieldErrors.name}</p>
      {/if}
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div class="space-y-1">
        <label for="edit-phone" class="text-xs font-semibold text-text-secondary">Phone <span class="text-danger">*</span></label>
        <Input
          id="edit-phone"
          class={fieldErrors.phone ? 'border-danger' : ''}
          placeholder="e.g. 08123456789"
          bind:value={phone}
        />
        {#if fieldErrors.phone}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.phone}</p>
        {/if}
      </div>
      <div class="space-y-1">
        <label for="edit-email" class="text-xs font-semibold text-text-secondary">Email <span class="text-danger">*</span></label>
        <Input
          id="edit-email"
          class={fieldErrors.email ? 'border-danger' : ''}
          placeholder="e.g. john@example.com"
          bind:value={email}
        />
        {#if fieldErrors.email}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.email}</p>
        {/if}
      </div>
    </div>
    <div class="space-y-1">
      <label for="edit-address" class="text-xs font-semibold text-text-secondary">Address</label>
      <Input
        id="edit-address"
        placeholder="e.g. 123 Main St"
        bind:value={address}
      />
    </div>
    <div class="space-y-1">
      <label for="edit-group" class="text-xs font-semibold text-text-secondary">Customer Group</label>
      <select
        id="edit-group"
        class="w-full rounded-xl border border-border-default bg-bg-secondary px-3.5 py-2.5 text-sm text-text-primary focus:border-primary-default focus:outline-none focus:ring-2 focus:ring-primary-default/20 transition-colors duration-200"
        bind:value={groupId}
      >
        <option value={null}>— No Group —</option>
        {#each groups as g}
          <option value={g.id}>{g.name}</option>
        {/each}
      </select>
    </div>
    <div class="space-y-1">
      <label for="edit-note" class="text-xs font-semibold text-text-secondary">Note</label>
      <Input
        tag="textarea"
        id="edit-note"
        class="min-h-[60px] resize-none"
        placeholder="Optional notes about this customer"
        bind:value={note}
      />
    </div>
    <label class="flex items-center gap-2 text-sm">
      <input type="checkbox" bind:checked={isActive} />
      <span class="text-text-secondary font-medium">Active</span>
    </label>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={handleCancel}>Cancel</Button>
    <Button variant="primary" class="px-5" disabled={saving} onclick={handleSave}>
      {#if saving}
        <Loader2 size={14} class="animate-spin mr-1" /> Saving...
      {:else}
        <Save size={14} class="mr-1" /> Save Changes
      {/if}
    </Button>
  {/snippet}
</Modal>
