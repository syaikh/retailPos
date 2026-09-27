import apiClient from "$shared/api/http-client";
import type { AuditLog, AuditLogFilters } from "../types";

export interface AuditLogListResponse {
  data: AuditLog[];
  total: number;
  has_more: boolean;
  next_cursor: string | null;
}

export async function getAuditLogs(
  filters: AuditLogFilters,
  signal?: AbortSignal,
  cursor?: string | null,
): Promise<AuditLogListResponse> {
  const params = new URLSearchParams({
    limit: filters.limit.toString(),
    offset: filters.offset.toString(),
    search: filters.search,
    start_date: filters.start_date,
    end_date: filters.end_date,
  });
  if (filters.action) params.append("action", filters.action);
  if (filters.entity_type) params.append("entity_type", filters.entity_type);
  if (cursor) {
    // Keyset cursor takes precedence over offset server-side.
    params.set("offset", "0");
    const sep = cursor.indexOf("|");
    params.append("after_created_at", cursor.slice(0, sep));
    params.append("after_id", cursor.slice(sep + 1));
  }

  const response = await apiClient.get(`audit-logs?${params.toString()}`, {
    signal,
  });
  const data = response.data || {};
  return {
    data: data.data || [],
    total: data.total || 0,
    has_more: data.has_more || false,
    next_cursor: data.next_cursor || null,
  };
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
  if (filters.action) params.append("action", filters.action);
  if (filters.entity_type) params.append("entity_type", filters.entity_type);
  return `/api/audit-logs/export?${params.toString()}`;
}
