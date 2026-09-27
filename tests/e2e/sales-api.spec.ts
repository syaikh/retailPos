import { test, expect } from "./fixtures";
import {
  TEST_USERS,
  API_BASE,
  authHeader,
  getToken as cachedGetToken,
} from "./fixtures";

const getToken = cachedGetToken;

test.describe("Sales API - Get By ID", () => {
  async function createTestSale(request: any, token: string) {
    const productsRes = await request.get(`${API_BASE}/api/products?limit=1`, {
      headers: authHeader(token),
    });
    const productsBody = await productsRes.json();
    const product = (productsBody.data || [])[0];
    if (!product) return null;

    const res = await request.post(`${API_BASE}/api/sales`, {
      headers: authHeader(token),
      data: {
        items: [{ product_id: product.id, quantity: 1 }],
        payment_method: "cash",
      },
    });
    if (!res.ok()) return null;
    return await res.json();
  }

  test("GET /api/sales/:id returns sale details", async ({ request }) => {
    const token = await getToken(request);
    const listRes = await request.get(`${API_BASE}/api/sales?limit=1`, {
      headers: authHeader(token),
    });
    expect(listRes.ok()).toBeTruthy();
    const listBody = await listRes.json();
    const sales = listBody.data || [];
    if (sales.length === 0) return;

    const saleId = sales[0].id;
    const res = await request.get(`${API_BASE}/api/sales/${saleId}`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.id).toBe(saleId);
    expect(body.data.invoice_number).toBeTruthy();
    expect(body.data.total_amount).toBeDefined();
    expect(body.data.payment_method).toBeTruthy();
    expect(body.data.status).toBeTruthy();
  });

  test("GET /api/sales/:id with invalid id returns 400", async ({
    request,
  }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/sales/abc`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(400);
  });

  test("GET /api/sales/:id with nonexistent id returns 404", async ({
    request,
  }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/sales/99999999`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(404);
  });

  test("GET /api/sales/:id without auth returns 401", async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/sales/1`);
    expect(res.status()).toBe(401);
  });
});

test.describe("Sales API - Export", () => {
  test("GET /api/sales/export returns CSV with expected headers", async ({
    request,
  }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/sales/export?format=csv`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const text = await res.text();
    expect(text).toContain("Invoice Number");
    expect(text).toContain("Total Amount");
  });

  test("GET /api/sales/export without auth returns 401", async ({
    request,
  }) => {
    const res = await request.get(`${API_BASE}/api/sales/export`);
    expect(res.status()).toBe(401);
  });

  test("GET /api/sales/export with date filter returns filtered results", async ({
    request,
  }) => {
    const token = await getToken(request);
    const res = await request.get(
      `${API_BASE}/api/sales/export?format=csv&start_date=2020-01-01&end_date=2020-01-02`,
      { headers: authHeader(token) },
    );
    expect(res.ok()).toBeTruthy();
    const text = await res.text();
    const lines = text.trim().split("\n");
    expect(lines.length).toBe(1);
  });
});

test.describe("Sales API - Keyset Pagination", () => {
  const cursorParams = (nextCursor: string, limit = 2) => {
    const sep = nextCursor.indexOf("|");
    const p = new URLSearchParams({
      after_created_at: nextCursor.slice(0, sep),
      after_id: nextCursor.slice(sep + 1),
      limit: String(limit),
      offset: "0",
    });
    return p.toString();
  };

  test("GET /api/sales returns has_more and next_cursor envelope", async ({
    request,
  }) => {
    const token = await getToken(request);
    const res = await request.get(`${API_BASE}/api/sales?limit=2&offset=0`, {
      headers: authHeader(token),
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(typeof body.total).toBe("number");
    expect(typeof body.has_more).toBe("boolean");
    if (body.total > 2) {
      expect(body.has_more).toBe(true);
      expect(body.next_cursor).toBeTruthy();
      expect(body.next_cursor).toContain("|");
    } else {
      expect(body.next_cursor).toBeUndefined();
    }
  });

  test("cursor round-trip pages forward without overlapping rows", async ({
    request,
  }) => {
    const token = await getToken(request);
    const page1Res = await request.get(
      `${API_BASE}/api/sales?limit=2&offset=0`,
      {
        headers: authHeader(token),
      },
    );
    expect(page1Res.ok()).toBeTruthy();
    const page1 = await page1Res.json();
    const sales1 = page1.data || [];
    if (sales1.length < 2 || !page1.next_cursor) return;

    const page2Res = await request.get(
      `${API_BASE}/api/sales?${cursorParams(page1.next_cursor)}`,
      { headers: authHeader(token) },
    );
    expect(page2Res.ok()).toBeTruthy();
    const page2 = await page2Res.json();
    const sales2 = page2.data || [];
    expect(sales2.length).toBeGreaterThan(0);

    const ids1 = new Set(sales1.map((s: { id: number }) => s.id));
    for (const s of sales2 as Array<{ id: number }>) {
      expect(ids1.has(s.id)).toBe(false);
    }
    expect(page2.offset).toBe(0);
  });

  test("cursor with non-default sort returns 400", async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(
      `${API_BASE}/api/sales?after_created_at=2026-01-01T00:00:00Z&after_id=1&sort_by=invoice_number&sort_dir=ASC`,
      { headers: authHeader(token) },
    );
    expect(res.status()).toBe(400);
  });

  test("malformed cursor returns 400", async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(
      `${API_BASE}/api/sales?after_created_at=not-a-date&after_id=1`,
      { headers: authHeader(token) },
    );
    expect(res.status()).toBe(400);
  });

  test("after_created_at without after_id returns 400", async ({ request }) => {
    const token = await getToken(request);
    const res = await request.get(
      `${API_BASE}/api/sales?after_created_at=2026-01-01T00:00:00Z`,
      { headers: authHeader(token) },
    );
    expect(res.status()).toBe(400);
  });
});
