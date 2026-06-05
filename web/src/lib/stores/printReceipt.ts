import { writable } from 'svelte/store';

export interface ReceiptData {
  invoice_number: string;
  created_at: string;
  items: Array<{
    name: string;
    quantity: number;
    unit_price: number;
  }>;
  total_amount: number;
  paymentMethod: string;
  cashReceived: number;
  changeDue: number;
}

export const printReceipt = writable<ReceiptData | null>(null);
