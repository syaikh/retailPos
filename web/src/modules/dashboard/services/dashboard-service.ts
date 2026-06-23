import { apiFetch } from '$shared/api/http-client';
import type { DashboardLiveStats, DashboardData } from '../types';

export async function getLiveStats(): Promise<DashboardData> {
  try {
    const res = await apiFetch('/api/dashboard/live');
    if (res.ok) {
      const data = (await res.json()) as { data?: DashboardLiveStats };
      if (data.data) {
        return {
          todaysRevenue: data.data.todays_revenue || 0,
          todaysSales: data.data.todays_sales || 0,
          totalProducts: data.data.total_products || 0,
          lowStockCount: data.data.low_stock_count || 0,
        };
      }
    }
  } catch {
    // ignore
  }
  return { todaysRevenue: 0, todaysSales: 0, totalProducts: 0, lowStockCount: 0 };
}
