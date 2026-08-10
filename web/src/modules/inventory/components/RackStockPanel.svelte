<script lang="ts">
  import { Button, Input, Modal, SelectSearch } from '$shared/ui';
  import { Loader2, PackageX } from 'lucide-svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { labels } from '$shared/i18n';
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
      toast.error(labels.toastGagalMemuatStokRak);
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
      setErrors = { location: labels.pilihLokasiPenyimpanan };
      return;
    }
    if (setQuantity < 0) {
      setErrors = { quantity: labels.jumlahTidakBolehNegatif };
      return;
    }
    savingSet = true;
    try {
      await setLocationStock({ product_id: productId, location_id: setLocationId, quantity: setQuantity });
      toast.success(labels.toastStokRakDiperbarui);
      showSetModal = false;
      await load();
      onChanged();
    } catch (e: any) {
      setErrors = { submit: e?.response?.data?.error || e?.message || labels.toastGagalMemperbaruiStokRak };
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
      transferErrors = { location: labels.pilihLokasiTujuan };
      return;
    }
    if (toLocationId === fromLocationId) {
      transferErrors = { location: labels.lokasiAsalDanTujuanHarusBerbeda };
      return;
    }
    if (transferQuantity <= 0) {
      transferErrors = { quantity: labels.jumlahHarusLebihDari0 };
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
      toast.success(labels.toastStokRakDipindahkan);
      showTransferModal = false;
      await load();
      onChanged();
    } catch (e: any) {
      transferErrors = { submit: e?.response?.data?.error || e?.message || labels.toastGagalMemindahkanStokRak };
    } finally {
      savingTransfer = false;
    }
  }
</script>

<div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
  <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
    <span class="text-base leading-none">🗄️</span>
    <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">{labels.stokRak}</h4>
    {#if canAdjust}
      <Button variant="secondary" size="sm" class="ml-auto" onclick={() => openSet()}>{labels.tambahStok}</Button>
    {/if}
  </div>
  <div class="px-3.5 py-2.5">
    {#if loading}
      <p class="text-xs text-text-muted flex items-center gap-1.5"><Loader2 size={13} class="animate-spin" /> {labels.loading}</p>
    {:else if rackRows.length === 0}
      <p class="text-xs text-text-muted flex items-center gap-1.5"><PackageX size={14} /> {labels.belumAdaStokRak}</p>
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
                <Button variant="ghost" size="sm" onclick={() => openSet(row.location_id)}>{labels.setStokRak}</Button>
                <Button variant="ghost" size="sm" onclick={() => openTransfer(row)}>{labels.transfer}</Button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}
    {#if rackRows.length > 0}
      <p class="text-[11px] text-text-muted/70 mt-2">{labels.stokRakSubAccount}</p>
    {/if}
  </div>
</div>

<Modal bind:open={showSetModal} title={`${labels.setStokRak} — ${productName}`} size="sm">
  <div class="space-y-4">
    <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
      <span>{labels.lokasiPenyimpanan}</span>
      <SelectSearch
        bind:value={setLocationId}
        options={locationOptions()}
        placeholder={labels.pilihLokasi}
        searchPlaceholder={labels.cari}
        notFoundText={labels.tidakDitemukan}
      />
      {#if setErrors.location}<p class="text-xs text-destructive">{setErrors.location}</p>{/if}
    </label>
    <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
      <span>{labels.jumlahEksak}</span>
      <Input type="number" bind:value={setQuantity} placeholder="0" min={0} />
      {#if setErrors.quantity}<p class="text-xs text-destructive">{setErrors.quantity}</p>{/if}
      <p class="text-xs text-text-muted">{labels.menimpaStokRak}</p>
    </label>
    {#if setErrors.submit}<p class="text-xs text-destructive">{setErrors.submit}</p>{/if}
  </div>
  {#snippet footer()}
    <Button variant="secondary" disabled={savingSet} onclick={() => (showSetModal = false)}>{labels.cancel}</Button>
    <Button disabled={savingSet} onclick={submitSet}>
      {#if savingSet}<Loader2 size={16} class="animate-spin mr-2" />{/if}{labels.save}
    </Button>
  {/snippet}
</Modal>

<Modal bind:open={showTransferModal} title={labels.transferStok} size="sm">
  <div class="space-y-4">
    <div>
      <p class="text-sm text-text-muted mb-1">{labels.asal} <span class="text-text-primary font-medium">{locationName(fromLocationId)}</span></p>
      <p class="text-sm text-text-muted">{labels.stokSaatIni} <span class="text-text-primary">{rackRows.find((r) => r.location_id === fromLocationId)?.quantity ?? 0}</span></p>
    </div>
    <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
      <span>{labels.lokasiTujuan}</span>
      <SelectSearch
        bind:value={toLocationId}
        options={locationOptions().filter((o) => o.value !== fromLocationId)}
        placeholder={labels.pilihLokasiTujuan}
        searchPlaceholder={labels.cari}
        notFoundText={labels.tidakDitemukan}
      />
      {#if transferErrors.location}<p class="text-xs text-destructive">{transferErrors.location}</p>{/if}
    </label>
    <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
      <span>{labels.jumlah}</span>
      <Input type="number" bind:value={transferQuantity} placeholder="0" min={1} />
      {#if transferErrors.quantity}<p class="text-xs text-destructive">{transferErrors.quantity}</p>{/if}
      <p class="text-xs text-text-muted">{labels.stokGlobalTidakBerubah}</p>
    </label>
    {#if transferErrors.submit}<p class="text-xs text-destructive">{transferErrors.submit}</p>{/if}
  </div>
  {#snippet footer()}
    <Button variant="secondary" disabled={savingTransfer} onclick={() => (showTransferModal = false)}>{labels.cancel}</Button>
    <Button disabled={savingTransfer} onclick={submitTransfer}>
      {#if savingTransfer}<Loader2 size={16} class="animate-spin mr-2" />{/if}{labels.transfer}
    </Button>
  {/snippet}
</Modal>
