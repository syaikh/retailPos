import { test, expect } from "./fixtures";
import { apiAs, ApiDriver } from "./api-driver";

/**
 * Supplier master-data governance, driven at the API layer.
 *
 * Pins the contract the UI depends on: soft-delete and deactivate are guarded,
 * edits are version-checked, a freed code can be reused, and the cross-module
 * usage breakdown is reported with the documented `code` discriminators. The
 * genuine UI (409 dialogs, stale-edit prompt, usage panel) stays in the
 * suppliers browser specs.
 */
const data = (body: any) =>
  body && body.data !== undefined ? body.data : body;
const firstOf = (body: any) =>
  Array.isArray(data(body)) ? data(body)[0] : data(body);

const uniq = () => `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;

async function createSupplier(api: ApiDriver, code = `E2E-GOV-${uniq()}`) {
  const res = await api.post("/api/suppliers", {
    name: `E2E Gov ${code}`,
    code,
    is_active: true,
  });
  expect(res.status).toBe(201);
  return data(res.body);
}

async function anyProductId(api: ApiDriver): Promise<number> {
  const product = firstOf((await api.get("/api/products?limit=1")).body);
  if (product?.id) return product.id;
  const created = await api.post("/api/products", {
    name: `E2E Gov Product ${uniq()}`,
    sku: `E2E-GOV-${uniq()}`,
    price: 10000,
    cost: 5000,
    stock: 10,
    status: "active",
  });
  return data(created.body).id;
}

test.describe("Supplier governance (API driver)", () => {
  test("rejects a stale version and accepts the current one", async ({
    request,
  }) => {
    const api = await apiAs(request, "superadmin");
    const supplier = await createSupplier(api);
    expect(supplier.version).toBe(1);

    const first = await api.put(`/api/suppliers/${supplier.id}`, {
      name: supplier.name,
      code: supplier.code,
      is_active: true,
      version: supplier.version,
    });
    expect(first.ok).toBeTruthy();
    expect(data(first.body).version).toBe(2);

    const stale = await api.put(`/api/suppliers/${supplier.id}`, {
      name: "Stale rename",
      code: supplier.code,
      is_active: true,
      version: supplier.version,
    });
    expect(stale.status).toBe(409);
    expect(stale.body.code).toBe("supplier_version_conflict");

    const retry = await api.put(`/api/suppliers/${supplier.id}`, {
      name: "Reapplied rename",
      code: supplier.code,
      is_active: true,
      version: 2,
    });
    expect(retry.ok).toBeTruthy();
    expect(data(retry.body).version).toBe(3);
  });

  test("frees a code once the supplier is soft-deleted", async ({
    request,
  }) => {
    const api = await apiAs(request, "superadmin");
    const code = `E2E-GOV-${uniq()}`;
    const original = await createSupplier(api, code);

    const dup = await api.post("/api/suppliers", {
      name: "Live duplicate",
      code,
      is_active: true,
    });
    expect(dup.ok).toBeFalsy();

    expect((await api.del(`/api/suppliers/${original.id}`)).status).toBe(200);

    const reused = await createSupplier(api, code);
    expect(reused.id).not.toBe(original.id);
    expect(reused.code).toBe(code);
  });

  test("refuses to delete a linked supplier, then succeeds after unlink", async ({
    request,
  }) => {
    const api = await apiAs(request, "superadmin");
    const supplier = await createSupplier(api);
    const productId = await anyProductId(api);

    const link = await api.post(`/api/suppliers/${supplier.id}/products`, {
      product_id: productId,
      unit_cost: 5000,
      lead_time_days: 7,
      is_preferred: false,
    });
    expect(link.ok).toBeTruthy();

    const blocked = await api.del(`/api/suppliers/${supplier.id}`);
    expect(blocked.status).toBe(409);
    expect(blocked.body.code).toBe("supplier_in_use");
    expect(blocked.body.usage.product_links).toBeGreaterThanOrEqual(1);

    expect(
      (await api.del(`/api/suppliers/${supplier.id}/products/${productId}`)).ok,
    ).toBeTruthy();
    expect((await api.del(`/api/suppliers/${supplier.id}`)).status).toBe(200);
  });

  test("reports the cross-module usage breakdown", async ({ request }) => {
    const api = await apiAs(request, "superadmin");
    const supplier = await createSupplier(api);
    const productId = await anyProductId(api);
    expect(
      (
        await api.post(`/api/suppliers/${supplier.id}/products`, {
          product_id: productId,
          unit_cost: 5000,
          is_preferred: false,
        })
      ).ok,
    ).toBeTruthy();

    const usage = data(
      (await api.get(`/api/suppliers/${supplier.id}/usage`)).body,
    );
    expect(usage.product_links).toBeGreaterThanOrEqual(1);
    expect(usage.open_purchase_orders).toBe(0);
    expect(usage.active_consignments).toBe(0);

    await api.del(`/api/suppliers/${supplier.id}/products/${productId}`);
  });

  test("refuses to deactivate a supplier with an open purchase order, then allows it after cancel", async ({
    request,
  }) => {
    const api = await apiAs(request, "superadmin");
    const supplier = await createSupplier(api);
    const productId = await anyProductId(api);
    const store = firstOf((await api.get("/api/stores/active")).body);
    expect(store?.id).toBeTruthy();

    const po = data(
      (
        await api.post("/api/purchase-orders", {
          supplier_id: supplier.id,
          store_id: store.id,
          items: [{ product_id: productId, qty_ordered: 5, unit_cost: 1000 }],
        })
      ).body,
    );
    expect(po.status).toBe("draft");

    const blocked = await api.put(`/api/suppliers/${supplier.id}`, {
      name: supplier.name,
      code: supplier.code,
      is_active: false,
      version: supplier.version,
    });
    expect(blocked.status).toBe(409);
    expect(blocked.body.code).toBe("supplier_in_use");
    expect(blocked.body.usage.open_purchase_orders).toBeGreaterThanOrEqual(1);

    expect(
      (await api.post(`/api/purchase-orders/${po.id}/cancel`, {})).ok,
    ).toBeTruthy();

    const allowed = await api.put(`/api/suppliers/${supplier.id}`, {
      name: supplier.name,
      code: supplier.code,
      is_active: false,
      version: supplier.version,
    });
    expect(allowed.ok).toBeTruthy();
    expect(data(allowed.body).is_active).toBe(false);
  });
});
