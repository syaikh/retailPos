import { apiFetch } from "$shared/api/http-client";
import type {
  Supplier,
  CreateSupplierPayload,
  UpdateSupplierPayload,
  ProductSupplier,
  SupplierUsage,
} from "../types";

export interface SupplierWriteResult {
  ok: boolean;
  status: number;
  code?: string;
  usage?: SupplierUsage;
  blocked_ids?: number[];
  updated?: number;
  deleted?: number;
}

async function toWriteResult(r: Response): Promise<SupplierWriteResult> {
  let body: Record<string, unknown> = {};
  try {
    body = (await r.json()) as Record<string, unknown>;
  } catch {
    body = {};
  }
  return {
    ok: r.ok,
    status: r.status,
    code: typeof body.code === "string" ? body.code : undefined,
    usage: (body.usage as SupplierUsage | undefined) ?? undefined,
    blocked_ids: Array.isArray(body.blocked_ids)
      ? (body.blocked_ids as number[])
      : undefined,
    updated: typeof body.updated === "number" ? body.updated : undefined,
    deleted: typeof body.deleted === "number" ? body.deleted : undefined,
  };
}

export interface SupplierListParams {
  limit: number;
  offset: number;
  search?: string;
  is_active?: boolean;
  is_consignment?: boolean;
  sort_by?: string;
  sort_dir?: string;
}

export interface SupplierListResponse {
  data: Supplier[];
  total: number;
}

export async function getSuppliers(
  params: SupplierListParams,
): Promise<SupplierListResponse> {
  const urlParams = new URLSearchParams({
    limit: params.limit.toString(),
    offset: (params.offset ?? 0).toString(),
  });
  if (params.search) urlParams.append("search", params.search);
  if (params.is_active !== undefined)
    urlParams.append("is_active", params.is_active.toString());
  if (params.is_consignment !== undefined)
    urlParams.append("is_consignment", params.is_consignment.toString());
  if (params.sort_by) urlParams.append("sort_by", params.sort_by);
  if (params.sort_dir) urlParams.append("sort_dir", params.sort_dir);

  const r = await apiFetch(`/api/suppliers?${urlParams.toString()}`);
  if (r.ok) {
    const data = await r.json();
    return { data: data.data || [], total: data.total || 0 };
  }
  return { data: [], total: 0 };
}

export async function getSupplier(id: number): Promise<Supplier | null> {
  const r = await apiFetch(`/api/suppliers/${id}`);
  if (r.ok) {
    const data = await r.json();
    return data.data || null;
  }
  return null;
}

export async function createSupplier(
  payload: CreateSupplierPayload,
): Promise<boolean> {
  const r = await apiFetch("/api/suppliers", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function updateSupplier(
  id: number,
  payload: UpdateSupplierPayload,
): Promise<SupplierWriteResult> {
  const r = await apiFetch(`/api/suppliers/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
  return toWriteResult(r);
}

export async function deleteSupplier(id: number): Promise<SupplierWriteResult> {
  const r = await apiFetch(`/api/suppliers/${id}`, { method: "DELETE" });
  return toWriteResult(r);
}

export async function getSupplierUsage(
  id: number,
): Promise<SupplierUsage | null> {
  const r = await apiFetch(`/api/suppliers/${id}/usage`);
  if (r.ok) {
    try {
      const data = await r.json();
      return data.data || null;
    } catch {
      return null;
    }
  }
  return null;
}

export async function getSuppliersByProduct(
  productId: number,
): Promise<ProductSupplier[]> {
  const r = await apiFetch(`/api/products/${productId}/suppliers`);
  if (r.ok) {
    const data = await r.json();
    return data.data || [];
  }
  return [];
}

export async function getProductsBySupplier(
  supplierId: number,
): Promise<ProductSupplier[]> {
  const r = await apiFetch(`/api/suppliers/${supplierId}/products`);
  if (r.ok) {
    const data = await r.json();
    return data.data || [];
  }
  return [];
}

export async function linkProduct(
  supplierId: number,
  payload: {
    product_id: number;
    unit_cost: number;
    lead_time_days: number;
    is_preferred: boolean;
  },
): Promise<boolean> {
  const r = await apiFetch(`/api/suppliers/${supplierId}/products`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function unlinkProduct(
  supplierId: number,
  productId: number,
): Promise<boolean> {
  const r = await apiFetch(
    `/api/suppliers/${supplierId}/products/${productId}`,
    { method: "DELETE" },
  );
  return r.ok;
}

export async function bulkUpdateSuppliers(
  ids: number[],
  isActive: boolean,
): Promise<SupplierWriteResult> {
  const r = await apiFetch("/api/suppliers/bulk", {
    method: "PUT",
    body: JSON.stringify({ ids, is_active: isActive }),
  });
  return toWriteResult(r);
}

export async function bulkDeleteSuppliers(
  ids: number[],
): Promise<SupplierWriteResult> {
  const r = await apiFetch("/api/suppliers/bulk", {
    method: "DELETE",
    body: JSON.stringify({ ids }),
  });
  return toWriteResult(r);
}
