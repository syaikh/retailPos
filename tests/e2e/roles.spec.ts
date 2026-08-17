import { test, expect } from './fixtures';
import type { Page } from '@playwright/test';
import { TEST_USERS, API_BASE, authHeader, getToken as cachedGetToken, loginUI, logoutUI } from './fixtures';

const getToken = cachedGetToken;

// ── Helpers ────────────────────────────────────────────────────────

async function navigateToRoles(page: Page) {
  await page.goto('http://localhost:5173/admin/roles');
  await expect(page).toHaveURL(/\/admin\/roles/);
  await expect(page.getByRole('heading', { name: 'Roles' })).toBeVisible({ timeout: 10000 });
}

async function openCreateRoleModal(page: Page) {
  await page.getByRole('button', { name: 'Create Role' }).first().click();
  await expect(page.getByRole('dialog').first()).toBeVisible({ timeout: 10000 });
}

function getModal(page: Page) {
  return page.getByRole('dialog').first();
}

function getSaveButton(page: Page) {
  return getModal(page).getByRole('button', { name: 'Create Role' });
}

function getNextButton(page: Page) {
  return getModal(page).getByRole('button', { name: 'Next →' });
}

function getCancelButton(page: Page) {
  return getModal(page).getByRole('button', { name: 'Cancel' });
}

async function fillRoleName(page: Page, name: string) {
  const input = page.locator('#role-name');
  await input.fill(name);
  await input.blur();
}

async function fillRoleDescription(page: Page, desc: string) {
  await page.fill('#role-desc', desc);
}

async function goToStep2(page: Page) {
  await getNextButton(page).click();
  await expect(getModal(page).getByText('PERMISSIONS', { exact: true })).toBeVisible({ timeout: 5000 });
}

async function expandGroup(page: Page, groupLabel: string) {
  const modal = getModal(page);
  const toggle = modal.getByRole('button', { name: `Toggle ${groupLabel} permissions` });
  const isExpanded = await toggle.getAttribute('aria-expanded');
  if (isExpanded === 'false') {
    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-expanded', 'true');
  }
}

async function collapseGroup(page: Page, groupLabel: string) {
  const modal = getModal(page);
  const toggle = modal.getByRole('button', { name: `Toggle ${groupLabel} permissions` });
  const isExpanded = await toggle.getAttribute('aria-expanded');
  if (isExpanded === 'true') {
    await toggle.click();
    await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  }
}

async function selectAllInGroup(page: Page, groupLabel: string) {
  await expandGroup(page, groupLabel);
  const modal = getModal(page);
  const btn = modal.getByRole('button', { name: `Select all ${groupLabel} permissions` });
  if (await btn.isVisible()) {
    await btn.click();
  }
}

async function deselectAllInGroup(page: Page, groupLabel: string) {
  await expandGroup(page, groupLabel);
  const modal = getModal(page);
  const btn = modal.getByRole('button', { name: `Deselect all ${groupLabel} permissions` });
  if (await btn.isVisible()) {
    await btn.click();
  }
}

async function searchPermissions(page: Page, query: string) {
  const searchInput = page.getByPlaceholder('Search permissions...');
  await searchInput.fill(query);
}

async function saveRole(page: Page) {
  await getSaveButton(page).click();
}

async function cancelModal(page: Page) {
  await getCancelButton(page).click();
}

// ── Test Suite ─────────────────────────────────────────────────────

