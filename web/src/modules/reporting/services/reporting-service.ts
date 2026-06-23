import { apiFetch } from '$shared/api/http-client';
import type { ChartDataPoint, ComparisonData } from '../types';

export interface ChartResponse {
  current: ChartDataPoint[];
  previous: ChartDataPoint[];
}

export interface ComparisonResponse {
  data: ComparisonData;
  meta: {
    is_partial: boolean;
    current_period?: { start: string; end: string };
    previous_period?: { start: string; end: string };
  };
}

export async function getAvailableYears(): Promise<number[]> {
  try {
    const res = await apiFetch('/api/dashboard/years');
    if (res.ok) {
      const data = await res.json();
      return data.data || [];
    }
    return [];
  } catch {
    return [];
  }
}

export async function getChartData(
  endpoint: string,
  startDate: string,
  endDate: string,
  prevStart?: string,
  prevEnd?: string,
): Promise<ChartResponse | null> {
  const url = `${endpoint}?startDate=${startDate}&endDate=${endDate}${prevStart && prevEnd ? `&prevStart=${prevStart}&prevEnd=${prevEnd}` : ''}`;
  const res = await apiFetch(url);
  if (res.ok) {
    const data = await res.json();
    return {
      current: data.current || data.data || [],
      previous: data.previous || [],
    };
  }
  return null;
}

export async function getComparison(
  period: string,
  mode: string,
  date: string,
): Promise<ComparisonResponse | null> {
  const res = await apiFetch(`/api/dashboard/comparison?period=${period}&mode=${mode}&date=${date}`);
  if (res.ok) {
    return await res.json();
  }
  return null;
}

export async function exportDashboard(
  period: string,
  mode: string,
  date: string,
  chartImage?: string,
): Promise<Blob | null> {
  const formData = new FormData();
  if (period) formData.set('period', period);
  if (mode) formData.set('mode', mode);
  if (date) formData.set('date', date);
  if (chartImage) formData.set('chartData', chartImage);

  const res = await apiFetch('/api/dashboard/export', {
    method: 'POST',
    body: formData,
  });
  if (!res.ok) return null;
  return res.blob();
}
