<script lang="ts">
  import { Modal, Button, Input } from '$shared/ui';
  import { labels } from '$shared/i18n';
  import { Loader2 } from 'lucide-svelte';
  import type { Supplier } from '../types';

  let {
    open = $bindable(false),
    mode = 'add',
    supplier = null,
    saving = false,
    onsave = () => {},
    oncancel = () => {},
  }: {
    open?: boolean;
    mode?: 'add' | 'edit';
    supplier?: Supplier | null;
    saving?: boolean;
    onsave?: (data: any) => void;
    oncancel?: () => void;
  } = $props();

  let form = $state({
    name: '',
    code: '',
    contact_name: '',
    phone: '',
    email: '',
    address: '',
    notes: '',
    is_active: true,
    is_consignment: false
  });

  $effect(() => {
    if (open) {
      if (mode === 'edit' && supplier) {
        form = {
          name: supplier.name,
          code: supplier.code,
          contact_name: supplier.contact_name || '',
          phone: supplier.phone || '',
          email: supplier.email || '',
          address: supplier.address || '',
          notes: supplier.notes || '',
          is_active: supplier.is_active,
          is_consignment: supplier.is_consignment || false
        };
      } else {
        form = { name: '', code: '', contact_name: '', phone: '', email: '', address: '', notes: '', is_active: true, is_consignment: false };
      }
    }
  });

  function handleSubmit(e: Event) {
    e.preventDefault();
    if (!form.name || !form.code) return;
    onsave(form);
  }
</script>

<Modal bind:open title={mode === 'add' ? labels.addSupplier : labels.editSupplier} size="md">
  <form onsubmit={handleSubmit} class="space-y-3">
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="sup_name" class="block text-xs font-medium text-text-secondary mb-1">{labels.supplierName} <span class="text-danger">*</span></label>
        <Input id="sup_name" bind:value={form.name} required placeholder={labels.contohNamaSupplier} class="h-9 text-sm" />
      </div>
      <div>
        <label for="sup_code" class="block text-xs font-medium text-text-secondary mb-1">{labels.supplierCode} <span class="text-danger">*</span></label>
        <Input id="sup_code" bind:value={form.code} required placeholder="SUP-001" class="h-9 text-sm" />
      </div>
    </div>
    <div class="grid grid-cols-3 gap-3">
      <div>
        <label for="sup_contact" class="block text-xs font-medium text-text-secondary mb-1">{labels.contactPerson}</label>
        <Input id="sup_contact" bind:value={form.contact_name} placeholder="Budi Santoso" class="h-9 text-sm" />
      </div>
      <div>
        <label for="sup_phone" class="block text-xs font-medium text-text-secondary mb-1">{labels.phone}</label>
        <Input id="sup_phone" bind:value={form.phone} placeholder="021-12345678" class="h-9 text-sm" />
      </div>
      <div>
        <label for="sup_email" class="block text-xs font-medium text-text-secondary mb-1">{labels.email}</label>
        <Input id="sup_email" type="email" bind:value={form.email} placeholder="info@supplier.co.id" class="h-9 text-sm" />
      </div>
    </div>
    <div>
      <label for="sup_address" class="block text-xs font-medium text-text-secondary mb-1">{labels.address}</label>
      <Input id="sup_address" tag="textarea" bind:value={form.address} placeholder={labels.fullAddress} class="min-h-[40px] resize-y text-sm" />
    </div>
    <div>
      <label for="sup_notes" class="block text-xs font-medium text-text-secondary mb-1">{labels.notes}</label>
      <Input id="sup_notes" tag="textarea" bind:value={form.notes} placeholder={labels.additionalNotes} class="min-h-[40px] resize-y text-sm" />
    </div>
    {#if mode === 'edit'}
      <div class="flex items-center gap-2">
        <input type="checkbox" bind:checked={form.is_active} id="is_active" class="rounded" />
        <label for="is_active" class="text-sm text-text-secondary">{labels.active}</label>
      </div>
    {/if}
    <div class="flex items-center gap-2 border-t border-border/50 pt-3">
      <input type="checkbox" bind:checked={form.is_consignment} id="is_consignment" class="rounded" />
      <label for="is_consignment" class="text-sm text-text-secondary">Supplier Konsinyasi</label>
    </div>
  </form>
  {#snippet footer()}
    <Button variant="secondary" onclick={oncancel} disabled={saving}>{labels.cancel}</Button>
    <Button variant="primary" class="min-w-32" onclick={handleSubmit} disabled={saving}>
      {#if saving}<Loader2 class="w-4 h-4 mr-2 animate-spin" />{/if}
      {mode === 'add' ? labels.create : labels.update}
    </Button>
  {/snippet}
</Modal>