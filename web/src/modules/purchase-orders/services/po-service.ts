import { apiFetch } from '$shared/api/http-client';

export async function getPurchaseOrders(filters: any, signal?: AbortSignal): Promise<{ data: any[]; total: number }> {
  const params = new URLSearchParams({
    limit: filters.pageSize.toString(),
    offset: (filters.page * filters.pageSize).toString(),
    search: filters.search || '',
    sort_by: filters.sortBy || 'created_at',
    sort_dir: filters.sortDir || 'desc',
  });
  if (filters.status) params.set('status', filters.status);
  if (filters.supplier_id) params.set('supplier_id', filters.supplier_id);
  if (filters.startDate) params.set('start_date', filters.startDate);
  if (filters.endDate) params.set('end_date', filters.endDate);

  const res = await apiFetch(`/api/purchase-orders?${params.toString()}`, { signal });
  if (res.ok) {
    const data = await res.json();
    return { data: data.data || [], total: data.total || 0 };
  }
  return { data: [], total: 0 };
}

export async function getPurchaseOrderById(id: number): Promise<any | null> {
  try {
    const res = await apiFetch(`/api/purchase-orders/${id}`);
    if (res.ok) {
      const json = await res.json();
      return json.data ?? json;
    }
    return null;
  } catch {
    return null;
  }
}

export async function createPurchaseOrder(po: any): Promise<any | null> {
  try {
    const res = await apiFetch('/api/purchase-orders', {
      method: 'POST',
      body: JSON.stringify(po),
    });
    if (res.ok) {
      return await res.json();
    }
    const err = await res.json();
    throw new Error(err.error?.message || err.message || 'Failed to create purchase order');
  } catch (e) {
    throw e;
  }
}

export async function updatePurchaseOrder(id: number, po: any): Promise<any | null> {
  try {
    const res = await apiFetch(`/api/purchase-orders/${id}`, {
      method: 'PUT',
      body: JSON.stringify(po),
    });
    if (res.ok) {
      return await res.json();
    }
    const err = await res.json();
    throw new Error(err.message || 'Failed to update purchase order');
  } catch (e) {
    throw e;
  }
}

export async function confirmPurchaseOrder(id: number): Promise<any | null> {
  try {
    const res = await apiFetch(`/api/purchase-orders/${id}/confirm`, { method: 'POST' });
    if (res.ok) {
      return await res.json();
    }
    const err = await res.json();
    throw new Error(err.error?.message || err.message || 'Failed to confirm purchase order');
  } catch (e) {
    throw e;
  }
}

export async function getReceipts(poId: number): Promise<any[]> {
  try {
    const res = await apiFetch(`/api/purchase-orders/${poId}/receipts`);
    if (res.ok) {
      const data = await res.json();
      return data.data || [];
    }
    return [];
  } catch {
    return [];
  }
}

export async function createGoodsReceipt(gr: any): Promise<any | null> {
  try {
    const res = await apiFetch('/api/goods-receipts', {
      method: 'POST',
      body: JSON.stringify(gr),
    });
    if (res.ok) {
      return await res.json();
    }
    const body = await res.json();
    const msg = body?.error?.message || body?.message || body?.error || 'Failed to create goods receipt';
    throw new Error(msg);
  } catch (e) {
    throw e;
  }
}
