import apiClient from '$shared/api/http-client';
import type { AuditLog, AuditLogFilters } from '../types';

export interface AuditLogListResponse {
  data: AuditLog[];
  total: number;
}

export async function getAuditLogs(
  filters: AuditLogFilters,
  signal?: AbortSignal,
): Promise<AuditLogListResponse> {
  const params = new URLSearchParams({
    limit: filters.limit.toString(),
    offset: filters.offset.toString(),
    search: filters.search,
    start_date: filters.start_date,
    end_date: filters.end_date,
  });
  if (filters.action) params.append('action', filters.action);
  if (filters.entity_type) params.append('entity_type', filters.entity_type);

  const response = await apiClient.get(`audit-logs?${params.toString()}`, { signal });
  const data = response.data || {};
  return { data: data.data || [], total: data.total || 0 };
}

export function buildExportUrl(
  format: string,
  filters: {
    search: string;
    start_date: string;
    end_date: string;
    action?: string;
    entity_type?: string;
  },
): string {
  const params = new URLSearchParams({
    format,
    search: filters.search,
    start_date: filters.start_date,
    end_date: filters.end_date,
  });
  if (filters.action) params.append('action', filters.action);
  if (filters.entity_type) params.append('entity_type', filters.entity_type);
  return `/api/audit-logs/export?${params.toString()}`;
}
