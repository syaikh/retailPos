<script lang="ts">
  import { Button, Input, Modal } from '$shared/ui';
  import { UserPlus, Loader2 } from 'lucide-svelte';

  let {
    open = $bindable(false),
    formName = $bindable(''),
    formPhone = $bindable(''),
    formEmail = $bindable(''),
    formAddress = $bindable(''),
    formNote = $bindable(''),
    formGroupId = $bindable(null as number | null),
    fieldErrors = $bindable({ name: '', phone: '', email: '', address: '', note: '' }),
    creating = $bindable(false),
    groups = [] as { id: number; name: string }[],
    oncreate = () => {},
  }: {
    open: boolean;
    formName?: string;
    formPhone?: string;
    formEmail?: string;
    formAddress?: string;
    formNote?: string;
    formGroupId?: number | null;
    fieldErrors?: { name: string; phone: string; email: string; address: string; note: string };
    creating?: boolean;
    groups?: { id: number; name: string }[];
    oncreate?: () => void;
  } = $props();
</script>

<Modal bind:open={open} title="Add Customer" size="md">
  <div class="space-y-4">
    <div class="space-y-1">
      <label for="customer-name" class="text-xs font-semibold text-text-secondary">Name <span class="text-danger">*</span></label>
      <Input
        id="customer-name"
        class={fieldErrors.name ? 'border-danger' : ''}
        placeholder="e.g. John Doe"
        bind:value={formName}
      />
      {#if fieldErrors.name}
        <p class="text-danger text-xs mt-0.5">{fieldErrors.name}</p>
      {/if}
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div class="space-y-1">
        <label for="customer-phone" class="text-xs font-semibold text-text-secondary">Phone <span class="text-danger">*</span></label>
        <Input
          id="customer-phone"
          class={fieldErrors.phone ? 'border-danger' : ''}
          placeholder="e.g. 08123456789"
          bind:value={formPhone}
        />
        {#if fieldErrors.phone}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.phone}</p>
        {/if}
      </div>
      <div class="space-y-1">
        <label for="customer-email" class="text-xs font-semibold text-text-secondary">Email <span class="text-danger">*</span></label>
        <Input
          id="customer-email"
          class={fieldErrors.email ? 'border-danger' : ''}
          placeholder="e.g. john@example.com"
          bind:value={formEmail}
        />
        {#if fieldErrors.email}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.email}</p>
        {/if}
      </div>
    </div>
    <div class="space-y-1">
      <label for="customer-address" class="text-xs font-semibold text-text-secondary">Address</label>
      <Input
        id="customer-address"
        placeholder="e.g. 123 Main St"
        bind:value={formAddress}
      />
    </div>
    <div class="space-y-1">
      <label for="customer-group" class="text-xs font-semibold text-text-secondary">Customer Group</label>
      <select
        id="customer-group"
        class="w-full rounded-xl border border-border-default bg-bg-secondary px-3.5 py-2.5 text-sm text-text-primary focus:border-primary-default focus:outline-none focus:ring-2 focus:ring-primary-default/20 transition-colors duration-200"
        bind:value={formGroupId}
      >
        <option value={null}>— No Group —</option>
        {#each groups as g}
          <option value={g.id}>{g.name}</option>
        {/each}
      </select>
    </div>
    <div class="space-y-1">
      <label for="customer-note" class="text-xs font-semibold text-text-secondary">Note</label>
      <Input
        tag="textarea"
        id="customer-note"
        class="min-h-[60px] resize-none"
        placeholder="Optional notes about this customer"
        bind:value={formNote}
      />
    </div>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={() => open = false}>Cancel</Button>
    <Button variant="primary" class="px-5" disabled={creating} onclick={oncreate}>
      {#if creating}
        <Loader2 size={14} class="animate-spin mr-1" /> Creating...
      {:else}
        <UserPlus size={14} class="mr-1" /> Create Customer
      {/if}
    </Button>
  {/snippet}
</Modal>
