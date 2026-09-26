/**
 * Forced first-login password rotation (users.must_change_password, migration
 * 050).
 *
 * Behaviour lives at the API layer: the login response carries the flag, every
 * non-allowlisted route answers 428 with the machine code, and ChangePassword
 * clears the gate. The last test covers the one genuine UI behaviour — the SPA
 * must block behind the rotation modal instead of half-rendering.
 */
import { test, expect } from './fixtures';
import { API_BASE, FRONTEND_BASE, authHeader } from './fixtures';
import { apiAs, ApiDriver } from './api-driver';
import { TestDataTracker } from './db-helper';

const suffix = Date.now().toString(36);
const INITIAL_PASSWORD = 'start1234';
const ROTATED_PASSWORD = 'rotated1234';

/** Creates a cashier account that owes a rotation, seeded into the default store. */
async function createGatedUser(super0: ApiDriver, username: string): Promise<number> {
  const rolesRes = await super0.get('/api/admin/roles');
  const roles = rolesRes.body.data ?? rolesRes.body;
  const cashier = roles.find((r: { name: string }) => r.name === 'cashier');
  expect(cashier, 'cashier role must be seeded').toBeTruthy();

  const storesRes = await super0.get('/api/stores?limit=1&offset=0');
  expect(storesRes.status, JSON.stringify(storesRes.body)).toBe(200);
  const storeId = storesRes.body.data[0].id;

  const res = await super0.post('/api/admin/users', {
    username,
    email: `${username}@retail-pos.local`,
    password: INITIAL_PASSWORD,
    role_id: cashier.id,
    store_id: storeId,
    is_active: true,
    must_change_password: true,
  });
  expect(res.ok, JSON.stringify(res.body)).toBeTruthy();
  return res.body.data?.id as number;
}

test.describe('Forced password rotation', () => {
  const tracker = new TestDataTracker();
  let super0: ApiDriver;

  test.beforeAll(async ({ request }) => {
    super0 = await apiAs(request, 'superadmin');
  });

  test.beforeEach(async ({ request }) => {
    super0 = new ApiDriver(request, super0.token);
  });

  test.afterAll(() => tracker.cleanup());

  test('a gated session gets 428 until the password is rotated', async ({
    request,
  }) => {
    const username = `gate${suffix}`;
    tracker.trackUser(await createGatedUser(super0, username));

    const login = await request.post(`${API_BASE}/api/login`, {
      data: { username, password: INITIAL_PASSWORD },
    });
    expect(login.ok(), await login.text()).toBeTruthy();
    const loginBody = await login.json();
    expect(loginBody.user?.must_change_password).toBe(true);
    let token = loginBody.access_token as string;

    // Non-allowlisted route → 428 with the machine code the client keys on.
    const gated = await request.get(`${API_BASE}/api/pos/cart`, {
      headers: authHeader(token),
    });
    expect(gated.status()).toBe(428);
    expect((await gated.json()).error?.code).toBe('PASSWORD_CHANGE_REQUIRED');

    // Allowlisted route still works and clears the flag on success.
    const rotate = await request.post(`${API_BASE}/api/change-password`, {
      headers: authHeader(token),
      data: { current_password: INITIAL_PASSWORD, new_password: ROTATED_PASSWORD },
    });
    expect(rotate.ok(), await rotate.text()).toBeTruthy();
    const rotateBody = await rotate.json();
    expect(rotateBody.access_token).toBeTruthy();
    token = rotateBody.access_token;

    const after = await request.get(`${API_BASE}/api/pos/cart`, {
      headers: authHeader(token),
    });
    expect(after.status()).not.toBe(428);

    // The rotation flag does not survive a re-login.
    const relogin = await request.post(`${API_BASE}/api/login`, {
      data: { username, password: ROTATED_PASSWORD },
    });
    expect(relogin.ok(), await relogin.text()).toBeTruthy();
    expect((await relogin.json()).user?.must_change_password).toBeFalsy();
  });

  test('the wrong current password keeps the gate in place', async ({
    request,
  }) => {
    const username = `badgate${suffix}`;
    tracker.trackUser(await createGatedUser(super0, username));

    const login = await request.post(`${API_BASE}/api/login`, {
      data: { username, password: INITIAL_PASSWORD },
    });
    const token = (await login.json()).access_token as string;

    const rotate = await request.post(`${API_BASE}/api/change-password`, {
      headers: authHeader(token),
      data: { current_password: 'not-my-password', new_password: ROTATED_PASSWORD },
    });
    expect(rotate.status()).toBe(401);

    const gated = await request.get(`${API_BASE}/api/pos/cart`, {
      headers: authHeader(token),
    });
    expect(gated.status()).toBe(428);
  });

  test('the SPA blocks a gated session behind the rotation modal', async ({
    page,
    request,
  }) => {
    const username = `uigate${suffix}`;
    tracker.trackUser(await createGatedUser(super0, username));

    await page.goto(`${FRONTEND_BASE}/login`);
    await page.fill('#username', username);
    await page.fill('#password', INITIAL_PASSWORD);
    await page.click('form button[type="submit"]');

    // Unlike loginUI (token injection), this test uses the real form login,
    // which applies the account language — users.language defaults to 'id',
    // so the modal may render in either locale.
    const dialog = page.getByRole('dialog', {
      name: /^(Change Your Password|Ganti Kata Sandi)$/,
    });
    await expect(dialog).toBeVisible({ timeout: 20000 });
    // Nothing of the app renders while the gate is up.
    await expect(page.locator('aside')).toHaveCount(0);

    await dialog.locator('#force-current-password').fill(INITIAL_PASSWORD);
    await dialog.locator('#force-new-password').fill(ROTATED_PASSWORD);
    await dialog.locator('#force-confirm-password').fill(ROTATED_PASSWORD);
    await dialog.getByRole('button', { name: /^(Save|Simpan)$/ }).click();

    await expect(dialog).toBeHidden({ timeout: 20000 });
    await expect(page.locator('aside')).toBeVisible({ timeout: 20000 });
  });
});
