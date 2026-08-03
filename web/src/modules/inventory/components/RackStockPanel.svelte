<script lang="ts">
  import { Button, Input, Modal, SelectSearch } from '$shared/ui';
  import { Loader2, PackageX } from 'lucide-svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { getStorageLocations } from '$modules/storage-location/services/storage-location-service';
  import { getLocationStock, setLocationStock, transferLocationStock } from '../services/inventory-service';
  import type { LocationStockItem } from '../types';
  import type { StorageLocation } from '$modules/storage-location/types';

  let {
    productId = null as number | null,
    productName = '',
    canAdjust = false,
    onChanged = () => {},
  } = $props();

  let rackRows = $state<LocationStockItem[]>([]);
  let locations = $state<StorageLocation[]>([]);
  let loading = $state(false);

  let showSetModal = $state(false);
  let setLocationId = $state<number | undefined>(undefined);
  let setQuantity = $state(0);
  let savingSet = $state(false);

  let showTransferModal = $state(false);
  let fromLocationId = $state<number | undefined>(undefined);
  let toLocationId = $state<number | undefined>(undefined);
  let transferQuantity = $state(0);
  let savingTransfer = $state(false);

  let setErrors = $state<Record<string, string>>({});
  let transferErrors = $state<Record<string, string>>({});

  function locationLabel(l: { name: string; code: string }): string {
    return l.code ? `${l.name} (${l.code})` : l.name;
  }

  function locationOptions(): { value: number; label: string }[] {
    return locations.map((l) => ({ value: l.id, label: locationLabel(l) }));
  }

  function locationName(id: number | undefined): string {
    if (!id) return '-';
    const found = locations.find((l) => l.id === id);
    return found ? locationLabel(found) : `#${id}`;
  }

  async function load() {
    if (!productId) {
      rackRows = [];
      locations = [];
      return;
    }
    loading = true;
    let rows: LocationStockItem[] = [];
    try {
      rows = await getLocationStock(productId);
    } catch {
      toast.error('Gagal memuat stok rak');
    }
    rackRows = rows;

    // Location metadata is only needed for the set/transfer modals (gated by
    // canAdjust); rack rows already carry their own location labels, so
    // read-only viewers never need the storage-locations call. A metadata
    // failure must not blank the rack rows.
    let locs: StorageLocation[] = [];
    if (canAdjust) {
      try {
        const locRes = await getStorageLocations({ is_active: true, limit: 500, offset: 0 });
        locs = locRes.data;
      } catch {
        locs = [];
      }
    }
    locations = locs;
    loading = false;
  }

  $effect(() => {
    if (productId) load();
    else {
      rackRows = [];
      locations = [];
    }
  });

  function openSet(locationId?: number) {
    setLocationId = locationId ?? (locations[0]?.id ?? undefined);
    setQuantity = 0;
    setErrors = {};
    showSetModal = true;
  }

  async function submitSet() {
    if (!productId || !setLocationId) {
      setErrors = { location: 'Pilih lokasi penyimpanan' };
      return;
    }
    if (setQuantity < 0) {
      setErrors = { quantity: 'Jumlah tidak boleh negatif' };
      return;
    }
    savingSet = true;
    try {
      await setLocationStock({ product_id: productId, location_id: setLocationId, quantity: setQuantity });
      toast.success('Stok rak diperbarui');
      showSetModal = false;
      await load();
      onChanged();
    } catch (e: any) {
      setErrors = { submit: e?.response?.data?.error || e?.message || 'Gagal memperbarui stok rak' };
    } finally {
      savingSet = false;
    }
  }

  function openTransfer(row: LocationStockItem) {
    fromLocationId = row.location_id;
    toLocationId = undefined;
    transferQuantity = 0;
    transferErrors = {};
    showTransferModal = true;
  }

  async function submitTransfer() {
    if (!productId || !fromLocationId || !toLocationId) {
      transferErrors = { location: 'Pilih lokasi tujuan' };
      return;
    }
    if (toLocationId === fromLocationId) {
      transferErrors = { location: 'Lokasi asal dan tujuan harus berbeda' };
      return;
    }
    if (transferQuantity <= 0) {
      transferErrors = { quantity: 'Jumlah harus lebih dari 0' };
      return;
    }
    savingTransfer = true;
    try {
      await transferLocationStock({
        product_id: productId,
        from_location_id: fromLocationId,
        to_location_id: toLocationId,
        quantity: transferQuantity,
      });
      toast.success('Stok rak dipindahkan');
      showTransferModal = false;
      await load();
      onChanged();
    } catch (e: any) {
      transferErrors = { submit: e?.response?.data?.error || e?.message || 'Gagal memindahkan stok rak' };
    } finally {
      savingTransfer = false;
    }
  }
</script>

