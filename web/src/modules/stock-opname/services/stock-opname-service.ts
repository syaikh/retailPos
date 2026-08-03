import apiClient from '$shared/api/http-client';
import type {
  StockOpnameSession,
  StockOpnameAssignment,
  AssignableUser,
  CountRecord,
  SessionSummary,
  StockOpnameFilters,
  CreateStockOpnamePayload,
  AssignPayload,
  ReassignPayload,
  SaveCountPayload,
  VerifyPayload,
  RejectPayload,
  RecountPayload,
  PostAdjustmentPayload,
  Adjustment,
} from '../types';

export async function createStockOpname(payload: CreateStockOpnamePayload): Promise<StockOpnameSession> {
  const res = await apiClient.post('/stock-opnames', payload);
  return res.data.data;
}

export async function listStockOpnames(
  filters: StockOpnameFilters,
  signal?: AbortSignal
): Promise<{ data: StockOpnameSession[]; total: number }> {
  const params = new URLSearchParams({
    limit: filters.limit.toString(),
    offset: filters.offset.toString(),
  });
  if (filters.status) params.set('status', filters.status);
  if (filters.search) params.set('search', filters.search);

  const res = await apiClient.get(`/stock-opnames?${params.toString()}`, { signal });
  return { data: res.data.data || [], total: res.data.total || 0 };
}

export async function getStockOpname(id: number): Promise<StockOpnameSession> {
  const res = await apiClient.get(`/stock-opnames/${id}`);
  return res.data.data;
}

export async function openStockOpname(id: number, comment: string): Promise<void> {
  await apiClient.post(`/stock-opnames/${id}/open`, { comment });
}

export async function cancelStockOpname(id: number): Promise<void> {
  await apiClient.post(`/stock-opnames/${id}/cancel`);
}

export async function assignCounter(id: number, payload: AssignPayload): Promise<void> {
  await apiClient.post(`/stock-opnames/${id}/assignments`, payload);
}

export async function getAssignableUsers(search?: string): Promise<AssignableUser[]> {
  const params = search ? `?${new URLSearchParams({ search })}` : '';
  const res = await apiClient.get(`/stock-opnames/assignable-users${params}`);
  return res.data.data || [];
}

export async function getAssignments(id: number): Promise<StockOpnameAssignment[]> {
  const res = await apiClient.get(`/stock-opnames/${id}/assignments`);
  return res.data.data || [];
}

export async function reassignCounter(id: number, assignmentId: number, payload: ReassignPayload): Promise<void> {
  await apiClient.put(`/stock-opnames/${id}/assignments/${assignmentId}`, payload);
}

export async function saveCount(itemId: number, payload: SaveCountPayload): Promise<void> {
  await apiClient.put(`/stock-opnames/items/${itemId}/count`, payload);
}

export async function getCountHistory(itemId: number): Promise<CountRecord[]> {
  const res = await apiClient.get(`/stock-opnames/items/${itemId}/counts`);
  return res.data.data || [];
}

export async function startCounting(id: number): Promise<void> {
  await apiClient.post(`/stock-opnames/${id}/start`);
}

export async function submitSession(id: number): Promise<void> {
  await apiClient.post(`/stock-opnames/${id}/submit`);
}

export async function verifySession(id: number, payload: VerifyPayload): Promise<void> {
  await apiClient.post(`/stock-opnames/${id}/verify`, payload);
}

export async function rejectSession(id: number, payload: RejectPayload): Promise<void> {
  await apiClient.post(`/stock-opnames/${id}/reject`, payload);
}

export async function requestRecount(id: number, payload: RecountPayload): Promise<void> {
  await apiClient.post(`/stock-opnames/${id}/recount`, payload);
}

export async function resumeCounting(id: number): Promise<void> {
  await apiClient.post(`/stock-opnames/${id}/resume`);
}

export async function postAdjustment(id: number, payload: PostAdjustmentPayload): Promise<Adjustment> {
  const res = await apiClient.post(`/stock-opnames/${id}/post-adjustment`, payload);
  return res.data.data;
}

export async function closeStockOpname(id: number): Promise<void> {
  await apiClient.post(`/stock-opnames/${id}/close`);
}

export async function getSessionSummary(id: number): Promise<SessionSummary> {
  const res = await apiClient.get(`/stock-opnames/${id}/summary`);
  return res.data.data;
}

export async function getDifferenceReport(id: number): Promise<StockOpnameSession> {
  const res = await apiClient.get(`/stock-opnames/${id}/difference`);
  return res.data.data;
}

export async function listAdjustments(
  filters: { status?: string; search?: string; limit: number; offset: number },
  signal?: AbortSignal
): Promise<{ data: Adjustment[]; total: number }> {
  const params = new URLSearchParams({
    limit: filters.limit.toString(),
    offset: filters.offset.toString(),
  });
  if (filters.status) params.set('status', filters.status);
  if (filters.search) params.set('search', filters.search);

  const res = await apiClient.get(`/stock-opnames/adjustments?${params.toString()}`, { signal });
  return { data: res.data.data || [], total: res.data.total || 0 };
}

export async function getAdjustment(id: number): Promise<Adjustment> {
  const res = await apiClient.get(`/stock-opnames/adjustments/${id}`);
  return res.data.data;
}

export async function exportStockOpname(id: number): Promise<Blob> {
  const res = await apiClient.get(`/stock-opnames/${id}/export`, {
    responseType: 'blob',
  });
  return res.data;
}
