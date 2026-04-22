import client from './client';
import type { SaleRequest, SaleResponse } from '$lib/domain/entities';

export const saleApi = {
	/**
	 * Create a new sale (checkout)
	 */
	checkout: (data: SaleRequest) =>
		client.post<SaleResponse>('/sales', data),

	/**
	 * Get all transactions with filters
	 */
	getTransactions: (params?: { start_date?: string; end_date?: string; limit?: number; offset?: number }) => {
		const query = new URLSearchParams();
		if (params?.start_date) query.append('start_date', params.start_date);
		if (params?.end_date) query.append('end_date', params.end_date);
		if (params?.limit) query.append('limit', params.limit.toString());
		if (params?.offset) query.append('offset', params.offset.toString());
		const qs = query.toString();
		return client.get<{ data: SaleResponse[]; total: number }>(`/sales?${qs}`);
	},

	/**
	 * Get sales chart data aggregated by period
	 */
	getChartData: (period: 'daily' | 'weekly' | 'monthly') =>
		client.get<{ labels: string[]; data: number[] }>(`/sales/chart?period=${period}`)
};
