export interface StockAdjustment {
  product_id: number;
  quantity_change: number;
  notes: string;
}

export interface StockThreshold {
  warning: number;
  critical: number;
}
