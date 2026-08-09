<script lang="ts">
  import { Button, Input, Modal } from '$shared/ui';
  import { Save, Loader2, X } from 'lucide-svelte';
  import { labels } from '$shared/i18n';

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
      errors.name = labels.errorNameRequired;
      valid = false;
    } else if (name.trim().length > 200) {
      errors.name = labels.errorNameMaxLength;
      valid = false;
    }

    if (!phone.trim()) {
      errors.phone = labels.errorPhoneRequired;
      valid = false;
    } else if (!validatePhone(phone.trim())) {
      errors.phone = labels.errorPhoneInvalid;
      valid = false;
    }

    if (!email.trim()) {
      errors.email = labels.errorEmailRequired;
      valid = false;
    } else if (!validateEmail(email.trim())) {
      errors.email = labels.errorEmailInvalid;
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

<Modal bind:open={open} title={labels.editCustomer} size="md">
  <div class="space-y-4">
    <div class="space-y-1">
      <label for="edit-name" class="text-xs font-semibold text-text-secondary">{labels.name} <span class="text-danger">*</span></label>
      <Input
        id="edit-name"
        class={fieldErrors.name ? 'border-danger' : ''}
        placeholder={labels.egName}
        bind:value={name}
      />
      {#if fieldErrors.name}
        <p class="text-danger text-xs mt-0.5">{fieldErrors.name}</p>
      {/if}
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div class="space-y-1">
        <label for="edit-phone" class="text-xs font-semibold text-text-secondary">{labels.phone} <span class="text-danger">*</span></label>
        <Input
          id="edit-phone"
          class={fieldErrors.phone ? 'border-danger' : ''}
          placeholder={labels.egPhone}
          bind:value={phone}
        />
        {#if fieldErrors.phone}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.phone}</p>
        {/if}
      </div>
      <div class="space-y-1">
        <label for="edit-email" class="text-xs font-semibold text-text-secondary">{labels.email} <span class="text-danger">*</span></label>
        <Input
          id="edit-email"
          class={fieldErrors.email ? 'border-danger' : ''}
          placeholder={labels.egEmail}
          bind:value={email}
        />
        {#if fieldErrors.email}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.email}</p>
        {/if}
      </div>
    </div>
    <div class="space-y-1">
      <label for="edit-address" class="text-xs font-semibold text-text-secondary">{labels.address}</label>
      <Input
        id="edit-address"
        placeholder={labels.egAddress}
        bind:value={address}
      />
    </div>
    <div class="space-y-1">
      <label for="edit-group" class="text-xs font-semibold text-text-secondary">{labels.customerGroup}</label>
      <select
        id="edit-group"
        class="w-full rounded-xl border border-border-default bg-bg-secondary px-3.5 py-2.5 text-sm text-text-primary focus:border-primary-default focus:outline-none focus:ring-2 focus:ring-primary-default/20 transition-colors duration-200"
        bind:value={groupId}
      >
        <option value={null}>{labels.noGroup}</option>
        {#each groups as g}
          <option value={g.id}>{g.name}</option>
        {/each}
      </select>
    </div>
    <div class="space-y-1">
      <label for="edit-note" class="text-xs font-semibold text-text-secondary">{labels.note}</label>
      <Input
        tag="textarea"
        id="edit-note"
        class="min-h-[60px] resize-none"
        placeholder={labels.optionalNotesCustomer}
        bind:value={note}
      />
    </div>
    <label class="flex items-center gap-2 text-sm">
      <input type="checkbox" bind:checked={isActive} />
      <span class="text-text-secondary font-medium">{labels.active}</span>
    </label>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={handleCancel}>{labels.cancel}</Button>
    <Button variant="primary" class="px-5" disabled={saving} onclick={handleSave}>
      {#if saving}
        <Loader2 size={14} class="animate-spin mr-1" /> {labels.saving}
      {:else}
        <Save size={14} class="mr-1" /> {labels.simpanPerubahan}
      {/if}
    </Button>
  {/snippet}
</Modal>
