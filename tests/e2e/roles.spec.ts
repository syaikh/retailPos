import { test, expect, type Page } from '@playwright/test';

// ── Helpers ────────────────────────────────────────────────────────

async function loginAsSuperadmin(page: Page) {
  await page.goto('http://localhost:5173/login');
  await page.fill('#username', 'superadmin');
  await page.fill('#password', 'admin123');
  await page.click('button[type="submit"]');
  await expect(page).toHaveURL(/\/$/, { timeout: 10000 });
}

async function navigateToRoles(page: Page) {
  await page.goto('http://localhost:5173/admin/roles');
  await expect(page).toHaveURL(/\/admin\/roles/);
  await expect(page.getByRole('heading', { name: 'Roles Management' })).toBeVisible({ timeout: 10000 });
}

async function openCreateRoleModal(page: Page) {
  await page.getByRole('button', { name: 'Create Role' }).first().click();
  await expect(page.getByRole('dialog', { name: 'Create New Role' })).toBeVisible({ timeout: 10000 });
}

function getModal(page: Page) {
  return page.getByRole('dialog', { name: 'Create New Role' });
}

function getSaveButton(page: Page) {
  return getModal(page).getByRole('button', { name: 'Create Role' });
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
  const searchInput = page.getByPlaceholder('Cari permission...');
  await searchInput.fill(query);
}

