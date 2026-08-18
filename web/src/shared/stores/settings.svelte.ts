import { fetchPublicBranding, fetchAllSettings, type AllSettings } from '$modules/admin/services/app-settings-service';

class SettingsStore {
  storeName = $state('RetailPOS');
  storeJargon = $state('Management System');
  logoPath = $state('');
  defaultLanguage = $state('id');
  receiptHeader = $state('');
  receiptFooter = $state('Terima kasih atas kunjungan Anda!');

  updateBranding(data: { store_name?: string; store_jargon?: string; logo_path?: string }) {
    if (data.store_name !== undefined) this.storeName = data.store_name;
    if (data.store_jargon !== undefined) this.storeJargon = data.store_jargon;
    if (data.logo_path !== undefined) this.logoPath = data.logo_path;
  }

  updateAll(data: Partial<AllSettings>) {
    if (data.store_name !== undefined) this.storeName = data.store_name;
    if (data.store_jargon !== undefined) this.storeJargon = data.store_jargon;
    if (data.logo_path !== undefined) this.logoPath = data.logo_path;
    if (data.default_language !== undefined) this.defaultLanguage = data.default_language;
    if (data.receipt_header !== undefined) this.receiptHeader = data.receipt_header;
    if (data.receipt_footer !== undefined) this.receiptFooter = data.receipt_footer;
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
