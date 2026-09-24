import { test, expect } from './fixtures';
import { API_BASE, authHeader, getToken, loginUI, logoutUI, waitForAppReady, FRONTEND_BASE } from './fixtures';
import { TestDataTracker, execSQL, querySQL } from './db-helper';

/**
 * Storage locations — genuine UI behaviour only: the store-scoped list must
 * render the caller's own rows (both store-linked and warehouse-linked) and
 * must not surface rows from another store. All CRUD/boundary behaviour is
 * covered by storage-locations-api.spec.ts.
 *
 * Hermetic setup: a fresh store + store-scoped manager (role 2), one location
 * directly in that store, one location through a warehouse linked to the
 * store, and one location in a different store. The search box drives the
 * scoped list request, so the assertions are stable regardless of pagination.
 */
test.describe('Storage locations - store-scoped list (UI)', () => {
  const tracker = new TestDataTracker();
  const suffix = Date.now();
  const password = 'e2eSlUi123';

  let managerCreds: { username: string; password: string };
  let ownStoreCode: string;
  let ownWarehouseCode: string;
  let foreignCode: string;

  test.beforeAll(async ({ request }) => {
    const super0Token = await getToken(request);
    const super0Headers = authHeader(super0Token);

    const ownStoreName = `E2E SL UI Own ${suffix}`;
    const foreignStoreName = `E2E SL UI Foreign ${suffix}`;

    const ownStoreRes = await request.post(`${API_BASE}/api/stores`, {
      headers: super0Headers,
      data: { name: ownStoreName },
    });
    expect(
      ownStoreRes.ok(),
      `create own store failed: ${ownStoreRes.status()} ${await ownStoreRes.text()}`
    ).toBeTruthy();
    const ownStore = (await ownStoreRes.json()).data;
    tracker.trackStore(ownStore.id);

    const foreignStoreRes = await request.post(`${API_BASE}/api/stores`, {
      headers: super0Headers,
      data: { name: foreignStoreName },
    });
    expect(
      foreignStoreRes.ok(),
      `create foreign store failed: ${foreignStoreRes.status()} ${await foreignStoreRes.text()}`
    ).toBeTruthy();
    const foreignStore = (await foreignStoreRes.json()).data;
    tracker.trackStore(foreignStore.id);

    // Warehouse linked to the own store (no warehouse POST API — SQL).
    // warehouses.code is varchar(20): prefix + 13-digit timestamp must fit.
    const whCode = `UIWH${suffix}`;
    execSQL(
      `INSERT INTO warehouses (name, code, store_id) VALUES ('E2E SL UI WH ${suffix}', '${whCode}', ${ownStore.id})`
    );
    const whId = Number(
      querySQL<{ id: number }>(`SELECT id FROM warehouses WHERE code = '${whCode}'`)[0].id
    );
    tracker.trackWarehouse(whId);

    // Store-scoped manager for the own store.
    const username = `e2eslui${suffix}`;
    const createUserRes = await request.post(`${API_BASE}/api/admin/users`, {
      headers: super0Headers,
      data: {
        username,
        email: `${username}@retail-pos.local`,
        password,
        role_id: 2,
        store_id: ownStore.id,
      },
    });
    expect(
      createUserRes.ok(),
      `create manager failed: ${createUserRes.status()} ${await createUserRes.text()}`
    ).toBeTruthy();
    tracker.trackUser((await createUserRes.json()).data?.id);
    managerCreds = { username, password };

    // One location per scope shape.
    ownStoreCode = `UIS${suffix}`;
    ownWarehouseCode = `UIW${suffix}`;
    foreignCode = `UIF${suffix}`;
    const mkLoc = async (code: string, scope: Record<string, number>) => {
      const res = await request.post(`${API_BASE}/api/storage-locations`, {
        headers: super0Headers,
        data: { code, name: `E2E UI ${code}`, ...scope },
      });
      expect(
        res.ok(),
        `create location ${code} failed: ${res.status()} ${await res.text()}`
      ).toBeTruthy();
      tracker.trackLocation((await res.json()).data.id);
    };
    await mkLoc(ownStoreCode, { store_id: ownStore.id });
    await mkLoc(ownWarehouseCode, { warehouse_id: whId });
    await mkLoc(foreignCode, { store_id: foreignStore.id });
  });

  test.afterAll(() => tracker.cleanup());

  test('list shows own store and warehouse rows but hides foreign rows', async ({ page }) => {
    await loginUI(page, managerCreds.username, managerCreds.password);
    await page.goto(`${FRONTEND_BASE}/storage-locations`);
    await waitForAppReady(page);

    const search = page.locator('#storage-location-search');
    await expect(search).toBeVisible({ timeout: 10000 });

    // Own store-linked row is findable.
    await search.fill(ownStoreCode);
    await expect(page.getByText(ownStoreCode, { exact: true })).toBeVisible({ timeout: 5000 });

    // Own warehouse-linked row is findable (list must inherit the store
    // boundary through warehouses.store_id, not only sl.store_id).
    await search.fill(ownWarehouseCode);
    await expect(page.getByText(ownWarehouseCode, { exact: true })).toBeVisible({ timeout: 5000 });

    // Foreign-store row never surfaces for this caller.
    await search.fill(foreignCode);
    await expect(page.getByText('No locations found')).toBeVisible({ timeout: 5000 });

    await logoutUI(page);
  });
});