test.describe('Admin Panel — Role Management (Create Role Modal)', () => {
  test.beforeEach(async ({ page }) => {
    await loginUI(page, 'superadmin', 'admin123');
    await navigateToRoles(page);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  // ── 1. Modal Open / Close ──────────────────────────────────────

  test('should open Create Role modal when clicking Create Role button', async ({ page }) => {
    await openCreateRoleModal(page);
    await expect(page.getByRole('dialog').first()).toBeVisible();
    await expect(page.getByLabel('Role Name')).toBeVisible();
    await expect(page.getByLabel('Description')).toBeVisible();
    await expect(getCancelButton(page)).toBeVisible();
    await expect(getNextButton(page)).toBeVisible();
  });

  test('should close modal when clicking Cancel with no changes', async ({ page }) => {
    await openCreateRoleModal(page);
    await cancelModal(page);
    await expect(page.getByRole('dialog').first()).toBeHidden({ timeout: 5000 });
  });

  test('should close modal when pressing Escape with no changes', async ({ page }) => {
    await openCreateRoleModal(page);
    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog').first()).toBeHidden({ timeout: 5000 });
  });

  // ── 2. Form Validation ─────────────────────────────────────────

  test('should show inline error when role name is empty and blurred', async ({ page }) => {
    await openCreateRoleModal(page);
    const nameInput = page.locator('#role-name');
    await nameInput.fill('x');
    await nameInput.fill('');
    await nameInput.blur();
    await expect(page.getByText('Role name is required')).toBeVisible({ timeout: 5000 });
    await expect(nameInput).toHaveAttribute('aria-invalid', 'true');
  });

  test('should show error for duplicate role name', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, 'superadmin');
    await expect(page.getByText('Role name already exists')).toBeVisible({ timeout: 5000 });
  });

  test('should disable Next button when name is empty', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, '');
    await expect(getNextButton(page)).toBeDisabled();
  });

  test('should enable Next button when name is valid', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, `TestRole${Date.now()}`);
    await expect(getNextButton(page)).toBeEnabled();
  });

  // ── 3. Basic Role Creation ─────────────────────────────────────

  test('should create a role with name and description only (no permissions)', async ({ page }) => {
    const uniqueName = `testrole_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await fillRoleDescription(page, 'E2E test role with no permissions');
    await goToStep2(page);
    await saveRole(page);
    await expect(page.getByRole('dialog').first()).toBeHidden({ timeout: 10000 });
    const searchBar = page.getByPlaceholder('Search roles...');
    await searchBar.fill(uniqueName);
    await expect(page.getByText(uniqueName, { exact: true })).toBeVisible({ timeout: 10000 });
  });

  test('should create a role with specific permissions', async ({ page }) => {
    const uniqueName = `testrole_perm_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await fillRoleDescription(page, 'E2E test role with inventory permissions');
    await goToStep2(page);
    await selectAllInGroup(page, 'Inventory');
    await expect(getModal(page).getByText('1 of', { exact: false })).toBeVisible({ timeout: 5000 });
    await saveRole(page);
    await expect(page.getByRole('dialog').first()).toBeHidden({ timeout: 10000 });
    const searchBar178 = page.getByPlaceholder('Search roles...');
    await searchBar178.fill(uniqueName);
    await expect(page.getByText(uniqueName, { exact: true })).toBeVisible({ timeout: 10000 });
  });

  // ── 4. Permission Groups ───────────────────────────────────────

  test('should display all permission groups collapsed by default', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, `testrole_${Date.now()}`);
    await goToStep2(page);

    const userRoleToggle = getModal(page).getByRole('button', { name: 'Toggle User & Role permissions' });
    await expect(userRoleToggle).toBeVisible();
    await expect(userRoleToggle).toHaveAttribute('aria-expanded', 'false');
    const productToggle = getModal(page).getByRole('button', { name: 'Toggle Product permissions' });
    await expect(productToggle).toHaveAttribute('aria-expanded', 'false');
  });

  test('should expand and collapse permission groups', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, `testrole_${Date.now()}`);
    await goToStep2(page);

    await expandGroup(page, 'Inventory');
    const toggle = getModal(page).getByRole('button', { name: 'Toggle Inventory permissions' });
    await expect(toggle).toHaveAttribute('aria-expanded', 'true');
    await expect(page.locator('#group-body-inventory')).toBeVisible();
    await collapseGroup(page, 'Inventory');
    await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  });

  test('should show Expand All / Collapse All toggle', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, `testrole_${Date.now()}`);
    await goToStep2(page);

    const modal = getModal(page);
    const expandAllBtn = modal.getByText('Expand All');
    await expect(expandAllBtn).toBeVisible();
    await expandAllBtn.click();
    await expect(modal.getByText('Collapse All')).toBeVisible();
    const userRoleToggle = modal.getByRole('button', { name: 'Toggle User & Role permissions' });
    await expect(userRoleToggle).toHaveAttribute('aria-expanded', 'true');
    await modal.getByText('Collapse All').click();
    await expect(modal.getByText('Expand All')).toBeVisible();
    await expect(userRoleToggle).toHaveAttribute('aria-expanded', 'false');
  });

  // ── 5. Select All / Deselect All per Group ─────────────────────

  test('should select all permissions in a group', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, `testrole_${Date.now()}`);
    await goToStep2(page);
    await selectAllInGroup(page, 'Inventory');

    const groupBody = page.locator('#group-body-inventory');
    const checkboxes = groupBody.locator('input[type="checkbox"]');
    await expect(checkboxes).toHaveCount(1);
    for (let i = 0; i < 1; i++) {
      await expect(checkboxes.nth(i)).toBeChecked();
    }
    await expect(getModal(page).getByRole('button', { name: 'Deselect all Inventory permissions' })).toBeVisible();
  });

  test('should deselect all permissions in a group', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, `testrole_${Date.now()}`);
    await goToStep2(page);
    await selectAllInGroup(page, 'Inventory');
    await deselectAllInGroup(page, 'Inventory');

    const groupBody = page.locator('#group-body-inventory');
    const checkboxes = groupBody.locator('input[type="checkbox"]');
    for (let i = 0; i < 1; i++) {
      await expect(checkboxes.nth(i)).not.toBeChecked();
    }
    await expect(getModal(page).getByRole('button', { name: 'Select all Inventory permissions' })).toBeVisible();
  });

  // ── 6. Search ──────────────────────────────────────────────────

  test('should filter permissions by search query', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, `testrole_${Date.now()}`);
    await goToStep2(page);
    await searchPermissions(page, 'inventory');
    await expect(getModal(page).getByRole('button', { name: 'Toggle Inventory permissions' })).toBeVisible();
    await expandGroup(page, 'Inventory');
    await expect(page.locator('#group-body-inventory')).toBeVisible();
  });

  test('should show empty state when no permissions match search', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, `testrole_${Date.now()}`);
    await goToStep2(page);
    await searchPermissions(page, 'zzzznonexistent');
    await expect(page.getByText(/No results:/)).toBeVisible({ timeout: 5000 });
  });

  test('should clear search with X button', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, `testrole_${Date.now()}`);
    await goToStep2(page);
    const searchInput = page.getByPlaceholder('Search permissions...');
    await searchInput.fill('inventory');
    await expect(searchInput).toHaveValue('inventory');

    const clearBtn = page.getByRole('button', { name: 'Clear search' });
    await expect(clearBtn).toBeVisible();
    await clearBtn.click();
    await page.waitForTimeout(300);
    await expect(searchInput).toHaveValue('', { timeout: 5000 });
  });

  test('should clear search with Escape key', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, `testrole_${Date.now()}`);
    await goToStep2(page);
    const searchInput = page.getByPlaceholder('Search permissions...');
    await searchInput.fill('inventory');
    await expect(searchInput).toHaveValue('inventory');

    await searchInput.press('Escape');
    await page.waitForTimeout(300);

    // Escape clears the search state — clear button should disappear
    await expect(page.getByRole('button', { name: 'Clear search' })).toBeHidden({ timeout: 3000 });
  });

  // ── 7. Unsaved Changes Guard ───────────────────────────────────

  test('should show discard confirmation when canceling with unsaved changes', async ({ page }) => {
    const uniqueName = `testrole_unsaved_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await goToStep2(page);
    await selectAllInGroup(page, 'Inventory');

    await cancelModal(page);

    const discardDialog = page.getByRole('dialog', { name: 'Discard Changes?' });
    await expect(discardDialog).toBeVisible({ timeout: 5000 });
    await expect(discardDialog.getByText('You have unsaved changes')).toBeVisible();
  });

  test('should discard changes when confirming discard', async ({ page }) => {
    const uniqueName = `testrole_discard_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await goToStep2(page);
    await selectAllInGroup(page, 'Inventory');

    await cancelModal(page);
    await expect(page.getByRole('dialog', { name: 'Discard Changes?' })).toBeVisible();

    await page.getByRole('button', { name: 'Discard' }).click();

    await expect(page.getByRole('dialog', { name: 'Discard Changes?' })).toBeHidden({ timeout: 5000 });
    await expect(page.getByRole('dialog').first()).toBeHidden({ timeout: 5000 });
    await expect(page.getByText(uniqueName, { exact: true })).toBeHidden();
  });

  test('should keep editing when canceling discard', async ({ page }) => {
    const uniqueName = `testrole_keep_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await goToStep2(page);
    await selectAllInGroup(page, 'Inventory');

    await cancelModal(page);
    await expect(page.getByRole('dialog', { name: 'Discard Changes?' })).toBeVisible();

    await page.getByRole('button', { name: 'Keep Editing' }).click();

    await expect(page.getByRole('dialog', { name: 'Discard Changes?' })).toBeHidden({ timeout: 5000 });
    await expect(page.getByRole('dialog').first()).toBeVisible();

    // Modal stays on Step 2 — permissions section should still be visible
    await expect(getModal(page).getByText('PERMISSIONS', { exact: true })).toBeVisible();
  });

  // ── 8. Selected Count ──────────────────────────────────────────

  test('should update selected count as permissions are toggled', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, `testrole_${Date.now()}`);
    await goToStep2(page);
    const modal = getModal(page);

    await expect(modal.getByText('0 of', { exact: false })).toBeVisible();
    await selectAllInGroup(page, 'Inventory');
    await expect(modal.getByText('1 of', { exact: false })).toBeVisible({ timeout: 5000 });
    await selectAllInGroup(page, 'Sales');
    await expect(modal.getByText('4 of', { exact: false })).toBeVisible({ timeout: 5000 });
    await deselectAllInGroup(page, 'Inventory');
    await expect(modal.getByText('3 of', { exact: false })).toBeVisible({ timeout: 5000 });
  });

  // ── 9. Group Count Badge ───────────────────────────────────────

  test('should show correct selected/total count per group', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, `testrole_${Date.now()}`);
    await goToStep2(page);
    await expandGroup(page, 'Inventory');

    const inventoryGroup = page.locator('[data-group]').filter({ hasText: 'Inventory' });
    await expect(inventoryGroup.getByText('0/1')).toBeVisible();

    const groupBody = page.locator('#group-body-inventory');
    const checkboxes = groupBody.locator('input[type="checkbox"]');
    await checkboxes.first().check();
    await expect(inventoryGroup.getByText('1/1')).toBeVisible({ timeout: 5000 });
  });

  // ── 10. Full Flow: Create Role with Multiple Groups ────────────

  test('should create a role selecting permissions from multiple groups', async ({ page }) => {
    const uniqueName = `testrole_full_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await fillRoleDescription(page, 'Full flow test role');
    await goToStep2(page);

    await selectAllInGroup(page, 'Inventory');
    await selectAllInGroup(page, 'Sales');
    await selectAllInGroup(page, 'Dashboard');
    await expect(getModal(page).getByText('5 of', { exact: false })).toBeVisible({ timeout: 5000 });

    await saveRole(page);
    await expect(page.getByRole('dialog').first()).toBeHidden({ timeout: 10000 });
    const searchBar410 = page.getByPlaceholder('Search roles...');
    await searchBar410.fill(uniqueName);
    await expect(page.getByText(uniqueName, { exact: true })).toBeVisible({ timeout: 10000 });

    const roleRow = page.locator('tr').filter({ hasText: uniqueName });
    await expect(roleRow.getByText('5', { exact: true })).toBeVisible({ timeout: 5000 });
  });

  // ── 11. Toast Feedback ─────────────────────────────────────────

  test('should show success toast after creating a role', async ({ page }) => {
    const uniqueName = `testrole_toast_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await goToStep2(page);
    await saveRole(page);
    await expect(page.getByText('Role created')).toBeVisible({ timeout: 10000 });
  });

  // ── 12. Stale State on Reopen ──────────────────────────────────

  test('should reset form state when reopening modal', async ({ page }) => {
    const name1 = `testrole_first_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, name1);
    await goToStep2(page);
    await selectAllInGroup(page, 'Inventory');
    await saveRole(page);
    await expect(page.getByRole('dialog').first()).toBeHidden({ timeout: 10000 });

    await openCreateRoleModal(page);

    await expect(page.locator('#role-name')).toHaveValue('');
    await expect(page.locator('#role-desc')).toHaveValue('');

    await fillRoleName(page, `temp_reset_${Date.now()}`);
    await goToStep2(page);

    await expect(getModal(page).getByText('0 of', { exact: false })).toBeVisible();
    const toggle = getModal(page).getByRole('button', { name: 'Toggle User & Role permissions' });
    await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  });
});

