import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { settingsStore, initSettings, loadFullSettings } from '../settings.svelte';

vi.mock('$modules/admin/services/app-settings-service', () => ({
  fetchPublicBranding: vi.fn(),
  fetchAllSettings: vi.fn(),
}));

import { fetchPublicBranding, fetchAllSettings } from '$modules/admin/services/app-settings-service';

const mockFetchPublicBranding = vi.mocked(fetchPublicBranding);
const mockFetchAllSettings = vi.mocked(fetchAllSettings);

describe('SettingsStore', () => {
  beforeEach(() => {
    settingsStore.storeName = 'RetailPOS';
    settingsStore.storeJargon = 'Management System';
    settingsStore.logoPath = '';
    settingsStore.storeAddress = '';
    settingsStore.storePhone = '';
    settingsStore.receiptHeader = '';
    settingsStore.receiptFooter = 'Terima kasih atas kunjungan Anda!';
    settingsStore.shiftDiscrepancyThreshold = 50000;
    settingsStore.shiftBlindClose = false;
    vi.resetAllMocks();
  });

  it('has correct defaults', () => {
    expect(settingsStore.storeName).toBe('RetailPOS');
    expect(settingsStore.storeJargon).toBe('Management System');
    expect(settingsStore.logoPath).toBe('');
    expect(settingsStore.storeAddress).toBe('');
    expect(settingsStore.storePhone).toBe('');
    expect(settingsStore.receiptHeader).toBe('');
    expect(settingsStore.receiptFooter).toBe('Terima kasih atas kunjungan Anda!');
    expect(settingsStore.shiftDiscrepancyThreshold).toBe(50000);
    expect(settingsStore.shiftBlindClose).toBe(false);
  });

  it('updateBranding updates storeName, storeJargon, logoPath', () => {
    settingsStore.updateBranding({
      store_name: 'New Store',
      store_jargon: 'POS System',
      logo_path: 'logo.png',
    });
    expect(settingsStore.storeName).toBe('New Store');
    expect(settingsStore.storeJargon).toBe('POS System');
    expect(settingsStore.logoPath).toBe('logo.png');
    expect(settingsStore.storeAddress).toBe('');
  });

  it('updateBranding ignores undefined fields', () => {
    settingsStore.storeName = 'Original';
    settingsStore.updateBranding({ store_jargon: 'Updated Jargon' });
    expect(settingsStore.storeName).toBe('Original');
    expect(settingsStore.storeJargon).toBe('Updated Jargon');
  });

  it('updateAll updates all fields including shift settings', () => {
    settingsStore.updateAll({
      store_name: 'Full Store',
      store_jargon: 'Full Jargon',
      logo_path: 'full-logo.png',
      store_address: 'Jl. Sudirman 123',
      store_phone: '021-1234567',
      receipt_header: 'Welcome!',
      receipt_footer: 'Come again!',
      shift_discrepancy_threshold: '75000',
      shift_blind_close: 'true',
    });
    expect(settingsStore.storeName).toBe('Full Store');
    expect(settingsStore.storeJargon).toBe('Full Jargon');
    expect(settingsStore.logoPath).toBe('full-logo.png');
    expect(settingsStore.storeAddress).toBe('Jl. Sudirman 123');
    expect(settingsStore.storePhone).toBe('021-1234567');
    expect(settingsStore.receiptHeader).toBe('Welcome!');
    expect(settingsStore.receiptFooter).toBe('Come again!');
    expect(settingsStore.shiftDiscrepancyThreshold).toBe(75000);
    expect(settingsStore.shiftBlindClose).toBe(true);
  });

  it('updateAll handles non-numeric threshold gracefully', () => {
    settingsStore.updateAll({ shift_discrepancy_threshold: 'abc' });
    expect(settingsStore.shiftDiscrepancyThreshold).toBe(50000);
  });

  it('updateAll parses blind_close string correctly', () => {
    settingsStore.updateAll({ shift_blind_close: 'false' });
    expect(settingsStore.shiftBlindClose).toBe(false);

    settingsStore.updateAll({ shift_blind_close: 'true' });
    expect(settingsStore.shiftBlindClose).toBe(true);
  });
});

describe('initSettings', () => {
  beforeEach(() => {
    settingsStore.storeName = 'RetailPOS';
    settingsStore.storeJargon = 'Management System';
    settingsStore.logoPath = '';
  });

  it('fetches public branding and updates store', async () => {
    mockFetchPublicBranding.mockResolvedValue({
      store_name: 'Fetched Store',
      store_jargon: 'Fetched Jargon',
      logo_path: 'fetched.png',
    });

    await initSettings();

    expect(settingsStore.storeName).toBe('Fetched Store');
    expect(settingsStore.storeJargon).toBe('Fetched Jargon');
    expect(settingsStore.logoPath).toBe('fetched.png');
    expect(mockFetchPublicBranding).toHaveBeenCalledOnce();
  });

  it('does not crash when fetch fails', async () => {
    mockFetchPublicBranding.mockResolvedValue(null);

    await initSettings();

    expect(settingsStore.storeName).toBe('RetailPOS');
  });
});

describe('loadFullSettings', () => {
  beforeEach(() => {
    settingsStore.storeName = 'RetailPOS';
    settingsStore.storeAddress = '';
    settingsStore.receiptHeader = '';
    settingsStore.shiftDiscrepancyThreshold = 50000;
    settingsStore.shiftBlindClose = false;
  });

  it('fetches all settings and updates store including shift settings', async () => {
    mockFetchAllSettings.mockResolvedValue({
      store_name: 'Full Store',
      store_jargon: 'Full Jargon',
      logo_path: 'full.png',
      store_address: 'Jl. Sudirman 123',
      store_phone: '021-1234567',
      receipt_header: 'Hello',
      receipt_footer: 'Goodbye',
      shift_discrepancy_threshold: '30000',
      shift_blind_close: 'true',
    });

    await loadFullSettings();

    expect(settingsStore.storeName).toBe('Full Store');
    expect(settingsStore.storeAddress).toBe('Jl. Sudirman 123');
    expect(settingsStore.storePhone).toBe('021-1234567');
    expect(settingsStore.receiptHeader).toBe('Hello');
    expect(settingsStore.shiftDiscrepancyThreshold).toBe(30000);
    expect(settingsStore.shiftBlindClose).toBe(true);
    expect(mockFetchAllSettings).toHaveBeenCalledOnce();
  });

  it('does not crash when fetch returns null', async () => {
    mockFetchAllSettings.mockResolvedValue(null);

    await loadFullSettings();

    expect(settingsStore.storeName).toBe('RetailPOS');
  });
});
