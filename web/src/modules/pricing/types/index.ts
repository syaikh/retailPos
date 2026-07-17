export type PricingMethod = 'fixed_price' | 'discount_percent' | 'discount_amount' | 'markup_percent';

export type PricingType = 'special_price' | 'promotion';

export interface PricingRule {
  id: number;
  product_id?: number;
  category_id?: number;
  brand_id?: number;
  pricing_type: PricingType;
  pricing_method: PricingMethod;
  pricing_value: number;
  name: string;
  minimum_quantity: number;
  maximum_quantity?: number;
  priority: number;
  customer_group_id?: number;
  store_id?: number;
  recurrence_days?: string[];
  time_from?: string;
  time_to?: string;
  allow_combine: boolean;
  is_active: boolean;
  effective_from?: string;
  effective_until?: string;
  created_at?: string;
  updated_at?: string;
}

export interface CreatePricingRulePayload {
  product_id?: number;
  category_id?: number;
  brand_id?: number;
  pricing_type: PricingType;
  pricing_method: PricingMethod;
  pricing_value: number;
  name: string;
  minimum_quantity: number;
  maximum_quantity?: number;
  priority: number;
  customer_group_id?: number;
  store_id?: number;
  recurrence_days?: string[];
  time_from?: string;
  time_to?: string;
  allow_combine?: boolean;
  is_active: boolean;
  effective_from?: string;
  effective_until?: string;
}

export interface UpdatePricingRulePayload {
  product_id?: number;
  category_id?: number;
  brand_id?: number;
  pricing_type?: PricingType;
  pricing_method?: PricingMethod;
  pricing_value?: number;
  name?: string;
  minimum_quantity?: number;
  maximum_quantity?: number;
  priority?: number;
  customer_group_id?: number;
  store_id?: number;
  recurrence_days?: string[];
  time_from?: string;
  time_to?: string;
  allow_combine?: boolean;
  is_active?: boolean;
  effective_from?: string;
  effective_until?: string;
}

export interface CustomerGroup {
  id: number;
  name: string;
  description?: string;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface Store {
  id: number;
  name: string;
  address?: string;
  phone?: string;
  is_active: boolean;
  created_at?: string;
}

export interface ProductSearchResult {
  id: number;
  name: string;
  sku: string;
  price: number;
}
