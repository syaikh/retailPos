export interface PurchaseOrderItem {
  id: number;
  purchase_order_id: number;
  product_id: number;
  qty_ordered: number;
  qty_received: number;
  unit_cost: number;
  discount_amount: number;
  subtotal: number;
  product_name: string;
  sku?: string;
  barcode?: string;
  uom_id?: number;
  uom_name?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface PurchaseOrder {
  id: number;
  po_number: string;
  supplier_id: number;
  store_id: number;
  warehouse_id?: number;
  status: string;
  expected_date?: string;
  payment_term?: string;
  delivery_address?: string;
  supplier_reference_number?: string;
  approval_status?: string;
  payment_status?: string;
  invoice_status?: string;
  currency_code?: string;
  exchange_rate?: number;
  approved_by?: number;
  approved_at?: string;
  subtotal: number;
  discount_amount: number;
  tax_amount: number;
  grand_total: number;
  notes?: string;
  confirmed_at?: string;
  confirmed_by?: number;
  cancelled_at?: string;
  cancelled_by?: number;
  created_by: number;
  updated_by: number;
  created_at: string;
  updated_at: string;
  items: PurchaseOrderItem[];
}

export interface GoodsReceiptItem {
  id: number;
  goods_receipt_id: number;
  purchase_order_item_id: number;
  product_id: number;
  qty_good: number;
  qty_damaged: number;
  unit_cost: number;
  product_name: string;
  supplier_id?: number;
  notes?: string;
  created_at: string;
}

export interface GoodsReceipt {
  id: number;
  gr_number: string;
  purchase_order_id: number;
  store_id: number;
  received_by: number;
  received_at: string;
  delivery_order_number?: string;
  shipping_method?: string;
  driver_name?: string;
  vehicle_plate_number?: string;
  notes?: string;
  created_at: string;
  items: GoodsReceiptItem[];
}

export interface CreatePOItemRequest {
  product_id: number;
  qty_ordered: number;
  unit_cost: number;
  discount_amount?: number;
  notes?: string;
}

export interface UpdatePOItemRequest {
  id?: number;
  product_id: number;
  qty_ordered: number;
  unit_cost: number;
  discount_amount?: number;
  notes?: string;
}

export interface CreateGRItemRequest {
  purchase_order_item_id: number;
  qty_good: number;
  qty_damaged?: number;
  notes?: string;
}

export interface PurchaseOrderFilters {
  search?: string;
  status?: string;
  supplier_id?: string;
  startDate?: string;
  endDate?: string;
  page: number;
  pageSize: number;
  sortBy: string;
  sortDir: 'asc' | 'desc';
}
