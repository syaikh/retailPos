import { test, expect } from './fixtures';
import { API_BASE, FRONTEND_BASE, authHeader, getToken, loginUI, logoutUI, waitForAppReady } from './fixtures';

// ============================================================================
// SelectSearch dropdown close behaviors inside the "New Arrangement" modal
//
// The SelectSearch component (web/src/shared/ui/SelectSearch.svelte) lives
// inside the ArrangementsPage create-arrangement modal.  We verify three
// close mechanisms:
//   1. Toggle – clicking the button again closes the dropdown
//   2. Escape – closes the dropdown but NOT the parent modal
//   3. Click-outside – closes the dropdown but NOT the parent modal
// ============================================================================

test.describe('SelectSearch dropdown close behaviors', () => {
  let adminUser: { username: string; password: string };

  test.beforeAll(async ({ request }) => {
    // Create a store-scoped admin (role 2) with consignment permissions.
    // Seed users have NULL store_id which 403s on consignment list endpoints,
    // so we must create a fresh user scoped to store 1.
    const superToken = await getToken(request);
    const superHeaders = authHeader(superToken);
    const suffix = Date.now();
    const username = `e2essc${suffix}`;
    const password = 'e2eSelectSearch123';

    const createRes = await request.post(`${API_BASE}/api/admin/users`, {
      headers: superHeaders,
      data: {
        username,
        email: `${username}@retail-pos.local`,
        password,
        role_id: 2, // admin role – holds all consignment.* permissions
        store_id: 1,
      },
    });
    expect(createRes.ok(), `create user failed: ${createRes.status()} ${await createRes.text()}`).toBeTruthy();
    adminUser = { username, password };

    // Create a consignment supplier so the SelectSearch has at least one option
    const adminToken = await getToken(request, username, password);
    const adminHeaders = authHeader(adminToken);
    const supRes = await request.post(`${API_BASE}/api/suppliers`, {
      headers: adminHeaders,
      data: {
        name: `E2E SelectSearch Supplier ${suffix}`,
        is_consignment: true,
      },
    });
    expect(supRes.ok(), `create supplier failed: ${supRes.status()} ${await supRes.text()}`).toBeTruthy();
  });

  test.beforeEach(async ({ page }) => {
    await loginUI(page, adminUser.username, adminUser.password);
  });

  test.afterEach(async ({ page }) => {
    await logoutUI(page);
  });

  // -------------------------------------------------------------------------
  // Helpers
  // -------------------------------------------------------------------------

  /** Navigate to the consignment page and open the "New Arrangement" modal. */
  async function openNewArrangementModal(page: any) {
    await page.goto(`${FRONTEND_BASE}/consignment`);
    await waitForAppReady(page);

    // Click the "New Arrangement" button
    const newBtn = page.locator('button').filter({ hasText: /New Arrangement/ }).first();
    await expect(newBtn).toBeVisible({ timeout: 10000 });
    await newBtn.click();

    // The modal should appear
    const modal = page.getByRole('dialog', { name: 'New Arrangement' });
    await expect(modal).toBeVisible({ timeout: 5000 });
    return modal;
  }

  /**
   * Open the supplier SelectSearch dropdown inside the modal.
   * The first [aria-haspopup="listbox"] inside the modal is the supplier
   * SelectSearch; the second is the store SelectSearch.
   */
  async function openSupplierDropdown(page: any, modal: any) {
    const selectBtn = modal.locator('[aria-haspopup="listbox"]').first();
    await expect(selectBtn).toBeVisible({ timeout: 3000 });
    await selectBtn.click();

    const listbox = page.locator('[role="listbox"]');
    await expect(listbox).toBeVisible({ timeout: 3000 });
    return { selectBtn, listbox };
  }

  // -------------------------------------------------------------------------
  // Test 1: Toggle open/close on button click
  // -------------------------------------------------------------------------

  test('clicking the SelectSearch button toggles the dropdown open and closed', async ({ page }) => {
    const modal = await openNewArrangementModal(page);
    const { selectBtn } = await openSupplierDropdown(page, modal);

    // Dropdown is open (listbox visible)
    const listbox = page.locator('[role="listbox"]');
    await expect(listbox).toBeVisible();

    // Click the same button again → dropdown closes
    await selectBtn.click();
    await expect(listbox).not.toBeVisible({ timeout: 3000 });

    // Click again → dropdown re-opens
    await selectBtn.click();
    await expect(listbox).toBeVisible({ timeout: 3000 });

    // Modal is still visible throughout
    await expect(modal).toBeVisible();
  });

  // -------------------------------------------------------------------------
  // Test 2: Escape key closes dropdown but NOT the modal
  // -------------------------------------------------------------------------

  test('Escape key closes the SelectSearch dropdown without closing the modal', async ({ page }) => {
    const modal = await openNewArrangementModal(page);
    const { listbox } = await openSupplierDropdown(page, modal);

    // Dropdown is open
    await expect(listbox).toBeVisible();

    // Press Escape
    await page.keyboard.press('Escape');

    // Dropdown should be closed
    await expect(listbox).not.toBeVisible({ timeout: 3000 });

    // The modal must still be open
    await expect(modal).toBeVisible();
  });

  // -------------------------------------------------------------------------
  // Test 3: Click outside the dropdown closes it but NOT the modal
  // -------------------------------------------------------------------------

  test('clicking outside the SelectSearch dropdown closes it without closing the modal', async ({ page }) => {
    const modal = await openNewArrangementModal(page);
    const { listbox } = await openSupplierDropdown(page, modal);

    // Dropdown is open
    await expect(listbox).toBeVisible();

    // Click inside the modal panel but outside the SelectSearch container
    // (e.g. the modal header area or the footer)
    await modal.locator('h2').first().click();

    // Dropdown should be closed
    await expect(listbox).not.toBeVisible({ timeout: 3000 });

    // The modal must still be open
    await expect(modal).toBeVisible();
  });

  // -------------------------------------------------------------------------
  // Test 4: Re-opening after close clears the search input
  // -------------------------------------------------------------------------

  test('re-opening the dropdown after closing clears the search input', async ({ page }) => {
    const modal = await openNewArrangementModal(page);
    const { selectBtn } = await openSupplierDropdown(page, modal);

    // Type a search term
    const searchInput = page.locator('[role="listbox"] input[type="text"]');
    await expect(searchInput).toBeVisible();
    await searchInput.fill('XYZNOMATCH');

    // All options should be filtered out (no match for the search term)
    const filteredOpts = page.locator('[role="listbox"] [role="option"]');
    await expect(filteredOpts).toHaveCount(0, { timeout: 2000 });

    // Close via Escape
    await page.keyboard.press('Escape');
    await expect(page.locator('[role="listbox"]')).not.toBeVisible({ timeout: 3000 });

    // Re-open
    await selectBtn.click();
    await expect(page.locator('[role="listbox"]')).toBeVisible({ timeout: 3000 });

    // The search input should be empty (search was cleared on close)
    const reopenedInput = page.locator('[role="listbox"] input[type="text"]');
    await expect(reopenedInput).toHaveValue('');

    // All options should be visible again (no longer filtered)
    const options = page.locator('[role="listbox"] [role="option"]');
    const count = await options.count();
    expect(count).toBeGreaterThan(0);
  });

  // -------------------------------------------------------------------------
  // Test 5: Selecting an option closes the dropdown
  // -------------------------------------------------------------------------

  test('selecting an option from the dropdown closes it', async ({ page }) => {
    const modal = await openNewArrangementModal(page);
    await openSupplierDropdown(page, modal);

    // Click the first option
    const firstOption = page.locator('[role="listbox"] [role="option"]').first();
    await expect(firstOption).toBeVisible({ timeout: 3000 });
    await firstOption.click();

    // Dropdown should be closed after selection
    await expect(page.locator('[role="listbox"]')).not.toBeVisible({ timeout: 3000 });

    // The SelectSearch button should now display the selected label
    const selectBtn = modal.locator('[aria-haspopup="listbox"]').first();
    const selectedText = await selectBtn.textContent();
    expect(selectedText?.trim().length).toBeGreaterThan(0);

    // Modal is still open
    await expect(modal).toBeVisible();
  });

  // -------------------------------------------------------------------------
  // Test 6: Each SelectSearch closes independently
  // -------------------------------------------------------------------------

  test('each SelectSearch dropdown closes independently', async ({ page }) => {
    const modal = await openNewArrangementModal(page);

    // Open the supplier (first) dropdown
    const supplierBtn = modal.locator('[aria-haspopup="listbox"]').first();
    await supplierBtn.click();
    const listbox = page.locator('[role="listbox"]');
    await expect(listbox).toBeVisible({ timeout: 3000 });

    // Close it via Escape
    await page.keyboard.press('Escape');
    await expect(listbox).not.toBeVisible({ timeout: 3000 });

    // Open the store (second) dropdown
    const storeBtn = modal.locator('[aria-haspopup="listbox"]').nth(1);
    await storeBtn.click();
    await expect(listbox).toBeVisible({ timeout: 3000 });

    // Close it via Escape
    await page.keyboard.press('Escape');
    await expect(listbox).not.toBeVisible({ timeout: 3000 });

    // Modal remains open
    await expect(modal).toBeVisible();
  });
});
