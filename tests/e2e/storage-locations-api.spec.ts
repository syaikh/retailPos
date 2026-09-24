import { test, expect } from './fixtures';
import { apiAs, loginDriver, type ApiDriver } from './api-driver';
import { TestDataTracker, execSQL, querySQL } from './db-helper';

/**
 * Storage-location store-boundary behaviour (API driver).
 *
 * A location belongs either directly to a store (store_id) or to a warehouse
 * (warehouse_id -> warehouses.store_id). Store-scoped callers may only read
 * and write rows in their own store; warehouses with no store (central) are
 * superadmin-only. Cross-store access returns 403 (not 404), and bulk
 * operations are strict: one foreign id aborts the whole batch with no
 * partial write.
 *
 * Setup (beforeAll) is hermetic: two fresh stores, two store-scoped manager
 * users (role 2 holds storage_location.*), and warehouses created via SQL
 * (there is no warehouse POST API). Per-test drivers are recreated in
 * beforeEach per the token-rule convention; tokens are warmed in beforeAll
 * so per-test logins hit the disk cache instead of the rate limiter.
 */
test.describe('Storage locations store boundary (API)', () => {
  const tracker = new TestDataTracker();
  const suffix = Date.now();
  const password = 'e2eSlPass123';

  let storeA: { id: number };
  let storeB: { id: number };
  let whA: number;
  let whB: number;
  let centralWh: number;
  let locA: { id: number; code: string; name: string };
  let locB: { id: number; code: string; name: string };
  let locCentral: { id: number; code: string };
  let locWHB: { id: number; code: string };
  let locWHA: { id: number; code: string };
  let userA: { username: string; password: string };
  let userB: { username: string; password: string };

  let driverA: ApiDriver;
  let driverB: ApiDriver;
  let superadmin: ApiDriver;

  test.beforeAll(async ({ request }) => {
    const super0 = await apiAs(request, 'superadmin');

    const storeARes = await super0.post('/api/stores', { name: `E2E SL Store A ${suffix}` });
    expect(storeARes.ok, `create store A failed: ${JSON.stringify(storeARes.body)}`).toBeTruthy();
    storeA = storeARes.body.data;
    tracker.trackStore(storeA.id);

    const storeBRes = await super0.post('/api/stores', { name: `E2E SL Store B ${suffix}` });
    expect(storeBRes.ok, `create store B failed: ${JSON.stringify(storeBRes.body)}`).toBeTruthy();
    storeB = storeBRes.body.data;
    tracker.trackStore(storeB.id);

    // Warehouses have no POST API — create directly via SQL.
    // warehouses.code is varchar(20): prefix + 13-digit timestamp must fit.
    const whACode = `SLWA${suffix}`;
    const whBCode = `SLWH${suffix}`;
    const centralCode = `SLCN${suffix}`;
    execSQL(
      `INSERT INTO warehouses (name, code, store_id) VALUES ('E2E SL WH A ${suffix}', '${whACode}', ${storeA.id})`
    );
    execSQL(
      `INSERT INTO warehouses (name, code, store_id) VALUES ('E2E SL WH B ${suffix}', '${whBCode}', ${storeB.id})`
    );
    execSQL(
      `INSERT INTO warehouses (name, code, store_id) VALUES ('E2E SL Central ${suffix}', '${centralCode}', NULL)`
    );
    whA = Number(
      querySQL<{ id: number }>(`SELECT id FROM warehouses WHERE code = '${whACode}'`)[0].id
    );
    whB = Number(
      querySQL<{ id: number }>(`SELECT id FROM warehouses WHERE code = '${whBCode}'`)[0].id
    );
    centralWh = Number(
      querySQL<{ id: number }>(`SELECT id FROM warehouses WHERE code = '${centralCode}'`)[0].id
    );
    tracker.trackWarehouse(whA);
    tracker.trackWarehouse(whB);
    tracker.trackWarehouse(centralWh);

    // Store-scoped managers (role 2 holds storage_location.create/update/delete/view).
    const usernameA = `e2esla${suffix}`;
    const usernameB = `e2eslb${suffix}`;
    const createUserA = await super0.post('/api/admin/users', {
      username: usernameA,
      email: `${usernameA}@retail-pos.local`,
      password,
      role_id: 2,
      store_id: storeA.id,
    });
    expect(
      createUserA.ok,
      `create user A failed: ${JSON.stringify(createUserA.body)}`
    ).toBeTruthy();
    tracker.trackUser(createUserA.body.data?.id);
    userA = { username: usernameA, password };

    const createUserB = await super0.post('/api/admin/users', {
      username: usernameB,
      email: `${usernameB}@retail-pos.local`,
      password,
      role_id: 2,
      store_id: storeB.id,
    });
    expect(
      createUserB.ok,
      `create user B failed: ${JSON.stringify(createUserB.body)}`
    ).toBeTruthy();
    tracker.trackUser(createUserB.body.data?.id);
    userB = { username: usernameB, password };

    // Seed one location per scope shape, created by superadmin.
    const mkLoc = async (code: string, name: string, scope: Record<string, number>) => {
      const res = await super0.post('/api/storage-locations', { code, name, ...scope });
      expect(res.ok, `create location ${code} failed: ${JSON.stringify(res.body)}`).toBeTruthy();
      tracker.trackLocation(res.body.data.id);
      return res.body.data as { id: number; code: string; name: string };
    };
    locA = await mkLoc(`SLA${suffix}`, 'E2E Loc A', { store_id: storeA.id });
    locB = await mkLoc(`SLB${suffix}`, 'E2E Loc B', { store_id: storeB.id });
    locCentral = await mkLoc(`SLC${suffix}`, 'E2E Loc Central', { warehouse_id: centralWh });
    locWHB = await mkLoc(`SLW${suffix}`, 'E2E Loc WH B', { warehouse_id: whB });
    locWHA = await mkLoc(`SLHA${suffix}`, 'E2E Loc WH A', { warehouse_id: whA });

    // Warm the disk token cache so per-test logins never hit the rate limiter.
    await loginDriver(request, userA.username, userA.password);
    await loginDriver(request, userB.username, userB.password);
  });

  test.beforeEach(async ({ request }) => {
    driverA = await loginDriver(request, userA.username, userA.password);
    driverB = await loginDriver(request, userB.username, userB.password);
    superadmin = await apiAs(request, 'superadmin');
  });

  test.afterAll(() => tracker.cleanup());

  test('list is scoped to the caller store (direct + warehouse-linked rows)', async () => {
    const res = await driverA.get('/api/storage-locations?limit=100');
    expect(res.status).toBe(200);
    const ids = (res.body.data || []).map((l: { id: number }) => l.id);
    expect(ids).toContain(locA.id);
    expect(ids).toContain(locWHA.id);
    expect(ids).not.toContain(locB.id);
    expect(ids).not.toContain(locWHB.id);
    expect(ids).not.toContain(locCentral.id);
  });

  test('superadmin sees every store and the central row', async () => {
    const res = await superadmin.get('/api/storage-locations?limit=100');
    expect(res.status).toBe(200);
    const ids = (res.body.data || []).map((l: { id: number }) => l.id);
    expect(ids).toEqual(
      expect.arrayContaining([locA.id, locB.id, locWHA.id, locWHB.id, locCentral.id])
    );
  });

  test('GET own row → 200, GET foreign row → 403', async () => {
    const own = await driverA.get(`/api/storage-locations/${locA.id}`);
    expect(own.status).toBe(200);
    expect(own.body.data.id).toBe(locA.id);

    const foreign = await driverA.get(`/api/storage-locations/${locB.id}`);
    expect(foreign.status).toBe(403);
    expect(String(foreign.body.error)).toMatch(/not in your store/i);

    // Symmetric: B cannot read A's row either.
    const reverse = await driverB.get(`/api/storage-locations/${locA.id}`);
    expect(reverse.status).toBe(403);
  });

  test('central warehouse row is superadmin-only', async () => {
    const forbidden = await driverA.get(`/api/storage-locations/${locCentral.id}`);
    expect(forbidden.status).toBe(403);

    const allowed = await superadmin.get(`/api/storage-locations/${locCentral.id}`);
    expect(allowed.status).toBe(200);
    expect(allowed.body.data.id).toBe(locCentral.id);
  });

  test('create into foreign warehouse/store → 403; own store → 201', async () => {
    const foreignWH = await driverA.post('/api/storage-locations', {
      code: `XWH${suffix}`,
      name: 'Foreign WH Loc',
      warehouse_id: whB,
    });
    expect(foreignWH.status).toBe(403);

    const foreignStore = await driverA.post('/api/storage-locations', {
      code: `XST${suffix}`,
      name: 'Foreign Store Loc',
      store_id: storeB.id,
    });
    expect(foreignStore.status).toBe(403);

    const central = await driverA.post('/api/storage-locations', {
      code: `XCN${suffix}`,
      name: 'Central Loc',
      warehouse_id: centralWh,
    });
    expect(central.status).toBe(403);

    const own = await driverA.post('/api/storage-locations', {
      code: `OWN${suffix}`,
      name: 'Own Store Loc',
      store_id: storeA.id,
    });
    expect(own.status, JSON.stringify(own.body)).toBe(201);
    tracker.trackLocation(own.body.data.id);
    expect(own.body.data.store_id).toBe(storeA.id);

    const ownWH = await driverA.post('/api/storage-locations', {
      code: `OWH${suffix}`,
      name: 'Own WH Loc',
      warehouse_id: whA,
    });
    expect(ownWH.status, JSON.stringify(ownWH.body)).toBe(201);
    tracker.trackLocation(ownWH.body.data.id);
    expect(ownWH.body.data.warehouse_id).toBe(whA);
  });

  test('update own row → 200', async () => {
    const upd = await driverA.put(`/api/storage-locations/${locA.id}`, {
      name: 'E2E Loc A Renamed',
    });
    expect(upd.status, JSON.stringify(upd.body)).toBe(200);
    expect(upd.body.data.name).toBe('E2E Loc A Renamed');

    const restore = await driverA.put(`/api/storage-locations/${locA.id}`, {
      name: locA.name,
    });
    expect(restore.status).toBe(200);
  });

  test('update/delete foreign row → 403 and the row is untouched', async () => {
    const upd = await driverA.put(`/api/storage-locations/${locB.id}`, { name: 'Hijacked' });
    expect(upd.status).toBe(403);

    const del = await driverA.del(`/api/storage-locations/${locB.id}`);
    expect(del.status).toBe(403);

    const check = await superadmin.get(`/api/storage-locations/${locB.id}`);
    expect(check.status).toBe(200);
    expect(check.body.data.name).toBe(locB.name);
  });

  test('bulk update with one foreign id → 403, no partial write', async () => {
    const own = await driverA.post('/api/storage-locations', {
      code: `BUK${suffix}`,
      name: 'Bulk Own Loc',
      store_id: storeA.id,
    });
    expect(own.status, JSON.stringify(own.body)).toBe(201);
    tracker.trackLocation(own.body.data.id);

    const res = await driverA.put('/api/storage-locations/bulk', {
      ids: [own.body.data.id, locB.id],
      is_active: false,
    });
    expect(res.status).toBe(403);

    const check = await superadmin.get(`/api/storage-locations/${own.body.data.id}`);
    expect(check.status).toBe(200);
    expect(check.body.data.is_active).toBe(true);

    const foreign = await superadmin.get(`/api/storage-locations/${locB.id}`);
    expect(foreign.body.data.is_active).toBe(true);
  });

  test('bulk delete with foreign id → 403, both rows survive', async () => {
    const res = await driverA.del('/api/storage-locations/bulk', {
      ids: [locA.id, locB.id],
    });
    expect(res.status).toBe(403);

    expect((await superadmin.get(`/api/storage-locations/${locA.id}`)).status).toBe(200);
    expect((await superadmin.get(`/api/storage-locations/${locB.id}`)).status).toBe(200);
  });

  test('own-store bulk update succeeds', async () => {
    const upd = await driverA.put('/api/storage-locations/bulk', {
      ids: [locA.id],
      is_active: false,
    });
    expect(upd.status, JSON.stringify(upd.body)).toBe(200);
    expect(upd.body.updated).toBe(1);

    const check = await superadmin.get(`/api/storage-locations/${locA.id}`);
    expect(check.body.data.is_active).toBe(false);

    const restore = await driverA.put('/api/storage-locations/bulk', {
      ids: [locA.id],
      is_active: true,
    });
    expect(restore.status).toBe(200);
  });
});
