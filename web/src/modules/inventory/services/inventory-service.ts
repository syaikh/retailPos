import apiClient from '$shared/api/http-client';
import type { StockAdjustment, StockThreshold } from '../types';

export async function adjustStock(payload: StockAdjustment): Promise<void> {
  await apiClient.post('/inventory/adjust', payload);
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
