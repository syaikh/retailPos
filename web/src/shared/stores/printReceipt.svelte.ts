import { writable } from 'svelte/store';

export interface ReceiptData {
  invoice_number: string;
  created_at: string;
  items: Array<{
    name: string;
    quantity: number;
    unit_price: number;
    original_price?: number;
    pricing_rule_name?: string;
    pricing_type?: string;
  }>;
  total_amount: number;
  subtotal_dpp?: number;
  tax?: number;
  paymentMethod: string;
  cashReceived: number;
  changeDue: number;
  customer_name?: string;
  total_savings?: number;
}

export const printReceipt = writable<ReceiptData | null>(null);