<div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
  <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
    <span class="text-base leading-none">🗄️</span>
    <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">Stok Rak (Lokasi)</h4>
    {#if canAdjust}
      <Button variant="secondary" size="sm" class="ml-auto" onclick={() => openSet()}>Tambah Stok</Button>
    {/if}
  </div>
  <div class="px-3.5 py-2.5">
    {#if loading}
      <p class="text-xs text-text-muted flex items-center gap-1.5"><Loader2 size={13} class="animate-spin" /> Memuat...</p>
    {:else if rackRows.length === 0}
      <p class="text-xs text-text-muted flex items-center gap-1.5"><PackageX size={14} /> Belum ada stok rak untuk produk ini.</p>
    {:else}
      <div class="space-y-2">
        {#each rackRows as row}
          <div class="flex items-center justify-between gap-3 py-1.5 border-b border-border/40 last:border-b-0">
            <div class="min-w-0">
              <p class="text-sm text-text-primary font-medium truncate">{row.location_name}</p>
              <p class="text-[11px] text-text-muted font-mono">{row.location_code || '—'}</p>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <span class="text-sm font-semibold text-text-secondary tabular-nums">{row.quantity}</span>
              {#if canAdjust}
                <Button variant="ghost" size="sm" onclick={() => openSet(row.location_id)}>Set</Button>
                <Button variant="ghost" size="sm" onclick={() => openTransfer(row)}>Transfer</Button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}
    {#if rackRows.length > 0}
      <p class="text-[11px] text-text-muted/70 mt-2">Stok rak adalah sub-akun dari stok global. Set/transfer tidak mengubah stok global.</p>
    {/if}
  </div>
</div>

<Modal bind:open={showSetModal} title={`Set Stok Rak — ${productName}`} size="sm">
  <div class="space-y-4">
    <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
      <span>Lokasi Penyimpanan</span>
      <SelectSearch
        bind:value={setLocationId}
        options={locationOptions()}
        placeholder="Pilih lokasi..."
        searchPlaceholder="Cari..."
        notFoundText="Tidak ditemukan"
      />
      {#if setErrors.location}<p class="text-xs text-destructive">{setErrors.location}</p>{/if}
    </label>
    <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
      <span>Jumlah (eksak)</span>
      <Input type="number" bind:value={setQuantity} placeholder="0" min={0} />
      {#if setErrors.quantity}<p class="text-xs text-destructive">{setErrors.quantity}</p>{/if}
      <p class="text-xs text-text-muted">Menimpa stok rak saat ini. Stok global tidak berubah.</p>
    </label>
    {#if setErrors.submit}<p class="text-xs text-destructive">{setErrors.submit}</p>{/if}
  </div>
  {#snippet footer()}
    <Button variant="secondary" disabled={savingSet} onclick={() => (showSetModal = false)}>Batal</Button>
    <Button disabled={savingSet} onclick={submitSet}>
      {#if savingSet}<Loader2 size={16} class="animate-spin mr-2" />{/if}Simpan
    </Button>
  {/snippet}
</Modal>

<Modal bind:open={showTransferModal} title="Transfer Stok Rak" size="sm">
  <div class="space-y-4">
    <div>
      <p class="text-sm text-text-muted mb-1">Asal: <span class="text-text-primary font-medium">{locationName(fromLocationId)}</span></p>
      <p class="text-sm text-text-muted">Stok saat ini: <span class="text-text-primary">{rackRows.find((r) => r.location_id === fromLocationId)?.quantity ?? 0}</span></p>
    </div>
    <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
      <span>Lokasi Tujuan</span>
      <SelectSearch
        bind:value={toLocationId}
        options={locationOptions().filter((o) => o.value !== fromLocationId)}
        placeholder="Pilih lokasi tujuan..."
        searchPlaceholder="Cari..."
        notFoundText="Tidak ditemukan"
      />
      {#if transferErrors.location}<p class="text-xs text-destructive">{transferErrors.location}</p>{/if}
    </label>
    <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
      <span>Jumlah</span>
      <Input type="number" bind:value={transferQuantity} placeholder="0" min={1} />
      {#if transferErrors.quantity}<p class="text-xs text-destructive">{transferErrors.quantity}</p>{/if}
      <p class="text-xs text-text-muted">Stok global tidak berubah.</p>
    </label>
    {#if transferErrors.submit}<p class="text-xs text-destructive">{transferErrors.submit}</p>{/if}
  </div>
  {#snippet footer()}
    <Button variant="secondary" disabled={savingTransfer} onclick={() => (showTransferModal = false)}>Batal</Button>
    <Button disabled={savingTransfer} onclick={submitTransfer}>
      {#if savingTransfer}<Loader2 size={16} class="animate-spin mr-2" />{/if}Transfer
    </Button>
  {/snippet}
</Modal>
