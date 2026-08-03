<script lang="ts">
  import { Button, Input, Modal, SelectSearch } from '$shared/ui';
  import { Plus, Loader2 } from 'lucide-svelte';

  let {
    open = $bindable(false),
    creating = $bindable(false),
    warehouseOptions = [] as { value: number; label: string }[],
    storeOptions = [] as { value: number; label: string }[],
    oncreate = (data: { code: string; name: string; warehouse_id?: number | null; store_id?: number | null; notes?: string }) => {},
  }: {
    open: boolean;
    creating?: boolean;
    warehouseOptions?: { value: number; label: string }[];
    storeOptions?: { value: number; label: string }[];
    oncreate?: (data: { code: string; name: string; warehouse_id?: number | null; store_id?: number | null; notes?: string }) => void;
  } = $props();

  let code = $state('');
  let name = $state('');
  let scopeType = $state<'warehouse' | 'store'>('warehouse');
  let warehouseId = $state<number | undefined>(undefined);
  let storeId = $state<number | undefined>(undefined);
  let notes = $state('');
  let fieldErrors = $state({ code: '', name: '', scope: '' });

  $effect(() => {
    if (open) {
      code = '';
      name = '';
      scopeType = 'warehouse';
      warehouseId = undefined;
      storeId = undefined;
      notes = '';
      fieldErrors = { code: '', name: '', scope: '' };
    }
  });

  function handleCreate() {
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

    oncreate({
      code: code.trim(),
      name: name.trim(),
      warehouse_id: scopeType === 'warehouse' ? warehouseId ?? null : null,
      store_id: scopeType === 'store' ? storeId ?? null : null,
      notes: notes.trim() || undefined,
    });
  }
</script>

<Modal bind:open={open} title="Tambah Lokasi Penyimpanan" size="md">
  <div class="space-y-4">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div class="space-y-1">
        <label for="sl-code" class="text-xs font-semibold text-text-secondary">Kode <span class="text-danger">*</span></label>
        <Input
          id="sl-code"
          class={fieldErrors.code ? 'border-danger' : ''}
          placeholder="e.g. RAK-A-01"
          bind:value={code}
        />
        {#if fieldErrors.code}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.code}</p>
        {/if}
      </div>
      <div class="space-y-1">
        <label for="sl-name" class="text-xs font-semibold text-text-secondary">Nama <span class="text-danger">*</span></label>
        <Input
          id="sl-name"
          class={fieldErrors.name ? 'border-danger' : ''}
          placeholder="e.g. Rak A - Baris 1"
          bind:value={name}
        />
        {#if fieldErrors.name}
          <p class="text-danger text-xs mt-0.5">{fieldErrors.name}</p>
        {/if}
      </div>
    </div>

    <div class="space-y-1" role="group" aria-labelledby="sl-scope-label">
      <span id="sl-scope-label" class="text-xs font-semibold text-text-secondary">Lingkup <span class="text-danger">*</span></span>
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
      <label for="sl-notes" class="text-xs font-semibold text-text-secondary">Catatan</label>
      <Input
        tag="textarea"
        id="sl-notes"
        class="min-h-[60px] resize-none"
        placeholder="Catatan opsional untuk lokasi ini"
        bind:value={notes}
      />
    </div>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" onclick={() => open = false}>Batal</Button>
    <Button variant="primary" class="px-5" disabled={creating} onclick={handleCreate}>
      {#if creating}
        <Loader2 size={14} class="animate-spin mr-1" /> Menyimpan...
      {:else}
        <Plus size={14} class="mr-1" /> Tambah Lokasi
      {/if}
    </Button>
  {/snippet}
</Modal>
