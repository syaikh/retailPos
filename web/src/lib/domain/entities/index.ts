export interface Product {
	id: number;
	sku: string;
	barcode: string | null;
	name: string;
	price: number;
	stock: number;
	created_at: string;
	updated_at: string;
}

export interface CartItem {
	id: number;
	sku: string;
	barcode: string | null;
	name: string;
	price: number;
	quantity: number;
	stock: number; // original stock at time of adding
}

export interface SaleItem {
	product_id: number;
	quantity: number;
	price_at_sale: number;
}

export interface SaleRequest {
	total_amount: number;
	payment_method: 'cash' | 'card';
	items: SaleItem[];
}

export interface SaleResponse {
	id: number;
	transaction_code: string;
	total_amount: number;
	payment_method: string;
	items: SaleItem[];
	created_at: string;
}

export interface ProductPage {
	data: Product[];
	total: number;
	limit: number;
	offset: number;
}
