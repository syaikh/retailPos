import apiClient from '$shared/api/http-client';
import type { PreviewResult, ImportResult, ImportProgress, ModuleInfo, ExportFormat } from '$shared/types/import-export';

const BASE = '/import-export';

export async function getModules(): Promise<ModuleInfo[]> {
  const { data } = await apiClient.get(`${BASE}/modules`);
  return data;
}

export async function downloadTemplate(module: string): Promise<void> {
  const response = await apiClient.get(`${BASE}/template/${module}`, { responseType: 'blob' });
  const url = window.URL.createObjectURL(new Blob([response.data]));
  const link = document.createElement('a');
  link.href = url;
  link.setAttribute('download', `${module}-template.xlsx`);
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(url);
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

export async function getHistory(module: string): Promise<ImportProgress[]> {
  const { data } = await apiClient.get(`${BASE}/history/${module}`);
  return data;
}

export async function cancelImport(jobId: string): Promise<void> {
  await apiClient.post(`${BASE}/cancel/${jobId}`);
}

export async function downloadExport(module: string, format: ExportFormat): Promise<void> {
  const response = await apiClient.get(`${BASE}/export/${module}`, {
    params: { format },
    responseType: 'blob',
  });
  const disposition = response.headers?.['content-disposition'] as string | undefined;
  const match = disposition?.match(/filename="?(.+?)"?$/);
  const filename = match?.[1] ?? `${module}-export.${format}`;
  const url = window.URL.createObjectURL(new Blob([response.data]));
  const link = document.createElement('a');
  link.href = url;
  link.setAttribute('download', filename);
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(url);
}
