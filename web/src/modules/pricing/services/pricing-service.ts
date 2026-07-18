import { apiFetch } from '$shared/api/http-client';
import type { PricingRule, CreatePricingRulePayload, UpdatePricingRulePayload, ProductSearchResult } from '../types';

export interface PricingRuleListParams {
  limit: number;
  offset: number;
  search?: string;
  product_id?: number;
  pricing_type?: string;
  pricing_method?: string;
  category_id?: number;
  brand_id?: number;
  customer_group_id?: number;
  store_id?: number;
  is_active?: boolean;
  status?: string;
  sort_by?: string;
  sort_dir?: string;
}

export interface PricingRuleListResponse {
  data: PricingRule[];
  total: number;
}

export async function getPricingRules(params: PricingRuleListParams): Promise<PricingRuleListResponse> {
  const urlParams = new URLSearchParams({
    limit: params.limit.toString(),
    offset: params.offset.toString(),
  });
  if (params.search) urlParams.append('search', params.search);
  if (params.product_id) urlParams.append('product_id', params.product_id.toString());
  if (params.pricing_type) urlParams.append('pricing_type', params.pricing_type);
  if (params.pricing_method) urlParams.append('pricing_method', params.pricing_method);
  if (params.category_id) urlParams.append('category_id', params.category_id.toString());
  if (params.brand_id) urlParams.append('brand_id', params.brand_id.toString());
  if (params.customer_group_id) urlParams.append('customer_group_id', params.customer_group_id.toString());
  if (params.store_id) urlParams.append('store_id', params.store_id.toString());
  if (params.is_active !== undefined) urlParams.append('is_active', params.is_active.toString());
  if (params.status) urlParams.append('status', params.status);
  if (params.sort_by) urlParams.append('sort_by', params.sort_by);
  if (params.sort_dir) urlParams.append('sort_dir', params.sort_dir);

  const r = await apiFetch(`/api/pricing-rules?${urlParams.toString()}`);
  if (r.ok) {
    const data = await r.json();
    return { data: data.data || [], total: data.total || 0 };
  }
  return { data: [], total: 0 };
}

export async function getPricingRule(id: number): Promise<PricingRule | null> {
  const r = await apiFetch(`/api/pricing-rules/${id}`);
  if (r.ok) {
    const data = await r.json();
    return data.data || null;
  }
  return null;
}

export async function createPricingRule(payload: CreatePricingRulePayload): Promise<boolean> {
  const r = await apiFetch('/api/pricing-rules', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function updatePricingRule(id: number, payload: UpdatePricingRulePayload): Promise<boolean> {
  const r = await apiFetch(`/api/pricing-rules/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function deletePricingRule(id: number): Promise<boolean> {
  const r = await apiFetch(`/api/pricing-rules/${id}`, { method: 'DELETE' });
  return r.ok;
}

export async function submitPricingRule(id: number): Promise<boolean> {
  const r = await apiFetch(`/api/pricing-rules/${id}/submit`, { method: 'POST' });
  return r.ok;
}

export async function approvePricingRule(id: number): Promise<boolean> {
  const r = await apiFetch(`/api/pricing-rules/${id}/approve`, { method: 'POST' });
  return r.ok;
}

export async function rejectPricingRule(id: number): Promise<boolean> {
  const r = await apiFetch(`/api/pricing-rules/${id}/reject`, { method: 'POST' });
  return r.ok;
}

export interface ResolveItem {
  product_id: number;
  quantity: number;
  customer_group_id?: number;
  store_id?: number;
}

export interface ResolvedPrice {
  unit_price: number;
  original_price: number;
  discount: number;
  pricing_type: string;
  pricing_method?: string;
  rule?: {
    id: number;
    name: string;
    pricing_type: string;
    pricing_method?: string;
    pricing_value?: number;
  };
}

export async function resolvePrices(items: ResolveItem[]): Promise<ResolvedPrice[]> {
  const r = await apiFetch('/api/pricing/resolve', {
    method: 'POST',
    body: JSON.stringify({ items }),
  });
  if (r.ok) {
    const data = await r.json();
    return data.data || [];
  }
  return [];
}

export async function searchProducts(query: string, limit = 10): Promise<ProductSearchResult[]> {
  const urlParams = new URLSearchParams({ q: query, limit: limit.toString() });
  const r = await apiFetch(`/api/products/search?${urlParams.toString()}`);
  if (r.ok) {
    const data = await r.json();
    return data.data || [];
  }
  return [];
}

export async function getCustomerGroups(): Promise<{ id: number; name: string }[]> {
  const r = await apiFetch('/api/customer-groups?limit=100&is_active=true');
  if (r.ok) {
    const data = await r.json();
    return data.data || [];
  }
  return [];
}

export async function getStores(): Promise<{ id: number; name: string }[]> {
  const r = await apiFetch('/api/stores/active');
  if (r.ok) {
    const data = await r.json();
    return data.data || [];
  }
  return [];
}

export interface CheckConflictsRequest {
  product_id?: number | null;
  category_id?: number | null;
  brand_id?: number | null;
  pricing_type: string;
  pricing_method: string;
  pricing_value: number;
  minimum_quantity: number;
  maximum_quantity?: number | null;
  priority: number;
  exclude_id?: number;
}

export interface ConflictRule {
  id: number;
  name: string;
  pricing_type: string;
  pricing_method: string;
  pricing_value: number;
  priority: number;
  minimum_quantity: number;
  maximum_quantity: number | null;
}

export interface CheckConflictsResponse {
  data: ConflictRule[];
  has_conflicts: boolean;
}

export async function checkConflicts(payload: CheckConflictsRequest): Promise<CheckConflictsResponse> {
  const r = await apiFetch('/api/pricing-rules/check-conflicts', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  if (r.ok) {
    return await r.json();
  }
  return { data: [], has_conflicts: false };
}
