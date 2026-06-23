export interface ChartDataPoint {
  hour?: number;
  date?: string;
  month_start?: string;
  week_start?: string;
  week_end?: string;
  total: number;
  label?: string;
  order_count?: number;
}

export interface KpiData {
  totalRevenue: number;
  previousRevenue: number;
  totalOrders: number;
  previousOrders: number;
  avgOrderValue: number;
  previousAvgOrderValue: number;
  revenuePerDay: number;
  previousRevenuePerDay: number;
  peakRevenueHour: number | null;
  previousPeakRevenue: number | null;
  peakRevenueMonth: number | null;
  previousPeakRevenueMonth: number | null;
  percentChange: number;
  comparisonType: string;
  isPartial: boolean;
  periodInfo: Record<string, unknown> | null;
}

export interface PeriodOption {
  value: string;
  label: string;
  icon: string;
  description?: string;
  type?: 'separator';
}

export interface ComparisonData {
  current_revenue: number;
  previous_revenue: number;
  current_orders: number;
  previous_orders: number;
  current_aov: number;
  previous_aov: number;
  revenue_per_day: number;
  previous_revenue_per_day: number;
  peak_revenue_hour: number | null;
  previous_peak_revenue: number | null;
  peak_revenue_month: number | null;
  previous_peak_revenue_month: number | null;
}
