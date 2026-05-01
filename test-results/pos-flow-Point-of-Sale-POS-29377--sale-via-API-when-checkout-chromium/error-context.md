# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: pos-flow.spec.ts >> Point of Sale (POS) Module >> should create sale via API when checkout
- Location: tests/e2e/pos-flow.spec.ts:213:7

# Error details

```
Error: expect(received).toBeTruthy()

Received: false
```

# Test source

```ts
  140 |     const firstRow = page.locator('tbody tr').first();
  141 |     await firstRow.getByRole('button', { name: 'Add' }).click();
  142 | 
  143 |     // Remove item
  144 |     const removeBtn = page.locator('.max-h-96 button').filter({ hasText: '×' }).first();
  145 |     await removeBtn.click();
  146 | 
  147 |     // Cart should be empty
  148 |     await expect(page.getByText('Your cart is empty')).toBeVisible({ timeout: 5000 });
  149 |   });
  150 | 
  151 |   test('should display subtotal, tax, and total in cart', async ({ page }) => {
  152 |     await loginAndNavigateToPos(page);
  153 |     await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });
  154 | 
  155 |     // Add a product
  156 |     const firstRow = page.locator('tbody tr').first();
  157 |     await firstRow.getByRole('button', { name: 'Add' }).click();
  158 | 
  159 |     // Check cart summary
  160 |     await expect(page.getByText('Subtotal')).toBeVisible({ timeout: 5000 });
  161 |     await expect(page.getByText('Tax (10%)')).toBeVisible();
  162 |     await expect(page.getByText('Total', { exact: true })).toBeVisible();
  163 |   });
  164 | 
  165 |   test('should show Complete Purchase button', async ({ page }) => {
  166 |     await loginAndNavigateToPos(page);
  167 |     await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });
  168 | 
  169 |     const firstRow = page.locator('tbody tr').first();
  170 |     await firstRow.getByRole('button', { name: 'Add' }).click();
  171 | 
  172 |     await expect(page.getByRole('button', { name: 'Complete Purchase' })).toBeVisible({ timeout: 5000 });
  173 |   });
  174 | 
  175 |   test('should search products by name', async ({ page }) => {
  176 |     await loginAndNavigateToPos(page);
  177 |     await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });
  178 | 
  179 |     // Get initial count
  180 |     const initialRows = await page.locator('tbody tr').count();
  181 | 
  182 |     // Type in search box
  183 |     const searchInput = page.getByPlaceholder('Search products (name or SKU)...');
  184 |     await searchInput.fill('Nonexistent');
  185 | 
  186 |     // Should show "No products found"
  187 |     await expect(page.getByText(/No products found/)).toBeVisible({ timeout: 5000 });
  188 |   });
  189 | 
  190 |   test('should filter products by category', async ({ page }) => {
  191 |     await loginAndNavigateToPos(page);
  192 |     await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });
  193 | 
  194 |     // Check category dropdown exists
  195 |     const categorySelect = page.locator('select');
  196 |     await expect(categorySelect).toBeVisible();
  197 | 
  198 |     // Select a specific category
  199 |     await categorySelect.selectOption('Makanan');
  200 | 
  201 |     // Wait for filter to apply
  202 |     await page.waitForTimeout(500);
  203 | 
  204 |     // All visible products should be in Makanan category
  205 |     const categoryCells = page.locator('tbody tr .text-xs.text-slate-400');
  206 |     const count = await categoryCells.count();
  207 |     for (let i = 0; i < count; i++) {
  208 |       const text = await categoryCells.nth(i).textContent();
  209 |       expect(text?.trim()).toBe('Makanan');
  210 |     }
  211 |   });
  212 | 
  213 |   test('should create sale via API when checkout', async ({ page }) => {
  214 |     const tokenResponse = await page.request.post('/api/login', {
  215 |       data: {
  216 |         username: TEST_USERS.superadmin.username,
  217 |         password: TEST_USERS.superadmin.password
  218 |       }
  219 |     });
  220 |     const tokenData = await tokenResponse.json();
  221 |     const token = tokenData.access_token;
  222 | 
  223 |     const saleResponse = await page.request.post('/api/sales', {
  224 |       headers: { Authorization: `Bearer ${token}` },
  225 |       data: {
  226 |         invoice_number: `INV-${Date.now()}`,
  227 |         cashier_id: 1,
  228 |         store_id: 1,
  229 |         subtotal: 15000,
  230 |         discount: 0,
  231 |         tax: 1500,
  232 |         total_amount: 16500,
  233 |         payment_method: 'cash',
  234 |         items: [
  235 |           { product_id: 1, quantity: 1, unit_price: 15000, subtotal: 15000 }
  236 |         ]
  237 |       }
  238 |     });
  239 | 
> 240 |     expect(saleResponse.ok()).toBeTruthy();
      |                               ^ Error: expect(received).toBeTruthy()
  241 |     const sale = await saleResponse.json();
  242 |     expect(sale.data).toHaveProperty('id');
  243 |     expect(sale.data.invoice_number).toBeTruthy();
  244 |   });
  245 | 
  246 |   test('GET /api/products should return products list', async ({ page }) => {
  247 |     const tokenResponse = await page.request.post('/api/login', {
  248 |       data: {
  249 |         username: TEST_USERS.superadmin.username,
  250 |         password: TEST_USERS.superadmin.password
  251 |       }
  252 |     });
  253 |     const token = (await tokenResponse.json()).access_token;
  254 | 
  255 |     const response = await page.request.get('/api/products', {
  256 |       headers: { Authorization: `Bearer ${token}` }
  257 |     });
  258 |     expect(response.ok()).toBeTruthy();
  259 |     const data = await response.json();
  260 |     expect(data.data).toBeTruthy();
  261 |     expect(Array.isArray(data.data)).toBeTruthy();
  262 |     expect(data.data.length).toBeGreaterThanOrEqual(5);
  263 |   });
  264 | 
  265 |   test('should show disabled Add button for out-of-stock product', async ({ page }) => {
  266 |     await loginAndNavigateToPos(page);
  267 |     await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 });
  268 | 
  269 |     // Products with stock=0 should have disabled Add button
  270 |     // Since seeded products have stock, we just verify Add buttons exist
  271 |     const addButtons = page.locator('tbody tr button:has-text("Add")');
  272 |     const count = await addButtons.count();
  273 |     expect(count).toBeGreaterThanOrEqual(1);
  274 |   });
  275 | });
  276 | 
```