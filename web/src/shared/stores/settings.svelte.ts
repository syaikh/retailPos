import { fetchPublicBranding, fetchAllSettings, type AllSettings } from '$modules/admin/services/app-settings-service';

class SettingsStore {
  storeName = $state('RetailPOS');
  storeJargon = $state('Management System');
  logoPath = $state('');
  storeAddress = $state('');
  storePhone = $state('');
  receiptHeader = $state('');
  receiptFooter = $state('Terima kasih atas kunjungan Anda!');
  shiftDiscrepancyThreshold = $state(50000);
  shiftBlindClose = $state(false);

  updateBranding(data: { store_name?: string; store_jargon?: string; logo_path?: string }) {
    if (data.store_name !== undefined) this.storeName = data.store_name;
    if (data.store_jargon !== undefined) this.storeJargon = data.store_jargon;
    if (data.logo_path !== undefined) this.logoPath = data.logo_path;
  }

  updateAll(data: Partial<AllSettings>) {
    if (data.store_name !== undefined) this.storeName = data.store_name;
    if (data.store_jargon !== undefined) this.storeJargon = data.store_jargon;
    if (data.logo_path !== undefined) this.logoPath = data.logo_path;
    if (data.store_address !== undefined) this.storeAddress = data.store_address;
    if (data.store_phone !== undefined) this.storePhone = data.store_phone;
    if (data.receipt_header !== undefined) this.receiptHeader = data.receipt_header;
    if (data.receipt_footer !== undefined) this.receiptFooter = data.receipt_footer;
    if (data.shift_discrepancy_threshold !== undefined) {
      const n = parseInt(data.shift_discrepancy_threshold, 10);
      this.shiftDiscrepancyThreshold = isNaN(n) ? 50000 : n;
    }
    if (data.shift_blind_close !== undefined) {
      this.shiftBlindClose = data.shift_blind_close === 'true';
    }
  }
}

export const settingsStore = new SettingsStore();

/**
 * Fetch public branding (no auth). Call at app init, before login.
 */
export async function initSettings(): Promise<void> {
  const data = await fetchPublicBranding();
  if (data) settingsStore.updateBranding(data);
}

/**
 * Fetch all settings (auth required). Call after successful login.
 */
export async function loadFullSettings(): Promise<void> {
  const data = await fetchAllSettings();
  if (data) settingsStore.updateAll(data);
}
