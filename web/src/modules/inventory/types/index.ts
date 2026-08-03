export interface StockAdjustment {
  product_id: number;
  quantity_change: number;
  notes: string;
}

export interface StockThreshold {
  warning: number;
  critical: number;
}

export interface LocationStockItem {
  product_id: number;
  sku: string;
  name: string;
  location_id: number;
  location_code: string;
  location_name: string;
  quantity: number;
}

export interface SetLocationStockPayload {
  product_id: number;
  location_id: number;
  quantity: number;
}

export interface TransferLocationStockPayload {
  product_id: number;
  from_location_id: number;
  to_location_id: number;
  quantity: number;
}
