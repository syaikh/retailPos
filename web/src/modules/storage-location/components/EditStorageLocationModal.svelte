<script lang="ts">
  import { Button, Input, Modal, SelectSearch } from '$shared/ui';
  import { Save, Loader2 } from 'lucide-svelte';
  import type { StorageLocation } from '../types';

  let {
    open = $bindable(false),
    location = $bindable(null as StorageLocation | null),
    saving = $bindable(false),
    warehouseOptions = [] as { value: number; label: string }[],
    storeOptions = [] as { value: number; label: string }[],
    onsave = (data: any) => {},
    oncancel = () => {},
  }: {
    open: boolean;
    location: StorageLocation | null;
    saving: boolean;
    warehouseOptions?: { value: number; label: string }[];
    storeOptions?: { value: number; label: string }[];
    onsave?: (data: any) => void;
    oncancel?: () => void;
  } = $props();

  let code = $state('');
  let name = $state('');
  let isActive = $state(true);
  let scopeType = $state<'warehouse' | 'store'>('warehouse');
  let warehouseId = $state<number | undefined>(undefined);
  let storeId = $state<number | undefined>(undefined);
  let notes = $state('');
  let fieldErrors = $state({ code: '', name: '', scope: '' });

  let origCode = $state('');
  let origName = $state('');
  let origNotes = $state<string | undefined>();
  let origWarehouseId = $state<number | null>(null);
  let origStoreId = $state<number | null>(null);

  $effect(() => {
    if (open && location) {
      code = location.code || '';
      name = location.name || '';
      isActive = location.is_active !== false;
      notes = location.notes || '';
      origCode = location.code || '';
      origName = location.name || '';
      origNotes = location.notes;
      origWarehouseId = location.warehouse_id ?? null;
      origStoreId = location.store_id ?? null;
      scopeType = location.warehouse_id != null ? 'warehouse' : 'store';
      warehouseId = location.warehouse_id ?? undefined;
      storeId = location.store_id ?? undefined;
      fieldErrors = { code: '', name: '', scope: '' };
    }
  });

  function handleSave() {
    const errors = { code: '', name: '', scope: '' };
    let valid = true;

    if (!code.trim()) {
      errors.code = 'Kode wajib diisi';
      valid = false;
    } else if (code.trim().length > 50) {
      errors.code = 'Kode maksimal 50 karakter';
      valid = false;
    }

    if (!name.trim()) {
      errors.name = 'Nama wajib diisi';
      valid = false;
    } else if (name.trim().length > 100) {
      errors.name = 'Nama maksimal 100 karakter';
      valid = false;
    }

    const scopeValid = scopeType === 'warehouse' ? warehouseId != null : storeId != null;
    if (!scopeValid) {
      errors.scope = 'Pilih gudang atau toko';
      valid = false;
    }

    fieldErrors = errors;
    if (!valid) return;

    const payload: Record<string, any> = { id: location.id };
    if (code.trim() !== origCode) payload.code = code.trim();
    if (name.trim() !== origName) payload.name = name.trim();
    if ((notes.trim() || undefined) !== (origNotes || undefined)) payload.notes = notes.trim() || null;
    if (isActive !== location.is_active) payload.is_active = isActive;
    if (scopeType === 'warehouse') {
      if (warehouseId != null && warehouseId !== origWarehouseId) payload.warehouse_id = warehouseId;
    } else {
      if (storeId != null && storeId !== origStoreId) payload.store_id = storeId;
    }
    onsave(payload);
  }

  function handleCancel() {
    fieldErrors = { code: '', name: '', scope: '' };
    oncancel();
  }
</script>

<Modal bind:open={open} title="Edit Lokasi Penyimpanan" size="md">
  <div class="space-y-4">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div class="space-y-1">
        <label for="edit-sl-code" class="text-xs font-semibold text-text-secondary">Kode <span class="text-danger">*</span></label>
        <Input
          id="edit-sl-code"
          class={fieldErrors.code ? 'border-danger' : ''}
          placeholder="e.g. RAK-A-01"
          bind:value={code}
        />
        {#if fieldErrors.code}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.code}</p>
        {/if}
      </div>
      <div class="space-y-1">
        <label for="edit-sl-name" class="text-xs font-semibold text-text-secondary">Nama <span class="text-danger">*</span></label>
        <Input
          id="edit-sl-name"
          class={fieldErrors.name ? 'border-danger' : ''}
          placeholder="e.g. Rak A - Baris 1"
          bind:value={name}
        />
        {#if fieldErrors.name}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.name}</p>
        {/if}
      </div>
    </div>

    <div class="space-y-1" role="group" aria-labelledby="edit-sl-scope-label">
      <span id="edit-sl-scope-label" class="text-xs font-semibold text-text-secondary">Lingkup <span class="text-danger">*</span></span>
      <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border/30 w-fit" role="group" aria-label="Tipe lingkup">
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {scopeType === 'warehouse' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { scopeType = 'warehouse'; warehouseId = undefined; }}
          aria-pressed={scopeType === 'warehouse'}
        >Gudang</button>
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {scopeType === 'store' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { scopeType = 'store'; storeId = undefined; }}
          aria-pressed={scopeType === 'store'}
        >Toko</button>
      </div>
      <div class="mt-2">
        {#if scopeType === 'warehouse'}
          <SelectSearch
            bind:value={warehouseId}
            options={warehouseOptions}
            placeholder="Pilih gudang..."
            searchPlaceholder="Cari gudang..."
            disabled={warehouseOptions.length === 0}
            notFoundText="Tidak ada gudang ditemukan"
            onchange={() => { fieldErrors = { ...fieldErrors, scope: '' }; }}
          />
        {:else}
          <SelectSearch
            bind:value={storeId}
            options={storeOptions}
            placeholder="Pilih toko..."
            searchPlaceholder="Cari toko..."
            disabled={storeOptions.length === 0}
            notFoundText="Tidak ada toko ditemukan"
            onchange={() => { fieldErrors = { ...fieldErrors, scope: '' }; }}
          />
        {/if}
        {#if fieldErrors.scope}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.scope}</p>
        {/if}
      </div>
    </div>

    <div class="space-y-1">
      <label for="edit-sl-notes" class="text-xs font-semibold text-text-secondary">Catatan</label>
      <Input
        tag="textarea"
        id="edit-sl-notes"
        class="min-h-[60px] resize-none"
        placeholder="Catatan opsional untuk lokasi ini"
        bind:value={notes}
      />
    </div>
    <label class="flex items-center gap-2 text-sm">
      <input type="checkbox" bind:checked={isActive} />
      <span class="text-text-secondary font-medium">Aktif</span>
    </label>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={handleCancel}>Batal</Button>
    <Button variant="primary" class="px-5" disabled={saving} onclick={handleSave}>
      {#if saving}
        <Loader2 size={14} class="animate-spin mr-1" /> Menyimpan...
      {:else}
        <Save size={14} class="mr-1" /> Simpan Perubahan
      {/if}
    </Button>
  {/snippet}
</Modal>
