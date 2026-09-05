<script>
  import { onMount } from 'svelte';
  import { labels } from '$shared/i18n';
  import { settingsStore, loadFullSettings } from '$shared/stores/settings.svelte';
  import { fetchAllSettings, updateSettings, uploadLogo, removeLogo } from '../services/app-settings-service';
  import { toast } from '$shared/stores/toast.svelte';
  import { useRBAC } from '$shared/composables/useRBAC.svelte';
  import { Permissions } from '$shared/constants/permissions';
  import { PageHeader, Button, Card, Input } from '$shared/ui';
  import { Save, Upload, Trash2, Image as ImageIcon, Loader2, Info, Globe, Receipt, Store, Check } from 'lucide-svelte';

  const rbac = useRBAC();

  let storeName = $state('');
  let storeJargon = $state('');
  let receiptHeader = $state('');
  let receiptFooter = $state('');
  let storeAddress = $state('');
  let storePhone = $state('');
  let shiftDiscrepancyThreshold = $state(50000);
  let shiftBlindClose = $state(false);

  let loading = $state(true);
  let saving = $state(false);
  let logoPreview = $state('');
  let logoFile = $state(null);
  let logoRemoved = $state(false);
  let fileInput = $state(null);
  let dragOver = $state(false);
  let saved = $state(false);

  const canUpdate = $derived(rbac.can(Permissions.appSettings.update));

  onMount(async () => {
    await loadSettings();
  });

  async function loadSettings() {
    loading = true;
    const data = await fetchAllSettings();
    if (data) {
      storeName = data.store_name ?? '';
      storeJargon = data.store_jargon ?? '';
      storeAddress = data.store_address ?? '';
      storePhone = data.store_phone ?? '';
      receiptHeader = data.receipt_header ?? '';
      receiptFooter = data.receipt_footer ?? '';
      if (data.shift_discrepancy_threshold) {
        const n = parseInt(data.shift_discrepancy_threshold, 10);
        shiftDiscrepancyThreshold = isNaN(n) ? 50000 : n;
      }
      if (data.shift_blind_close !== undefined) {
        shiftBlindClose = data.shift_blind_close === 'true';
      }
      if (data.logo_path) {
        logoPreview = `/api/settings/logo?v=${Date.now()}`;
      }
    }
    loading = false;
  }

  function handleLogoDrop(e) {
    e.preventDefault();
    dragOver = false;
    const file = e.dataTransfer?.files?.[0] ?? e.target?.files?.[0];
    if (!file) return;

    const allowed = ['image/png', 'image/jpeg'];
    if (!allowed.includes(file.type)) {
      toast.error(labels.logoUploadError);
      return;
    }
    if (file.size > 2 * 1024 * 1024) {
      toast.error(labels.logoMaxSize);
      return;
    }

    logoFile = file;
    logoRemoved = false;
    logoPreview = URL.createObjectURL(file);
  }

  function handleRemoveLogo() {
    logoFile = null;
    logoRemoved = true;
    logoPreview = '';
    if (fileInput) fileInput.value = '';
  }

  function triggerFileInput() {
    fileInput?.click();
  }

  async function handleSave() {
    if (!storeName.trim()) {
      toast.error('Store name is required');
      return;
    }

    saving = true;

    if (logoFile) {
      const logoPath = await uploadLogo(logoFile);
      if (!logoPath) {
        toast.error(labels.logoUploadError);
        saving = false;
        return;
      }
      settingsStore.updateBranding({ logo_path: logoPath });
    } else if (logoRemoved) {
      await removeLogo();
      settingsStore.updateBranding({ logo_path: '' });
    }

    const settings = {
      store_name: storeName.trim(),
      store_jargon: storeJargon.trim(),
      store_address: storeAddress.trim(),
      store_phone: storePhone.trim(),
      receipt_header: receiptHeader.trim(),
      receipt_footer: receiptFooter.trim(),
      shift_discrepancy_threshold: shiftDiscrepancyThreshold.toString(),
      shift_blind_close: shiftBlindClose.toString(),
    };

    const ok = await updateSettings(settings);
    if (ok) {
      settingsStore.updateAll(settings);
      saved = true;
      toast.success(labels.settingsSaved);
      setTimeout(() => (saved = false), 2000);
    } else {
      toast.error(labels.settingsSaveError);
    }

    saving = false;
  }