async function clearSearch(page: Page) {
  const clearBtn = page.getByRole('button', { name: 'Clear search' });
  if (await clearBtn.isVisible()) {
    await clearBtn.click();
  } else {
    const searchInput = page.getByPlaceholder('Cari permission...');
    await searchInput.fill('');
  }
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
    await loginAsSuperadmin(page);
    await navigateToRoles(page);
  });

  // ── 1. Modal Open / Close ──────────────────────────────────────

  test('should open Create Role modal when clicking Create Role button', async ({ page }) => {
    await openCreateRoleModal(page);

    // Verify modal title
    await expect(page.getByRole('dialog', { name: 'Create New Role' })).toBeVisible();

    // Verify form fields exist
    await expect(page.getByLabel('Role Name')).toBeVisible();
    await expect(page.getByLabel('Description')).toBeVisible();

    // Verify permissions section exists — use exact match to avoid "No permissions assigned"
    await expect(getModal(page).getByText('Permissions', { exact: true })).toBeVisible();

    // Verify footer buttons
    await expect(getCancelButton(page)).toBeVisible();
    await expect(getSaveButton(page)).toBeVisible();
  });

  test('should close modal when clicking Cancel with no changes', async ({ page }) => {
    await openCreateRoleModal(page);
    await cancelModal(page);
    await expect(page.getByRole('dialog', { name: 'Create New Role' })).toBeHidden({ timeout: 5000 });
  });

  test('should close modal when pressing Escape with no changes', async ({ page }) => {
    await openCreateRoleModal(page);
    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog', { name: 'Create New Role' })).toBeHidden({ timeout: 5000 });
  });

  // ── 2. Form Validation ─────────────────────────────────────────

  test('should show inline error when role name is empty and blurred', async ({ page }) => {
    await openCreateRoleModal(page);
    // Type something then clear it to trigger validation
    const nameInput = page.locator('#role-name');
    await nameInput.fill('x');
    await nameInput.fill('');
    await nameInput.blur();

    await expect(page.getByText('Role name is required')).toBeVisible({ timeout: 5000 });
    await expect(nameInput).toHaveAttribute('aria-invalid', 'true');
  });

  test('should show error for duplicate role name', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, 'superadmin'); // already exists
    await expect(page.getByText('Role name already exists')).toBeVisible({ timeout: 5000 });
  });

  test('should disable save button when name is invalid', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, '');

    await expect(getSaveButton(page)).toBeDisabled();
  });

  test('should enable save button when name is valid', async ({ page }) => {
    await openCreateRoleModal(page);
    await fillRoleName(page, `TestRole${Date.now()}`);

    await expect(getSaveButton(page)).toBeEnabled();
  });

  // ── 3. Basic Role Creation ─────────────────────────────────────

  test('should create a role with name and description only (no permissions)', async ({ page }) => {
    const uniqueName = `testrole_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await fillRoleDescription(page, 'E2E test role with no permissions');
    await saveRole(page);

    // Modal should close
    await expect(page.getByRole('dialog', { name: 'Create New Role' })).toBeHidden({ timeout: 10000 });

    // Role should appear in the list
    await expect(page.getByText(uniqueName, { exact: true })).toBeVisible({ timeout: 10000 });
  });

  test('should create a role with specific permissions', async ({ page }) => {
    const uniqueName = `testrole_perm_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await fillRoleDescription(page, 'E2E test role with inventory permissions');

    // Expand Inventory group and select all
    await selectAllInGroup(page, 'Inventory');

    // Verify selected count updated
    await expect(getModal(page).getByText('3 of', { exact: false })).toBeVisible({ timeout: 5000 });

    await saveRole(page);

    // Modal should close
    await expect(page.getByRole('dialog', { name: 'Create New Role' })).toBeHidden({ timeout: 10000 });

    // Role should appear in the list
    await expect(page.getByText(uniqueName, { exact: true })).toBeVisible({ timeout: 10000 });
  });

  // ── 4. Permission Groups ───────────────────────────────────────

  test('should display all permission groups collapsed by default', async ({ page }) => {
    await openCreateRoleModal(page);

    // Check that group toggle buttons exist with aria-expanded="false"
    const userRoleToggle = getModal(page).getByRole('button', { name: 'Toggle User & Role permissions' });
    await expect(userRoleToggle).toBeVisible();
    await expect(userRoleToggle).toHaveAttribute('aria-expanded', 'false');

    const productToggle = getModal(page).getByRole('button', { name: 'Toggle Product permissions' });
    await expect(productToggle).toHaveAttribute('aria-expanded', 'false');
  });

  test('should expand and collapse permission groups', async ({ page }) => {
    await openCreateRoleModal(page);

    // Expand Inventory
    await expandGroup(page, 'Inventory');
    const toggle = getModal(page).getByRole('button', { name: 'Toggle Inventory permissions' });
    await expect(toggle).toHaveAttribute('aria-expanded', 'true');

    // Verify group body exists
    await expect(page.locator('#group-body-inventory')).toBeVisible();

    // Collapse it
    await collapseGroup(page, 'Inventory');
    await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  });

  test('should show Expand All / Collapse All toggle', async ({ page }) => {
    await openCreateRoleModal(page);

    const modal = getModal(page);
    const expandAllBtn = modal.getByText('Expand All');
    await expect(expandAllBtn).toBeVisible();

    // Expand all
    await expandAllBtn.click();
    await expect(modal.getByText('Collapse All')).toBeVisible();

    // All groups should be expanded
    const userRoleToggle = modal.getByRole('button', { name: 'Toggle User & Role permissions' });
    await expect(userRoleToggle).toHaveAttribute('aria-expanded', 'true');

    // Collapse all
    await modal.getByText('Collapse All').click();
    await expect(modal.getByText('Expand All')).toBeVisible();
    await expect(userRoleToggle).toHaveAttribute('aria-expanded', 'false');
  });

  // ── 5. Select All / Deselect All per Group ─────────────────────

  test('should select all permissions in a group', async ({ page }) => {
    await openCreateRoleModal(page);
    await selectAllInGroup(page, 'Inventory');

    // All 3 inventory permissions should be checked
    const groupBody = page.locator('#group-body-inventory');
    const checkboxes = groupBody.locator('input[type="checkbox"]');
    await expect(checkboxes).toHaveCount(3);

    for (let i = 0; i < 3; i++) {
      await expect(checkboxes.nth(i)).toBeChecked();
    }

    // Button should now say "Deselect All"
    await expect(getModal(page).getByRole('button', { name: 'Deselect all Inventory permissions' })).toBeVisible();
  });

  test('should deselect all permissions in a group', async ({ page }) => {
    await openCreateRoleModal(page);
    await selectAllInGroup(page, 'Inventory');
    await deselectAllInGroup(page, 'Inventory');

    const groupBody = page.locator('#group-body-inventory');
    const checkboxes = groupBody.locator('input[type="checkbox"]');
    for (let i = 0; i < 3; i++) {
      await expect(checkboxes.nth(i)).not.toBeChecked();
    }

    // Button should say "Select All" again
    await expect(getModal(page).getByRole('button', { name: 'Select all Inventory permissions' })).toBeVisible();
  });

  // ── 6. Search ──────────────────────────────────────────────────

  test('should filter permissions by search query', async ({ page }) => {
    await openCreateRoleModal(page);
    await searchPermissions(page, 'inventory');

    // Groups with matches should show their toggle buttons
    // (permissions are filtered within groups)
    await expect(getModal(page).getByRole('button', { name: 'Toggle Inventory permissions' })).toBeVisible();

    // Expand the group to see filtered permissions
    await expandGroup(page, 'Inventory');
    await expect(page.locator('#group-body-inventory')).toBeVisible();
  });

  test('should show empty state when no permissions match search', async ({ page }) => {
    await openCreateRoleModal(page);
    await searchPermissions(page, 'zzzznonexistent');

    await expect(page.getByText('Tidak ada permission yang cocok')).toBeVisible({ timeout: 5000 });
  });

  test('should clear search with X button', async ({ page }) => {
    await openCreateRoleModal(page);
    const searchInput = page.getByPlaceholder('Cari permission...');
    await searchInput.fill('inventory');
    await expect(searchInput).toHaveValue('inventory');

    // Clear via X button
    const clearBtn = page.getByRole('button', { name: 'Clear search' });
    await expect(clearBtn).toBeVisible();
    await clearBtn.click();
    await page.waitForTimeout(300);

    // Search should be empty
    await expect(searchInput).toHaveValue('', { timeout: 5000 });
  });

  test('should clear search with Escape key', async ({ page }) => {
    await openCreateRoleModal(page);
    const searchInput = page.getByPlaceholder('Cari permission...');
    await searchInput.fill('inventory');
    await expect(searchInput).toHaveValue('inventory');

    // Press Escape to clear
    await searchInput.press('Escape');
    await page.waitForTimeout(300);

    // Search should be empty
    await expect(searchInput).toHaveValue('', { timeout: 5000 });
  });

  // ── 7. Unsaved Changes Guard ───────────────────────────────────

  test('should show discard confirmation when canceling with unsaved changes', async ({ page }) => {
    const uniqueName = `testrole_unsaved_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await selectAllInGroup(page, 'Inventory');

    // Try to cancel
    await cancelModal(page);

    // Should show discard confirmation dialog
    await expect(page.getByRole('dialog', { name: 'Discard Changes?' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('You have unsaved changes')).toBeVisible();
  });

  test('should discard changes when confirming discard', async ({ page }) => {
    const uniqueName = `testrole_discard_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await selectAllInGroup(page, 'Inventory');

    // Cancel → discard confirmation
    await cancelModal(page);
    await expect(page.getByRole('dialog', { name: 'Discard Changes?' })).toBeVisible();

    // Confirm discard
    await page.getByRole('button', { name: 'Discard' }).click();

    // Both modals should close
    await expect(page.getByRole('dialog', { name: 'Discard Changes?' })).toBeHidden({ timeout: 5000 });
    await expect(page.getByRole('dialog', { name: 'Create New Role' })).toBeHidden({ timeout: 5000 });

    // Role should NOT have been created
    await expect(page.getByText(uniqueName, { exact: true })).toBeHidden();
  });

  test('should keep editing when canceling discard', async ({ page }) => {
    const uniqueName = `testrole_keep_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await selectAllInGroup(page, 'Inventory');

    // Cancel → discard confirmation
    await cancelModal(page);
    await expect(page.getByRole('dialog', { name: 'Discard Changes?' })).toBeVisible();

    // Keep editing
    await page.getByRole('button', { name: 'Keep Editing' }).click();

    // Discard modal closes, create modal stays open
    await expect(page.getByRole('dialog', { name: 'Discard Changes?' })).toBeHidden({ timeout: 5000 });
    await expect(page.getByRole('dialog', { name: 'Create New Role' })).toBeVisible();

    // Name should still be there
    await expect(page.locator('#role-name')).toHaveValue(uniqueName);
  });

  // ── 8. Selected Count ──────────────────────────────────────────

  test('should update selected count as permissions are toggled', async ({ page }) => {
    await openCreateRoleModal(page);
    const modal = getModal(page);

    // Initially 0 selected
    await expect(modal.getByText('0 of', { exact: false })).toBeVisible();

    // Select all in Inventory (3 permissions)
    await selectAllInGroup(page, 'Inventory');
    await expect(modal.getByText('3 of', { exact: false })).toBeVisible({ timeout: 5000 });

    // Select all in Sales (3 permissions)
    await selectAllInGroup(page, 'Sales');
    await expect(modal.getByText('6 of', { exact: false })).toBeVisible({ timeout: 5000 });

    // Deselect all in Inventory
    await deselectAllInGroup(page, 'Inventory');
    await expect(modal.getByText('3 of', { exact: false })).toBeVisible({ timeout: 5000 });
  });

  // ── 9. Group Count Badge ───────────────────────────────────────

  test('should show correct selected/total count per group', async ({ page }) => {
    await openCreateRoleModal(page);
    await expandGroup(page, 'Inventory');

    // Initially 0/3
    const inventoryGroup = page.locator('[data-group]').filter({ hasText: 'Inventory' });
    await expect(inventoryGroup.getByText('0/3')).toBeVisible();

    // Select one permission
    const groupBody = page.locator('#group-body-inventory');
    const checkboxes = groupBody.locator('input[type="checkbox"]');
    await checkboxes.first().check();

    // Should show 1/3
    await expect(inventoryGroup.getByText('1/3')).toBeVisible({ timeout: 5000 });
  });

  // ── 10. Full Flow: Create Role with Multiple Groups ────────────

  test('should create a role selecting permissions from multiple groups', async ({ page }) => {
    const uniqueName = `testrole_full_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await fillRoleDescription(page, 'Full flow test role');

    // Select all Inventory permissions (3)
    await selectAllInGroup(page, 'Inventory');

    // Select all Sales permissions (3)
    await selectAllInGroup(page, 'Sales');

    // Select all Dashboard permissions (1)
    await selectAllInGroup(page, 'Dashboard');

    // Verify count: 3 + 3 + 1 = 7
    await expect(getModal(page).getByText('7 of', { exact: false })).toBeVisible({ timeout: 5000 });

    await saveRole(page);

    // Modal closes
    await expect(page.getByRole('dialog', { name: 'Create New Role' })).toBeHidden({ timeout: 10000 });

    // Role appears in list
    await expect(page.getByText(uniqueName, { exact: true })).toBeVisible({ timeout: 10000 });

    // Verify the role card shows correct permission count
    const roleCard = page.locator('.card').filter({ hasText: uniqueName });
    await expect(roleCard.getByText('7 Perms')).toBeVisible({ timeout: 5000 });
  });

  // ── 11. Toast Feedback ─────────────────────────────────────────

  test('should show success toast after creating a role', async ({ page }) => {
    const uniqueName = `testrole_toast_${Date.now()}`;

    await openCreateRoleModal(page);
    await fillRoleName(page, uniqueName);
    await saveRole(page);

    // Success toast should appear
    await expect(page.getByText('Role created')).toBeVisible({ timeout: 10000 });
  });

  // ── 12. Stale State on Reopen ──────────────────────────────────

  test('should reset form state when reopening modal', async ({ page }) => {
    const name1 = `testrole_first_${Date.now()}`;

    // Create first role
    await openCreateRoleModal(page);
    await fillRoleName(page, name1);
    await selectAllInGroup(page, 'Inventory');
    await saveRole(page);
    await expect(page.getByRole('dialog', { name: 'Create New Role' })).toBeHidden({ timeout: 10000 });

    // Reopen modal
    await openCreateRoleModal(page);

    // Form should be reset
    await expect(page.locator('#role-name')).toHaveValue('');
    await expect(page.locator('#role-desc')).toHaveValue('');
    await expect(getModal(page).getByText('0 of', { exact: false })).toBeVisible();

    // All groups should be collapsed
    const toggle = getModal(page).getByRole('button', { name: 'Toggle User & Role permissions' });
    await expect(toggle).toHaveAttribute('aria-expanded', 'false');
  });
});
