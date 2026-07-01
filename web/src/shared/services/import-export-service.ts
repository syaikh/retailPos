import apiClient from '$shared/api/http-client';
import type { PreviewResult, ImportResult, ImportProgress, ModuleInfo, ExportFormat } from '$shared/types/import-export';

const BASE = '/import-export';

function getToken(): string {
  return sessionStorage.getItem('access_token') || '';
}

export async function getModules(): Promise<ModuleInfo[]> {
  const { data } = await apiClient.get(`${BASE}/modules`);
  return data;
}

export function downloadTemplate(module: string): void {
  const token = getToken();
  window.open(`${import.meta.env.VITE_API_URL || ''}/api${BASE}/template/${module}?token=${token}`, '_blank');
}

export async function uploadPreview(module: string, file: File): Promise<PreviewResult> {
  const form = new FormData();
  form.append('file', file);
  const { data } = await apiClient.post(`${BASE}/preview/${module}`, form);
  return data;
}

export async function confirmImport(module: string, token: string): Promise<ImportResult> {
  const { data } = await apiClient.post(`${BASE}/confirm/${module}?token=${token}`);
  return data;
}

export async function getProgress(jobId: string): Promise<ImportProgress> {
  const { data } = await apiClient.get(`${BASE}/progress/${jobId}`);
  return data;
}

export async function cancelImport(jobId: string): Promise<void> {
  await apiClient.post(`${BASE}/cancel/${jobId}`);
}

export function downloadExport(module: string, format: ExportFormat): void {
  const token = getToken();
  window.open(`${import.meta.env.VITE_API_URL || ''}/api${BASE}/export/${module}?format=${format}&token=${token}`, '_blank');
}