</script>

<div class="max-w-4xl mx-auto px-4 sm:px-6 py-8 space-y-8">
  <PageHeader title={labels.appSettings} subtitle={canUpdate ? labels.settingsPageSubtitle : undefined}>
    {#snippet actions()}
      {#if canUpdate}
        <Button onclick={handleSave} disabled={saving || loading} variant="primary">
          {#if saving}
            <Loader2 size={16} class="animate-spin" />
            {labels.saving}
          {:else if saved}
            <Check size={16} />
            {labels.saved}
          {:else}
            <Save size={16} />
            {labels.save}
          {/if}
        </Button>
      {/if}
    {/snippet}
  </PageHeader>

  {#if !canUpdate}
    <div class="flex items-center gap-3 p-4 rounded-xl bg-amber-50 dark:bg-amber-900/15 border border-amber-200 dark:border-amber-800/30">
      <div class="flex-shrink-0 w-8 h-8 rounded-lg bg-amber-100 dark:bg-amber-900/30 flex items-center justify-center">
        <Info size={16} class="text-amber-600 dark:text-amber-400" />
      </div>
      <p class="text-sm text-amber-700 dark:text-amber-300">{labels.settingsReadOnly}</p>
    </div>
  {/if}

  {#if loading}
    <div class="space-y-8">
      {#each [1, 2, 3] as _}
        <div class="h-56 bg-surface-secondary rounded-2xl animate-pulse"></div>
      {/each}
    </div>
  {:else}
    <!-- ─── Branding ──────────────────────────────────────────── -->
    <Card class="overflow-hidden">
      <div class="px-6 pt-6 pb-4">
        <div class="flex items-center gap-3 mb-6">
          <div class="w-9 h-9 rounded-xl bg-primary/10 flex items-center justify-center">
            <Store size={18} class="text-primary" />
          </div>
          <div>
            <h2 class="text-base font-semibold text-text-primary">{labels.storeBranding}</h2>
            <p class="text-xs text-text-muted mt-0.5">{labels.brandingSubtitle}</p>
          </div>
        </div>

        <!-- Logo Upload Zone -->
        <div class="flex flex-col sm:flex-row items-start gap-5">
          <div class="relative group">
            <input
              bind:this={fileInput}
              type="file"
              accept="image/png,image/jpeg"
              class="hidden"
              onchange={handleLogoDrop}
            />
            <button
              type="button"
              class="relative w-28 h-28 rounded-2xl border-2 border-dashed overflow-hidden transition-all duration-200
                {dragOver
                  ? 'border-primary bg-primary/5 scale-105'
                  : logoPreview
                    ? 'border-border-default hover:border-primary/50 hover:shadow-lg hover:shadow-primary/5'
                    : 'border-border hover:border-primary/40 hover:bg-primary/5'}
                {canUpdate ? 'cursor-pointer' : 'cursor-default'}"
              onclick={triggerFileInput}
              ondrop={handleLogoDrop}
              ondragover={(e) => { e.preventDefault(); dragOver = true; }}
              ondragleave={() => dragOver = false}
              disabled={!canUpdate}
            >
              {#if logoPreview}
                <img src={logoPreview} alt="Logo" class="w-full h-full object-contain p-1" />
                {#if canUpdate}
                  <div class="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                    <Upload size={18} class="text-white" />
                  </div>
                {/if}
              {:else}
                <div class="w-full h-full flex flex-col items-center justify-center gap-1.5 bg-surface-secondary">
                  <ImageIcon size={24} class="text-text-muted" />
                  <span class="text-[10px] text-text-muted font-medium">Logo</span>
                </div>
              {/if}
            </button>
            {#if logoPreview && canUpdate}
              <button
                type="button"
                class="absolute -top-1.5 -right-1.5 w-6 h-6 rounded-full bg-danger text-white flex items-center justify-center shadow-md opacity-0 group-hover:opacity-100 transition-opacity"
                onclick={handleRemoveLogo}
              >
                <Trash2 size={12} />
              </button>
            {/if}
          </div>

          <div class="flex-1 space-y-3">
            <div>
              <p class="text-sm text-text-secondary">{canUpdate ? labels.clickLogoToUpload : labels.currentLogo}</p>
              <p class="text-xs text-text-muted mt-1">{labels.logoFormatInfo}. {labels.logoSizeHint}.</p>
            </div>
            {#if !logoPreview && canUpdate}
              <Button variant="ghost" size="sm" onclick={triggerFileInput}>
                <Upload size={14} />
                {labels.uploadLogo}
              </Button>
            {/if}
          </div>
        </div>
      </div>

      <div class="border-t border-border-default"></div>

      <!-- Store Name & Jargon -->
      <div class="px-6 py-5 space-y-4">
        <div>
          <label for="store-name" class="block text-sm font-medium text-text-secondary mb-1.5">
            {labels.storeName} <span class="text-danger">*</span>
          </label>
          <Input
            id="store-name"
            type="text"
            bind:value={storeName}
            placeholder={labels.storeNamePlaceholder}
            disabled={!canUpdate}
          />
        </div>
        <div>
          <label for="store-jargon" class="block text-sm font-medium text-text-secondary mb-1.5">{labels.storeJargon}</label>
          <Input
            id="store-jargon"
            type="text"
            bind:value={storeJargon}
            placeholder={labels.storeJargonPlaceholder}
            disabled={!canUpdate}
          />
          <p class="text-xs text-text-muted mt-1.5">{labels.jargonPlacementHint}</p>
        </div>
      </div>
    </Card>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
      <!-- ─── Branch Info ──────────────────────────────────────── -->
      <Card>
        <div class="px-6 py-5">
          <div class="flex items-center gap-3 mb-5">
            <div class="w-9 h-9 rounded-xl bg-emerald-500/10 flex items-center justify-center">
              <Globe size={18} class="text-emerald-600 dark:text-emerald-400" />
            </div>
            <div>
              <h2 class="text-base font-semibold text-text-primary">{labels.branchInfo}</h2>
              <p class="text-xs text-text-muted mt-0.5">{labels.branchInfoSubtitle}</p>
            </div>
          </div>
          <div class="space-y-3">
            <div>
              <label for="branch-address" class="block text-sm font-medium text-text-secondary mb-1.5">{labels.address}</label>
              <Input
                id="branch-address"
                type="text"
                bind:value={storeAddress}
                placeholder={labels.storeAddressPlaceholder}
                disabled={!canUpdate}
              />
            </div>
            <div>
              <label for="branch-phone" class="block text-sm font-medium text-text-secondary mb-1.5">{labels.phone}</label>
              <Input
                id="branch-phone"
                type="text"
                bind:value={storePhone}
                placeholder={labels.storePhonePlaceholder}
                disabled={!canUpdate}
              />
            </div>
          </div>
          <p class="text-xs text-text-muted mt-3">{labels.branchInfoHint}</p>
        </div>
      </Card>

      <!-- ─── Receipt ───────────────────────────────────────────── -->
      <Card>
        <div class="px-6 py-5">
          <div class="flex items-center gap-3 mb-5">
            <div class="w-9 h-9 rounded-xl bg-violet-500/10 flex items-center justify-center">
              <Receipt size={18} class="text-violet-600 dark:text-violet-400" />
            </div>
            <div>
              <h2 class="text-base font-semibold text-text-primary">{labels.receiptSettings}</h2>
              <p class="text-xs text-text-muted mt-0.5">{labels.receiptSubtitle}</p>
            </div>
          </div>
          <div class="space-y-4">
            <div>
              <label for="receipt-header" class="block text-sm font-medium text-text-secondary mb-1.5">{labels.receiptHeader}</label>
              <Input
                tag="textarea"
                id="receipt-header"
                bind:value={receiptHeader}
                placeholder={labels.receiptHeaderPlaceholder}
                rows="3"
              />
              <p class="text-xs text-text-muted mt-1.5">{labels.receiptHeaderHint}</p>
            </div>
            <div>
              <label for="receipt-footer" class="block text-sm font-medium text-text-secondary mb-1.5">{labels.receiptFooter}</label>
              <Input
                tag="textarea"
                id="receipt-footer"
                bind:value={receiptFooter}
                placeholder={labels.receiptFooterPlaceholder}
                rows="3"
              />
            </div>
          </div>
        </div>
      </Card>
    </div>

    <!-- ─── Shift Management ────────────────────────────────── -->
    <div>
      <Card variant="glass">
        <div class="px-6 py-5">
          <div class="flex items-center gap-2 mb-4">
            <div class="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center">
              <span class="text-base">Shift</span>
            </div>
            <div>
              <h3 class="text-sm font-semibold text-text-primary">{labels.shiftManagement}</h3>
              <p class="text-xs text-text-muted">{labels.shiftManagementDesc}</p>
            </div>
          </div>
          <div class="space-y-4">
            <div>
              <label for="discrepancy-threshold" class="block text-sm font-medium text-text-secondary mb-1.5">
                {labels.discrepancyThreshold}
              </label>
              <Input
                id="discrepancy-threshold"
                type="number"
                bind:value={shiftDiscrepancyThreshold}
                placeholder="50000"
              />
              <p class="text-xs text-text-muted mt-1.5">{labels.discrepancyThresholdHint}</p>
            </div>
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-text-secondary">{labels.blindClose}</label>
                <p class="text-xs text-text-muted">{labels.blindCloseDesc}</p>
              </div>
              <button
                type="button"
                class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors {shiftBlindClose ? 'bg-primary' : 'bg-border'}"
                onclick={() => { shiftBlindClose = !shiftBlindClose; }}
                disabled={!canUpdate}
              >
                <span class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform {shiftBlindClose ? 'translate-x-6' : 'translate-x-1'}"></span>
              </button>
            </div>
          </div>
        </div>
      </Card>
    </div>

    <!-- ─── Receipt Preview ──────────────────────────────────── -->
    {#if receiptHeader || receiptFooter || storeName || storeAddress || storePhone}
      <Card variant="glass">
        <div class="px-6 py-5">
          <h3 class="text-sm font-semibold text-text-secondary mb-4">{labels.receiptPreview}</h3>
          <div class="max-w-[280px] mx-auto bg-white dark:bg-gray-950 rounded-xl border border-border dark:border-gray-800 shadow-sm overflow-hidden font-mono text-xs">
            <div class="p-4 space-y-3 text-center text-text-primary dark:text-gray-100">
              {#if storeName}
                <p class="font-bold text-sm tracking-wide">{storeName}</p>
              {/if}
              {#if storeJargon}
                <p class="text-[10px] text-text-muted dark:text-gray-400 -mt-1">{storeJargon}</p>
              {/if}
              {#if storeAddress}
                <p class="text-[10px] text-text-muted dark:text-gray-400">{storeAddress}</p>
              {/if}
              {#if storePhone}
                <p class="text-[10px] text-text-muted dark:text-gray-400">{labels.phone}: {storePhone}</p>
              {/if}
              {#if receiptHeader}
                <div class="whitespace-pre-line text-[10px] text-text-secondary dark:text-gray-300 leading-relaxed">{receiptHeader}</div>
              {/if}
              <div class="border-t border-dashed border-border dark:border-gray-700"></div>
              <div class="space-y-1 text-left text-[10px] text-text-muted">
                <div class="flex justify-between">
                   <span>Sample Product A</span>
                   <span>{labels.currencySymbol} 25,000</span>
                 </div>
                 <div class="flex justify-between">
                   <span>Sample Product B</span>
                   <span>{labels.currencySymbol} 15,000</span>
                 </div>
                <div class="border-t border-dashed border-border dark:border-gray-700 my-1"></div>
                <div class="flex justify-between font-bold text-text-primary dark:text-gray-100">
                   <span>{labels.total}</span>
                   <span>{labels.currencySymbol} 40,000</span>
                </div>
              </div>
              {#if receiptFooter}
                <div class="border-t border-dashed border-border dark:border-gray-700"></div>
                <div class="whitespace-pre-line text-[10px] text-text-muted dark:text-gray-400 leading-relaxed">{receiptFooter}</div>
              {/if}
            </div>
          </div>
        </div>
      </Card>
    {/if}
  {/if}
</div>
