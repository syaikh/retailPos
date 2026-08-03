import apiClient from '$shared/api/http-client';
import type { LocationStockItem, SetLocationStockPayload, StockAdjustment, StockThreshold, TransferLocationStockPayload } from '../types';

export async function adjustStock(payload: StockAdjustment): Promise<void> {
  await apiClient.post('/inventory/adjust', payload);
}

export async function getLocationStock(productId: number, locationId?: number): Promise<LocationStockItem[]> {
  const params: Record<string, string | number> = { product_id: productId };
  if (locationId) params.location_id = locationId;
  const r = await apiClient.get('/inventory/locations', { params });
  return r.data.data || [];
}

export async function setLocationStock(payload: SetLocationStockPayload): Promise<void> {
  await apiClient.post('/inventory/locations', payload);
}

export async function transferLocationStock(payload: TransferLocationStockPayload): Promise<void> {
  await apiClient.post('/inventory/locations/transfer', payload);
}

export async function getStockThresholds(): Promise<StockThreshold> {
  try {
    const r = await apiClient.get('/stock-thresholds');
    return {
      warning: r.data.warning ?? 10,
      critical: r.data.critical ?? 5,
    };
  } catch {
    return { warning: 10, critical: 5 };
  }
}
