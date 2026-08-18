import { apiFetch } from '$shared/api/http-client';

export interface BrandingSettings {
  store_name: string;
  store_jargon: string;
  logo_path: string;
}

export interface AllSettings {
  store_name: string;
  store_jargon: string;
  logo_path: string;
  default_language: string;
  receipt_header: string;
  receipt_footer: string;
}

/**
 * Fetch public branding (no auth). Called on app init before login.
 * Uses raw `fetch` intentionally — `apiFetch` injects auth tokens which
 * are not available (and not needed) for this public endpoint.
 */
export async function fetchPublicBranding(): Promise<BrandingSettings | null> {
  try {
    const res = await fetch('/api/settings/public');
    if (res.ok) return res.json();
  } catch { /* ignore */ }
  return null;
}

/**
 * Fetch all settings (auth required). Called after login.
 */
export async function fetchAllSettings(): Promise<AllSettings | null> {
  try {
    const res = await apiFetch('/api/settings');
    if (res.ok) {
      const data = await res.json();
      return data.settings ?? null;
    }
  } catch { /* ignore */ }
  return null;
}

/**
 * Bulk-update settings (auth required).
 */
export async function updateSettings(settings: Record<string, string>): Promise<boolean> {
  const res = await apiFetch('/api/settings', {
    method: 'PUT',
    body: JSON.stringify({ settings }),
  });
  return res.ok;
}

/**
 * Upload a logo image (auth required).
 */
export async function uploadLogo(file: File): Promise<string | null> {
  const formData = new FormData();
  formData.append('file', file);
  const res = await apiFetch('/api/settings/logo', {
    method: 'POST',
    body: formData,
  });
  if (res.ok) {
    const data = await res.json();
    return data.logo_path ?? null;
  }
  return null;
}

/**
 * Remove the current logo (auth required).
 */
export async function removeLogo(): Promise<boolean> {
  const res = await apiFetch('/api/settings/logo', { method: 'DELETE' });
  return res.ok;
}
