import { apiFetch } from '$shared/api/http-client';
import type { PricingRule, CreatePricingRulePayload, UpdatePricingRulePayload } from '../types';

export interface PricingRuleListParams {
  limit: number;
  offset: number;
  search?: string;
  product_id?: number;
  pricing_type?: string;
  is_active?: boolean;
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
  if (params.is_active !== undefined) urlParams.append('is_active', params.is_active.toString());

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

export interface ResolveItem {
  product_id: number;
  quantity: number;
}

export interface ResolvedPrice {
  unit_price: number;
  original_price: number;
  discount: number;
  pricing_type: string;
  rule?: {
    id: number;
    name: string;
    pricing_type: string;
    price: number;
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