// ============================================================================
// Role API: Update, Delete, Permissions
// ============================================================================

test.describe('Roles API - Update Role', () => {
  test('PUT /admin/roles/:id updates role name', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    // Create a role first
    const createRes = await request.post(`${API_BASE}/api/admin/roles`, {
      headers: authHeader(token),
      data: { name: `updatable_${Date.now()}`, description: 'to be updated' },
    });
    expect(createRes.ok(), `create failed: ${createRes.status()}`).toBeTruthy();
    const created = await createRes.json();
    const roleId = created.data.id;

    const newName = `updated_${Date.now()}`;
    const res = await request.put(`${API_BASE}/api/admin/roles/${roleId}`, {
      headers: authHeader(token),
      data: { name: newName },
    });
    expect(res.ok(), `update failed: ${res.status()}: ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    expect(body.data.name).toBe(newName);
  });

  test('PUT /admin/roles/:id updates description', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const createRes = await request.post(`${API_BASE}/api/admin/roles`, {
      headers: authHeader(token),
      data: { name: `descupdate_${Date.now()}`, description: 'old desc' },
    });
    const created = await createRes.json();
    const roleId = created.data.id;

    const res = await request.put(`${API_BASE}/api/admin/roles/${roleId}`, {
      headers: authHeader(token),
      data: { description: 'new description' },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.data.description).toBe('new description');
  });

  test('PUT /admin/roles/:id returns 404 for nonexistent role', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);
    const res = await request.put(`${API_BASE}/api/admin/roles/999999`, {
      headers: authHeader(token),
      data: { name: 'nope' },
    });
    expect(res.status()).toBe(404);
  });

  test('PUT /admin/roles/:id with restricted role returns 403', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.put(`${API_BASE}/api/admin/roles/1`, {
      headers: authHeader(token),
      data: { name: 'hack' },
    });
    expect(res.status()).toBe(403);
  });
});

test.describe('Roles API - Update Permissions', () => {
  test('PUT /admin/roles/:id/permissions updates permission set', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    // Get valid permission IDs from the permissions endpoint
    const permRes = await request.get(`${API_BASE}/api/admin/permissions`, { headers: authHeader(token) });
    expect(permRes.ok()).toBeTruthy();
    const permBody = await permRes.json();
    const perms = permBody.data || [];
    if (perms.length === 0) return;
    const permIds = perms.slice(0, 3).map((p: any) => p.id);

    const createRes = await request.post(`${API_BASE}/api/admin/roles`, {
      headers: authHeader(token),
      data: { name: `permrole_${Date.now()}`, description: 'for perm test' },
    });
    const created = await createRes.json();
    const roleId = created.data.id;

    const res = await request.put(`${API_BASE}/api/admin/roles/${roleId}/permissions`, {
      headers: authHeader(token),
      data: { permission_ids: permIds },
    });
    expect(res.ok(), `perm update failed: ${res.status()}: ${await res.text()}`).toBeTruthy();
    const body = await res.json();
    expect(body.data.permissions).toBeDefined();
  });

  test('PUT /admin/roles/:id/permissions with empty array is accepted', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const createRes = await request.post(`${API_BASE}/api/admin/roles`, {
      headers: authHeader(token),
      data: { name: `emptyperm_${Date.now()}`, description: 'for empty perm test' },
    });
    const created = await createRes.json();
    const roleId = created.data.id;

    const res = await request.put(`${API_BASE}/api/admin/roles/${roleId}/permissions`, {
      headers: authHeader(token),
      data: { permission_ids: [] },
    });
    expect(res.ok()).toBeTruthy();
  });
});

test.describe('Roles API - Delete Role', () => {
  test('DELETE /admin/roles/:id deletes a role with no users', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    const createRes = await request.post(`${API_BASE}/api/admin/roles`, {
      headers: authHeader(token),
      data: { name: `deleteme_${Date.now()}`, description: 'to be deleted' },
    });
    expect(createRes.ok()).toBeTruthy();
    const created = await createRes.json();
    const roleId = created.data.id;

    const deleteRes = await request.delete(`${API_BASE}/api/admin/roles/${roleId}`, {
      headers: authHeader(token),
    });
    expect(deleteRes.ok(), `delete failed: ${deleteRes.status()}: ${await deleteRes.text()}`).toBeTruthy();
    const body = await deleteRes.json();
    expect(body.status).toBe('deleted');

    // Verify it's gone
    const getRes = await request.get(`${API_BASE}/api/admin/roles?limit=100`, { headers: authHeader(token) });
    const roles = await getRes.json();
    const found = roles.data?.find((r: any) => r.id === roleId);
    expect(found).toBeUndefined();
  });

  test('DELETE /admin/roles/:id returns 400 for role with assigned users', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.superadmin.username, TEST_USERS.superadmin.password);

    // Get the admin role (id=2) which has users assigned
    const res = await request.delete(`${API_BASE}/api/admin/roles/2`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(400);
    const body = await res.json();
    expect(body.error).toContain('users are assigned');
  });

  test('DELETE /admin/roles/:id with restricted role returns 403', async ({ request }) => {
    const token = await getToken(request, TEST_USERS.cashier.username, TEST_USERS.cashier.password);
    const res = await request.delete(`${API_BASE}/api/admin/roles/1`, {
      headers: authHeader(token),
    });
    expect(res.status()).toBe(403);
  });
});
