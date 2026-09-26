/**
 * Store onboarding — behaviour layer (API).
 *
 * The wizard's step gating is genuine UI behaviour and lives in
 * stores.spec.ts; everything derived from it (computed readiness blockers)
 * is asserted here through the API, where it is cheaper and deterministic.
 * HQ-only provisioning (store.create revoked from manager in migration 050)
 * is RBAC enforcement and is asserted in rbac-api.spec.ts.
 *
 * The three readiness tests share one store created in beforeAll and mutate it
 * in order (storage location → staff), so each assertion shows the blocker
 * disappearing rather than being absent by construction.
 *
 * Retry safety (CI runs `retries: 2`): a retried test does not re-run its
 * predecessors, so every mutating test mints a fresh unique code/username per
 * attempt and asserts counts as "at least one" — a partial first attempt can
 * therefore never fail the retry on a uniqueness violation.
 */
import { test, expect } from './fixtures';
import { apiAs, ApiDriver } from './api-driver';
import { TestDataTracker } from './db-helper';

const suffix = Date.now().toString(36).toUpperCase();

test.describe('Store onboarding (API)', () => {
  const tracker = new TestDataTracker();
  let super0: ApiDriver;
  let storeId: number;

  test.beforeAll(async ({ request }) => {
    super0 = await apiAs(request, 'superadmin');

    const create = await super0.post('/api/stores', {
      name: `E2E Onboard ${suffix}`,
      address: 'Jl. Onboarding No. 1',
      phone: '022-555-001',
    });
    expect(create.status, JSON.stringify(create.body)).toBe(201);
    storeId = create.body.data.id;
    tracker.trackStore(storeId);
  });

  // Token rules: recreate the driver per test; never reuse a beforeAll request.
  test.beforeEach(async ({ request }) => {
    super0 = new ApiDriver(request, super0.token);
  });

  test.afterAll(() => tracker.cleanup());

  test('a fresh store is blocked on staff and storage location, not on profile', async () => {
    const res = await super0.get(`/api/stores/${storeId}/readiness`);
    expect(res.status, JSON.stringify(res.body)).toBe(200);
    const r = res.body.data;

    expect(r.store_id).toBe(storeId);
    expect(r.ready).toBe(false);
    expect(r.address_set).toBe(true);
    expect(r.phone_set).toBe(true);
    expect(r.storage_locations).toBe(0);
    expect(r.staff).toEqual({});
    // Catalog scope: SellableStats are store-agnostic, so the seeded catalogue
    // is visible from a brand new store (query path via product.SellableStats).
    expect(r.catalog.active_products).toBeGreaterThan(0);
    expect(r.catalog.zero_stock_products).toBeLessThanOrEqual(r.catalog.active_products);
    expect(r.required_roles).toEqual(
      expect.arrayContaining([
        'manager',
        'supervisor',
        'cashier',
        'inventory_staff',
        'finance',
      ])
    );
    expect(r.blockers).toEqual(
      expect.arrayContaining([
        'staff.manager',
        'staff.supervisor',
        'staff.cashier',
        'staff.inventory_staff',
        'staff.finance',
        'storage_location',
      ])
    );
    expect(r.blockers).not.toContain('store.address');
    expect(r.blockers).not.toContain('store.phone');
    expect(r.blockers).not.toContain('store.inactive');
  });

  test('creating a storage location clears only the storage_location blocker', async () => {
    const attempt = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`;
    const loc = await super0.post('/api/storage-locations', {
      code: `ONB${attempt}`.slice(0, 24).toUpperCase(),
      name: 'E2E Onboard Rack',
      store_id: storeId,
    });
    expect(loc.ok, JSON.stringify(loc.body)).toBeTruthy();
    tracker.trackLocation(loc.body.data.id);

    const r = (await super0.get(`/api/stores/${storeId}/readiness`)).body.data;
    expect(r.storage_locations).toBeGreaterThanOrEqual(1);
    expect(r.blockers).not.toContain('storage_location');
    expect(r.blockers).toContain('staff.manager');
    expect(r.ready).toBe(false);
  });

  test('staffing one role clears only that role blocker', async () => {
    const rolesRes = await super0.get('/api/admin/roles');
    const roles = rolesRes.body.data ?? rolesRes.body;
    const cashierRole = roles.find((r: { name: string }) => r.name === 'cashier');
    expect(cashierRole, 'cashier role must be seeded').toBeTruthy();

    const username = `onbcash${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`
      .toLowerCase()
      .slice(0, 24);
    const created = await super0.post('/api/admin/users', {
      username,
      email: `${username}@retail-pos.local`,
      password: 'start1234',
      role_id: cashierRole.id,
      store_id: storeId,
      is_active: true,
      must_change_password: true,
    });
    expect(created.ok, JSON.stringify(created.body)).toBeTruthy();
    tracker.trackUser(created.body.data?.id);

    const r = (await super0.get(`/api/stores/${storeId}/readiness`)).body.data;
    expect(r.staff.cashier).toBeGreaterThanOrEqual(1);
    expect(r.blockers).not.toContain('staff.cashier');
    expect(r.blockers).toContain('staff.manager');
    expect(r.ready).toBe(false);
  });

  test('invalid readiness id is rejected before any lookup', async () => {
    const res = await super0.get('/api/stores/not-a-number/readiness');
    expect(res.status).toBe(400);
  });
});
