# Retail POS System — User Manual

This is the end-user manual for the Retail POS System, published as a
standalone document so shop staff can read, print, or share it without the
developer material that used to precede it in this file. For installation,
deployment, API, and architecture documentation, see the
[project README](../../README.md).

## Table of Contents

1. [Getting Started](#1-getting-started)
   - [Roles and Permissions](#roles-and-permissions)
   - [Logging In](#logging-in)
   - [The Main Screen (Navigation)](#the-main-screen-navigation)
   - [Notifications](#notifications)
   - [Logging Out](#logging-out)
   - [User Preferences](#user-preferences)
   - [Application Settings (Superadmin Only)](#application-settings-superadmin-only)
2. [Dashboard](#2-dashboard)
3. [Point of Sale (POS)](#3-point-of-sale-pos)
   - [Before You Start: Open a Shift](#before-you-start-open-a-shift)
   - [Adding Items to the Cart](#adding-items-to-the-cart)
   - [Editing the Cart](#editing-the-cart)
   - [Holding and Recalling Sales](#holding-and-recalling-sales)
   - [Checkout & Payment](#checkout--payment)
   - [Customer Selection](#customer-selection)
   - [Reprinting a Receipt](#reprinting-a-receipt)
   - [Keyboard Shortcuts](#keyboard-shortcuts)
4. [Transaction History](#4-transaction-history)
5. [Shifts](#5-shifts)
6. [Products & Inventory](#6-products--inventory)
   - [Browsing Products](#browsing-products)
   - [Adding / Editing a Product](#adding--editing-a-product)
   - [Adjusting Stock](#adjusting-stock)
   - [Rack Stock (Stok Rak)](#rack-stock-stok-rak)
   - [Low Stock Alerts](#low-stock-alerts)
   - [Bulk Actions](#bulk-actions)
7. [Categories, Brands & Units of Measure](#7-categories-brands--units-of-measure)
8. [Customers & Customer Groups](#8-customers--customer-groups)
9. [Suppliers](#9-suppliers)
10. [Storage Locations](#10-storage-locations)
11. [Pricing Rules](#11-pricing-rules)
    - [Creating a Pricing Rule](#creating-a-pricing-rule)
    - [Approval Workflow](#approval-workflow)
    - [Simulating a Price](#simulating-a-price)
12. [Purchase Orders](#12-purchase-orders)
    - [Creating a Purchase Order](#creating-a-purchase-order)
    - [Confirming a Purchase Order](#confirming-a-purchase-order)
    - [Receiving Goods](#receiving-goods)
    - [Cancelling a Purchase Order](#cancelling-a-purchase-order)
13. [Stock Opname (Stock Count)](#13-stock-opname-stock-count)
    - [The 9-State Workflow](#the-9-state-workflow)
    - [Creating a Session](#creating-a-session)
    - [Assigning Counters](#assigning-counters)
    - [Opening & Counting](#opening--counting)
    - [Verification](#verification)
    - [Posting Adjustments](#posting-adjustments)
    - [Closing & Cancelling](#closing--cancelling)
    - [Adjustments Report](#adjustments-report)
14. [Konsinyasi Supplier](#14-konsinyasi-supplier)
    - [Core Concept — How Consignment Works](#core-concept--how-consignment-works)
    - [Prerequisites](#prerequisites)
    - [Navigating the Module](#navigating-the-module)
    - [Tab 1: Penerimaan (Receiving Goods)](#tab-1-penerimaan-receiving-goods)
    - [Tab 2: Terms (Price & Store Share)](#tab-2-terms-price--store-share)
    - [Tab 3: Retur Tertunda (Pending Return)](#tab-3-retur-tertunda-pending-return)
    - [Tab 4: Retur (Formal Return)](#tab-4-retur-formal-return)
    - [Tab 5: Settlement & Payout](#tab-5-penyelesaian-settlement--payout)
    - [Tab 6: Stok (Consignment Stock)](#tab-6-stok-consignment-stock)
    - [Quick Reference: Document Numbers](#quick-reference-document-numbers)
    - [Quick Reference: Stock Math](#quick-reference-stock-math)
    - [Complete Walkthrough — End-to-End Example](#complete-walkthrough--end-to-end-example)
15. [Reports](#15-reports)
16. [Store Management](#16-store-management)
17. [Administration](#17-administration)
    - [Users](#users)
    - [Roles & Permissions](#roles--permissions)
    - [Audit Logs](#audit-logs)
18. [Import & Export](#18-import--export)
19. [Appendix A: Role / Permission Matrix](#appendix-a-role--permission-matrix)
20. [Appendix B: Status Reference](#appendix-b-status-reference)

---

## 1. Getting Started

### Roles and Permissions

The system has six built-in roles. Your role determines which menus you see and which actions you can take.

| Role | Typical user | What they do |
|------|--------------|--------------|
| **Superadmin** | System owner | Everything, including user deletion, role management, and audit logs |
| **Manager** | Store administrator | Dashboard, POS sales, transactions, reports, shifts, purchase orders, stock opname; manages products, categories, customers, pricing rules, suppliers, stores, users and roles; views audit logs and app settings |
| **Supervisor** | Shift lead / store operator | POS sales, own shifts, receives purchase orders, verifies and posts stock opname, product/category/customer CRUD, pricing, consignment view/create/update/settle |
| **Cashier** | Front-line seller | POS, own transactions, own shifts, customer lookup, stock counting |
| **Inventory staff** | Warehouse/counter staff | Stock opname counting, inventory adjustments, storage locations |
| **Finance** | Finance officer | Records supplier payments and views financial reports (consignment view/pay, reporting) |

A complete permission-to-role matrix is in [Appendix A](#appendix-a-role--permission-matrix).

### Logging In

1. Open the Retail POS application in your browser.
2. On the login page, enter your **Username** and **Password**.
3. Click **Login** (or press Enter).

After a successful login you are taken to the screen appropriate for your role:

- **Cashier** → the **Shifts** page (you must open a shift before using the POS).
- **Inventory staff** → the **Stock Opname** page.
- **Everyone else** (superadmin, manager, supervisor, finance) → the **Dashboard**.

> Your session is active for the current browser tab only. If you close the browser, you will need to log in again.

### The Main Screen (Navigation)

The left sidebar contains the main navigation. What you see depends on your role:

**Main menu**
- **Dashboard** — today's revenue and quick access tiles.
- **Point of Sale** — the cash register (requires `sale.create`).
- **Transactions** — sales history.
- **Reports** — revenue analytics.
- **Shifts** — cash register shifts.
- **Purchase Orders** — purchasing from suppliers (not shown for cashier/inventory staff).
- **Stock Opname** — physical stock counting.

**Master Data** (collapsible group)
- Products, Categories, Brands, Units, Customers, Pricing Rules, Customer Groups, Suppliers, Storage Locations.

**Administration** (shown for manager/superadmin — finance sees only Audit Logs)
- Stores, Users, Roles, Audit Logs, Settings (Audit Logs also for finance).

Sidebar visibility by role:

- **Cashier** — Point of Sale, Transactions, Shifts. Before a shift is open the sidebar collapses to **Shifts** only; Point of Sale and Transactions appear once the shift is open.
- **Inventory staff** — Stock Opname.
- **Manager / Superadmin** — Dashboard, Point of Sale, Transactions, Reports, Shifts, Purchase Orders, Stock Opname, Konsinyasi, and Master Data (Products, Categories, Brands, Units, Customers, Pricing Rules, Customer Groups, Suppliers, Storage Locations), plus Administration (Stores, Users, Roles, Audit Logs, Settings).
- **Supervisor** — Dashboard, Transactions, Reports, Shifts, Purchase Orders, Stock Opname, Konsinyasi, and the same Master Data group as manager (no Administration; Point of Sale works via direct URL — the sidebar grouping for supervisors omits it).
- **Finance** — Dashboard, Transactions, Reports, Konsinyasi, and Administration → Audit Logs.

> The sidebar shows only the menus above, but a role can also navigate directly to a URL whose permission code it holds. For example a cashier can open the Stock Opname page by URL, because `stock_opname.view` is in the default cashier grant even though the sidebar does not link it (see §13).

At the top of the screen you'll find:
- The page title (breadcrumb).
- A **live Jakarta clock** and date.
- A **WebSocket status dot** (Online / Connecting… / Offline) — when it is Offline, live updates are paused.
- The **notification bell**.

At the bottom of the sidebar is your **username, role, and the Logout button**.

### Notifications

The bell shows live notifications as events happen, including:
- **Out-of-stock alerts** — a product whose stock has reached 0.
- **New sales** — when a transaction is completed.
- **Purchase order received** — when goods are received.
- **Stock opname events** — created / submitted / approved / needs recount / cancelled (requires `stock_opname.view`).

Clicking a notification jumps to the relevant page (e.g. the stock opname session, the transaction, or the product list filtered to low stock).

### Logging Out

1. Click **Logout** at the bottom of the sidebar.

> **Cashier note:** you cannot log out while a shift is open. Close your shift first (the Logout button shows the tooltip *"Close shift first"*).

### User Preferences

Each user can personalise their experience from the icon buttons next to your username at the bottom of the sidebar:

- **Theme** — Switch between **Light** and **Dark** mode. The setting is saved per user and applied on every login.
- **Language** — Choose the display language. The setting is saved per user and applied on every login.
- **Change Password** — set a new password (leads to `/account/password`).

> These preferences are stored server-side and follow you across devices.

### Application Settings (Superadmin Only)

Superadmins can configure global branding under **Administration → Settings**:

- **Store Name** — displayed in the sidebar, login page, and receipts.
- **Store Jargon** — a subtitle/tagline (e.g. "Management System").
- **Logo** — uploaded image shown on receipts and the login page.
- **Receipt Header** — custom text printed at the top of receipts.
- **Receipt Footer** — custom text printed at the bottom of receipts (default: "Terima kasih atas kunjungan Anda!").

Managers can view these settings but only superadmins can edit them.

---

## 2. Dashboard

The Dashboard gives you a live summary of the day:

- **Today's Revenue** — total revenue so far today (updates live as sales are completed).
- **Transactions** — how many sales were completed today.
- **Categories** — active product categories (superadmin / manager only).
- **Out of Stock** — products with zero stock, with an "Action required" or "All stock healthy" note (superadmin / manager / supervisor only).

Below the cards are **Quick Access** tiles that jump to Point of Sale, Inventory, Reports, and Administration.

---

## 3. Point of Sale (POS)

The POS is the cash register screen. It has two areas: a **product search panel** on the left and a **cart** on the right. On mobile the cart becomes a bottom sheet that you can show or hide.

### Before You Start: Open a Shift

If you are a **cashier**, you must open a shift first:

1. When you reach the POS without an open shift, you'll see *"Anda harus membuka sif terlebih dahulu"* (You must open a shift first) and be redirected to **Shifts**.
2. Click **Open Shift** and enter your **opening balance** — the amount of cash in the drawer at the start of the shift.
3. Confirm. You are taken to the POS.

The opening balance is used later to reconcile the drawer when you close the shift.

### Adding Items to the Cart

1. Press **F2** (or click) to focus the search box.
2. Type the product **name, SKU, or barcode**. Results update as you type.
3. **Enter** adds the first matching product to the cart. Alternatively, use **Arrow Up / Arrow Down** to highlight a product and press **Enter**, or click a row then the **Add** button (double-click a row also adds it).

The table shows **Product name / Stock / Price / Add**. The stock shown is the available quantity *after* subtracting what is already in your cart. Products with no stock have a disabled Add button. A colored stock badge tells you at a glance:

- Red `0` — out of stock
- Red — at or below the critical threshold
- Amber — at or below the warning threshold
- Green — healthy

When a product has an active pricing rule, the cart shows the discounted price in green with the original price struck through, and the name of the applied rule. Items whose price was frozen for an in-progress transaction show *"harga dibekukan"* (price frozen).

### Editing the Cart

- Use the **+ / −** buttons, or type a quantity in the box (limited to available stock).
- Click the **X** to remove an item.
- Press **ALT+Delete** to clear the entire cart.
- Press **F6** to hold (park) the sale for later.

The cart footer shows the **Subtotal (DPP)**, **PPN 11%** (when applicable), and **Total** above the pay button. Discounts are applied automatically by Pricing Rules — there is no manual discount entry at the register.

### Holding and Recalling Sales

You can park a sale and resume it later — stock is **not** reduced while held.

- **Hold:** press **F6** (or click Hold). The cart is saved and the toast *"Sale held"* appears.
- **Recall:** press **F7** (or click Recall) to open the **Held Sales** list. (F7 is context-sensitive: on the bare cart screen it opens Held Sales; inside the checkout modal it sets cash to the exact total.) Each entry shows `Cart #id`, the total, and the item count. Click **Recall** on one to restore it to the cart (*"Sale resumed"*).

### Checkout & Payment

1. Press **F4** or click the green **Bayar [F4]** (Pay) button. The **Pembayaran** (Payment) modal opens with a default **CASH** row already filled with the sale total.
2. Click the payment-method buttons to add an **Alokasi Pembayaran** (Payment Allocation) row for each method. The available methods are **Cash (CASH)**, **Card (CARD)**, **E-Wallet (E_WALLET)**, **Transfer (TRANSFER)**, and **QRIS**.
   - For non-cash methods a **No. Referensi** (Reference Number) field is pre-filled; you can edit it.
   - For cash, use the quick buttons **5rb / 10rb / 20rb / 50rb / 100rb** to add denominations, or press **F7 (Tepat)** (Exact) to set cash to exactly the total.
   - Use **Atur Ulang** (Reset) to zero a row and **Hapus semua** (Remove all) to remove all allocations.
3. Split payment is supported — add multiple allocations as long as they sum to the total.
4. Press **Enter** or click **Selesai** (Done) to complete the sale. This button is enabled when the allocations equal the total, or when cash exceeds it — the modal then shows the **Kembalian** (Change Due) amount. A non-cash allocation may never push the total past the sale total.
5. Press **Esc** (or click **Batal** (Cancel)) to cancel the checkout and return to the cart.

On success you'll see *"Sale completed"*, the cart clears, and a receipt is printed automatically according to the selected **print mode** (see below).

### Customer Selection

By default the sale is to **Walk-in / General**. To attach a customer:

1. Click the customer row in the checkout modal.
2. In the **Pilih Pelanggan** (Select Customer) dialog, search by **name or phone**, or choose **Walk-in / Umum** (Walk-in / General).
3. Selecting a customer applies that customer's group pricing to any item added **after** the selection. Items already in the cart keep the price they were added at — remove and re-add them to reprice.

### Reprinting a Receipt

After a sale, the cart footer shows **Print · {invoice number}**. Click it to reprint the last sale's receipt.

### Receipt Printing Modes

The POS prints receipts in one of two modes, chosen with the **Print** toggle in the cart (the gear icon opens a field to set the print-agent URL and a *Test* button):

- **Preview** (default) — renders the 58mm receipt overlay and opens the browser print dialog, which you confirm as before.
- **Silent** — sends the receipt straight to the local **print agent** (`tools/print-agent`), which routes it to a 58mm thermal printer (or, with no printer attached, to a file). No browser dialog appears, so the cashier never confirms a preview — the way real high-volume retail prints.

The mode is stored per browser. If the agent is unreachable in Silent mode, the POS shows a Retry / Dismiss message and does **not** fall back to the browser dialog — the cashier can reprint the receipt later from the transaction.

**Enforcing silent on registers (production).** Building the frontend with `VITE_PRINT_MODE=silent` makes silent the *locked* default: the mode toggle is hidden (a `Silent` badge is shown instead) and a previously stored `preview` preference in `localStorage` is ignored, so cashiers cannot revert a register back to preview. The agent-URL gear remains available so each register can still be pointed at its own local agent. Build with:

```bash
cd web && VITE_PRINT_MODE=silent npm run build
```

Use a non-`silent` build for development/back-office terminals where the preview dialog is still wanted.

Run the agent during development or testing (Go binary, no Node required):

```bash
cd tools/print-agent
PRINT_TRANSPORT=file go run ./cmd/print-agent   # writes the ESC/POS .bin stream to the temp dir
# or use the flag-driven launcher:
./print-agent.sh -t file -p 9123 -o /tmp/receipt-out
```

Use `PRINT_TRANSPORT=tcp` with `PRINT_TCP_ADDR=192.168.x.x:9100` for a network thermal printer, or `PRINT_TRANSPORT=serial` with `PRINT_SERIAL_DEVICE=/dev/ttyUSB0` for a USB-serial printer. The default agent URL and mode can also be set globally with `VITE_PRINT_AGENT_URL` / `VITE_PRINT_MODE` in `.env`.

### Setting up a register

A register is a self-contained terminal: one machine running the built frontend, its own local print agent on `localhost:9123`, and its own physical printer. Printing is decentralised — the browser talks to `localhost`, so each register prints to *its* printer and never crosses to another. To bring up a register:

```bash
# 1. Build the frontend with silent printing locked on (do this once per register build)
cd web && VITE_PRINT_MODE=silent npm run build

# 2. Start the print agent on the register machine (one per register)
cd tools/print-agent
./print-agent.sh -t tcp --tcp-addr 192.168.x.x:9100   # network 58mm thermal printer
# ./print-agent.sh -t serial --serial-device /dev/ttyUSB0   # USB-serial printer
# ./print-agent.sh -t file -p 9123 -o /tmp/receipt-out      # dev / no printer (writes .bin)

# 3. Start the app
./run-dev.sh                                   # backend (port 9095) — use your prod server in deployment
cd web && npm run preview                      # serve the built frontend (port 4173)
```

Repeat steps 2–3 on every register terminal. No central print configuration is needed; each agent is independent and the backend is not in the print path. Use a non-`silent` build (`npm run dev` or a plain `npm run build`) only for development/back-office terminals where the preview dialog is still wanted.

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| **F2** | Focus the product search |
| **Arrow Up / Down** | Move product selection |
| **Enter** | Add selected product (or first search result) |
| **F4** | Open checkout |
| **F6** | Hold the current sale |
| **F7** | Open Held Sales / Recall (on the cart screen) |
| **ALT+Delete** | Clear the cart |
| **F7** | Set cash to exact total (in checkout) |
| **Enter** | Finish checkout |
| **Esc** | Clear search / close modal / cancel checkout |

---

## 4. Transaction History

The **Transactions** page lists sales, newest first. Which view you get depends on the **report.view** permission:

- **Roles that hold `report.view`** (manager, finance, superadmin) see a single, store-wide list with **no tabs** — every cashier's sales are already included.
- **Roles without it** (cashier, inventory staff) get two tabs: **My Transactions** (own sales, the default) and **Find Transaction**.
- **Find Transaction** searches across *all* cashiers' sales. It requires the **`sale.lookup`** permission — cashier only by default (it is the one code superadmin does not hold). Results are a **redacted summary** — invoice number, date/time, cashier name, and total. Opening a result shows the itemised receipt (needed to reprint it) together with each payment method and amount; unit cost, payment reference numbers, and customer details are withheld in every view.

**Filtering**
- **Search** — by invoice number, product, or customer (an `INV-` prefix is ignored).
- **Payment method** — multi-select list (All methods or a specific method).
- **Amount range** — min/max in Rupiah.
- **Date range** — presets: Today, Yesterday, Last 7 Days, Last 30 Days, This Month, This Year, or a custom range. The default is **Last 30 Days**. All dates use Jakarta time.

For roles without `report.view`, the **My Transactions** tab shows only the cashier's own sales and **Find Transaction** is the only place cross-cashier sales appear.

**Viewing a transaction**
Click a row to open the **Transaction Details** drawer:
- Invoice number, date/time, customer, and payment methods (with per-method amounts and reference numbers).
- The item list with quantities, prices, and subtotals (original price struck through when discounted).
- Totals: **Hemat** (Savings), **Subtotal (DPP)**, **PPN 11%**, and **TOTAL**.
- Actions: **Print Receipt** (thermal receipt) and **Download Invoice** (a PDF invoice).

**Exporting**
Use the Export dropdown to download **CSV** or **Excel** of the filtered results (`transactions-YYYY-MM-DD`).

> Note: there is currently no void/refund feature in the system and purchases cannot be returned through the app. Sales left on hold (`parked`) and cancelled ones also appear in this list, and the list has no status column — identify them by the absence of an itemised drawer.

---

## 5. Shifts

The **Shifts** page manages cash drawer shifts.

**Cashiers** see and manage only their own shifts. **Managers/admins** see all shifts and can review them.

**Opening a shift** — see [Before You Start: Open a Shift](#before-you-start-open-a-shift).

**Closing a shift**
1. Click **Close Shift** (only available when you have an open shift).
2. Review the summary: Opening Balance, Cash Sales, Non-Cash Sales, Transactions, Total Sales, and **Expected Cash** (= opening + cash sales).
3. Enter the **Closing Balance** (actual cash counted in the drawer) using the cash breakdown grid.
4. A live **Discrepancy** indicator shows "Balanced" or the difference (surplus/shortage).
5. Add optional notes and confirm.

After closing, the shift shows a badge of `Closed`, or a warning badge if it **needs review** (when there is a discrepancy).

**Filters:** Status (Open/Closed), Review Status (Needs Review/Reviewed), and Discrepancy (Balanced/Surplus/Shortage).

**Manager controls (shift drawer):**
- **Review & Approve** — marks a closed shift as reviewed and approved.
- **Audit** — a "Surprise Audit" that compares the system's expected cash against an entered actual balance, recording the difference.

You can also **export** shifts to CSV or Excel.

---

## 6. Products & Inventory

The **Products** page (**Master Data → Products** in the sidebar; the Dashboard's **Inventory** quick-access tile opens the same page) is your product catalog and your stock-level screen.

### Browsing Products

- **Search** — by name, SKU, or barcode.
- **Kategori** (Category) — filter by one or more categories.
- **Status** — All / Active / Inactive / Archived.
- **Low Stock** toggle — show only products at or below the critical threshold.
- **Supplier** — filter to a specific supplier's products (arrived via the Suppliers page).
- Sortable columns and pagination (20 per page).

Active filter chips appear below the toolbar; use the **X** on a chip or **Clear all** to reset.

### Adding / Editing a Product

Click **Add Product** (superadmin, manager or supervisor) and fill in:

- **Name** (required), **SKU** (required), **Barcode** (optional)
- **Category** (required) — type to search existing categories
- **Brand**, **Unit**, **Tax Class** (e.g. PPN 11%)
- **Price (IDR)** (required), **Cost (IDR)**, **Stock** (required)
- **Description** (optional)
- **Status** — Draft / Active / Inactive / Discontinued (Archived is available to superadmin/manager)

On **edit**, a **Pricing Rules** panel lists the rules currently attached to the product (inactive rules are dimmed).

**Deleting a product** hides it from the catalog — the row is soft-deleted (`deleted_at` set) and its status becomes **Archived**, where it can still be found with the Archived filter and restored. Only superadmin and manager can delete.

### Adjusting Stock

To change the on-hand quantity of a product:

1. On the product row, open the **Adjust Stock** action.
2. Enter a **Quantity Change** — positive adds stock, negative reduces it.
3. Enter **Notes** — a reason is required (e.g. "damaged", "return", "found on shelf").
4. Click **Adjust Stock**.

This records an inventory adjustment; the note is kept as the reason.

> Stock is also changed automatically when a sale completes (reduced), when a purchase order is received (increased), and when a stock opname is posted.

### Rack Stock (Stok Rak)

Opening a product's **detail drawer** shows a **Stok Rak (Lokasi)** (Rack Stock (Location)) panel listing how much of the product sits in each storage location (rack/shelf). Rack rows are a *sub-account* of the global stock — set/transfer operations never change the global stock number.

- **Tambah Stok / Set** (Add Stock / Set) — records the exact quantity of the product in a chosen location (upsert; overwrites the current rack figure).
- **Transfer** — moves a quantity from one location to another (requires the source to have enough stock).

Rack stock is reconciled automatically when a **stock opname scoped to a storage location** is posted: the rack row is corrected to the physical count, and the global stock is recomputed from that count (see §13), so a rack count reconciles the sub-account with the global number.

### Low Stock Alerts

Thresholds are configured system-wide (defaults: warning 10, critical 5). Products at or below the critical level are highlighted in red and trigger a dashboard alert and a notification.

### Bulk Actions

Tick the checkboxes on rows to select products, then use the bulk bar to:
- **Change Status** — set selected products to Active / Inactive / Archived.
- **Export / Import** — see [Import & Export](#18-import--export).

---

## 7. Categories, Brands & Units of Measure

These master-data pages live under Master Data (Categories is also under Administration).

**Categories** (`/categories`)
- Create/edit/delete categories; the list shows each category's product count.
- Products reference categories by name.

**Brands** (`/brands`)
- Create/edit/delete brands (name, etc.), with export/import and an import history.

**Units of Measure** (`/units-of-measure`)
- Create/edit units (code, name, description, active). Units are used on product records (e.g. pcs, box, kg).

---

## 8. Customers & Customer Groups

### Customers (`/customers`)

**Search & filter:** by name, phone, or email; status (All/Active/Inactive); and customer group.

**Creating a customer**
Click **Add Customer** and fill in:
- **Name** (required), **Phone** (required, 7–20 digits/format), **Email** (required)
- Optional: **Address**, **Customer Group**, **Note**

**Editing** — change details and toggle the **Active** checkbox.

**Deactivating** — use the trash icon; the customer is hidden from active listings but their history is preserved. (Reactivation is done via the edit form's Active checkbox.)

**Bulk actions** — change status (Active/Inactive) or delete selected customers (history preserved).

There is no credit/balance feature; customers are used mainly to attach group pricing and to record who bought what.

### Customer Groups (`/customer-groups`)

Groups allow you to apply group-specific pricing at the POS.

**Create:** **Tambah Grup Pelanggan** (Add Customer Group) → **Group Name** (required), **Description**, and an avatar **color**.
**Edit / Delete / Duplicate:** via the row's kebab menu (Duplicate pre-fills the name as `{name} (Salinan)` (Copy)).
**View members:** kebab menu → **Lihat Anggota** (View Members) jumps to the Customers page filtered to that group; a **Kembali** (Back) banner returns to the group list.

Clicking a row opens a drawer with group details and an **activity history** (created/updated/deleted by whom and when).

At the POS, when a customer in a group is attached to a cart, the group's pricing rules apply automatically.

---

## 9. Suppliers

The **Suppliers** page manages the vendors you purchase from.

**Create:** **Add Supplier** → **Supplier Name** (required) and **Supplier Code** (required), plus optional contact person, phone, email, address, and notes.

**Edit:** change details and toggle **Active**.

**View:** the detail drawer shows supplier info and their products. A **products** link filters the Products page to this supplier (with a **Kembali ke Pemasok** (Back to Suppliers) banner).

**Consignment filter:** click **Konsinyasi** in the toolbar to show only suppliers flagged as consignment suppliers. This filter stacks with the Active/Inactive status filter. The URL `/suppliers?is_consignment=true` deep-links to this filtered view. When navigating from the Consignment module's **View Suppliers** link, a back arrow appears above the toolbar to return to the arrangements list.

Suppliers are used by Purchase Orders — when creating a PO you pick a supplier and choose only products linked to that supplier.

**Store scope (supplier vs. its terms):** a supplier is a single global record — one vendor, one contact
list, visible to every store that trades with it. Its **commercial terms are per store**: the unit cost
and lead time you record against a product are your store's own numbers.

- Terms left without a store are the **estate-wide default**, inherited by every store.
- A store can add its **own** terms for the same product/supplier pair; they override the default for
  that store alone and are invisible to the others.
- Editing or deleting terms that a store has inherited (rather than negotiated) is refused. To use
  your own price, add the product to the supplier **for your store** first, then edit that row.
- **Preferred supplier** is a per-store choice, so two stores can buy the same product from different
  suppliers. To prefer one, link it for your store before selecting it.

Only superadmin can edit the estate-wide default terms.

---

## 10. Storage Locations

**Storage Locations** (`/storage-locations`, Indonesian UI) are master data for where products are physically kept (racks/shelves), scoped to a **warehouse** or a **store**.

- **Search** by code or name; filter **Semua / Aktif / Nonaktif** (All / Active / Inactive).
- **Tambah Lokasi** (Add Location) requires **Kode** (Code, required, unique, e.g. `RAK-A-01`) and **Nama** (Name, required, e.g. `Rak A - Baris 1`), a scope (**Gudang**/Warehouse or **Toko**/Store), and optional **Catatan** (Notes).
- **Edit** and **Delete** via the row's action menu; bulk **Aktifkan / Nonaktifkan / Hapus** (Activate / Deactivate / Delete).

This is master data only for the location itself. Rack-level stock tracking is live — see **Rack Stock (Stok Rak)** in §6 — and rack-aware stock counts are available via the **Storage Location (Rack)** scope in §13.

---

## 11. Pricing Rules

Pricing Rules define special prices, promotions, and markups. They are applied automatically at the POS; the register shows the discounted price and the rule name.

**Rule types**
- **Default** — the product's base price.
- **Harga Khusus** (Special Price, `special_price`) — a specific price.
- **Promosi** (Promotion, `promotion`) — a discount or markup.

**Methods**
- **Harga Tetap** (Fixed Price, `fixed_price`) — set an exact price.
- **Diskon (%)** (Discount Percent, `discount_percent`) — percentage off.
- **Diskon (Rp)** (Discount Amount, `discount_amount`) — fixed amount off.
- **Markup (%)** (`markup_percent`) — percentage added.

### Creating a Pricing Rule

Click **Buat Aturan Harga** (Create Rule) and complete the five-step form:

1. **Informasi Aturan** (Rule Information) — Name (required), Price Type, Method, and Value.
2. **Kondisi** (Conditions) — minimum/maximum quantity (empty = unlimited), customer group (All Groups), outlet (All Outlets).
3. **Target** — choose products, categories, and/or brands (at least one target is required; leave unused fields empty).
4. **Jadwal** (Schedule) — **Semua Hari** (All Days) / individual day chips (Senin–Minggu) / an **Akhir Pekan** (Weekend) quick-select, active hours (Dari Jam–Sampai Jam) (From Hour – To Hour), and validity dates (empty = always).
5. **Ringkasan Aturan** (Rule Summary) — a live 12-row preview of the rule.

You can tick **"Boleh digabung (stacking)"** (Allow stacking) to allow the rule to combine with other rules.

On save, the system checks for **conflicts** with existing rules. If a conflict is found, a **Konflik Ditemukan** (Conflict Found) warning lists the conflicting rules and lets you choose **Tetap Simpan** (Save Anyway) or go back.

### Approval Workflow

Rules move through an approval workflow:

```
Pending → Approved
        ↘ Rejected
```

- A newly saved rule starts as `pending` and inactive (`055_pricing_rule_new_rows_start_pending.sql`); there is no separate submit step.
- **Approve** (`pricing.approve`) — approves a pending rule and makes it active. The rule's author cannot approve their own rule (`pricing_rules.created_by`).
- **Reject** (`pricing.approve`) — marks the rule `rejected`.
- The former `draft` status was retired in `057_pricing_retire_draft_status.sql`.

You can also **Edit**, **Duplikasi** (Duplicate), **Hapus** (Delete), and **Aktifkan/Nonaktifkan** (Enable/Disable) rules, and use the bulk bar.

**Filters:** search, All/Aktif/Nonaktif (All/Active/Inactive), approval status (Semua Persetujuan) (All Approvals), rule type, and method.

### Simulating a Price

The **Simulasi Harga** (Price Simulation) tool answers "what will this cost?":

1. Click **Simulasi Harga** (Simulate).
2. Select a **product** (type at least 2 characters), **Jumlah** (Quantity), **Customer Group**, and **Toko** (Store).
3. Click **Hitung** (Calculate).
4. The result shows the original price, the final price, and the rule applied (discounted, markup, or normal).

The Product edit form also shows the rules attached to each product.

---

## 12. Purchase Orders

The **Purchase Orders** page manages orders to suppliers. Statuses: **Draft → Confirmed → Partial Received → Fully Received**, or **Cancelled** (the backend also supports `waiting_approval`/`rejected` for the approval workflow).

### Creating a Purchase Order

Click **Create Purchase Order** (requires `purchase_order.create`). The form has two steps:

**Step 1 — PO Details**
- **Supplier** (required) — pick from your suppliers.
- **Expected Date** (required).
- **Payment Term** (required) — Cash on Delivery, Net 15/30/60/90, Due on Receipt, 50% Upfront 50% on Delivery, or a custom term.
- Optional: **Supplier Ref**, **Delivery Address**, **Notes**.

**Step 2 — Items**
- Choose products from the **supplier's product list** (products must be linked to the supplier). If none are linked, you'll see *"No products for this supplier"* — link products to the supplier first to add them
- For each item: **Product**, **Qty**, **Unit Cost**, and **Discount** (Rp). The subtotal is calculated automatically and the **Total** shown in the footer.
- Click **Save Draft** to save. The PO is created in **Draft** status. The PO is created in **Draft** status.

### Confirming a Purchase Order

While a PO is in **Draft** you can **Edit** it, **Confirm** it, or **Cancel** it. Confirming locks the order and makes it ready for receiving.

### Receiving Goods

When goods arrive (PO status **Confirmed** or **Partial Received**), click **Receive Goods**:

1. The **Receive Goods** modal lists each item with **Ordered**, **Remaining** (= ordered − received), and fields for **Qty Good** and **Qty Damaged**.
2. Enter how many units arrived in good condition and how many were damaged. The two are constrained so the total never exceeds the remaining quantity.
3. Add optional **Notes**, then **Create Goods Receipt**.

On success:
- A **Goods Receipt** is recorded with a **DO number** (Delivery Order) that is generated automatically — you'll see the toast *"Goods receipt DO-2026-000123 created successfully"*. The DO number also appears on the PO detail drawer.
- Good stock is added to inventory automatically. Damaged stock is not.
- The PO status is recalculated: **Partial Received** (some items still outstanding) or **Fully Received** (everything received).

You can receive goods in multiple batches until the PO is fully received. The PO detail drawer lists all DO numbers.

### Cancelling a Purchase Order

Use **Cancel PO** on a **Draft** or **Confirmed** PO (with confirmation). Once a PO is fully received it can no longer be cancelled.

---

## 13. Stock Opname (Stock Count)

Stock Opname is the physical stock count workflow. It produces an official record of actual vs. system stock, and — after approval — automatically adjusts inventory.

### The 9-State Workflow

```
Draft → Open → Counting → Verification → Approved → Posted → Closed
                          ↑                 |
                      needs_recount         |
                          ↓                 |
                      Counting ←------------┘
   (Cancelled can be reached from Draft / Open / Counting / needs_recount)
```

| State | Meaning |
|-------|---------|
| **Draft** | Session created, not yet opened |
| **Open** | Session opened, counters can start |
| **Counting** | Physical counts being entered |
| **Verification** | Counts submitted, waiting for review |
| **Needs Recount** | Verification found issues; back to counting |
| **Approved** | Verified and approved, ready to post |
| **Posted** | Adjustments applied to inventory (IA- document created) |
| **Closed** | Record finalized |
| **Cancelled** | Session voided |

### Creating a Session

1. On the Stock Opname page click **New Stock Opname**.
2. Optional **Title**.
3. Optional **Blind count** checkbox — *hide system quantities from counters* (counters only see physical numbers, so they are not biased).
4. Add one or more **Scopes** — pick a scope type (e.g. store, warehouse, category, product, etc.) and the specific value. A "manual" row covers **all active products**.
   - The session covers the union of the selected scopes. Sessions may run in parallel as long as they never count the same SKU.
   - **Storage Location (Rack)** is a scope type that counts the products sitting in one rack. It must be the *only* scope of the session. Expected quantities come from the rack's `product_stock` row (products with no rack row are expected at 0). When the session is **posted**, the rack row is corrected to the physical count, and the global stock is recomputed as *the old global minus the old rack figure (never below 0), plus the new rack count* — so a rack count reconciles the sub-account with the global number even when sales have caused the two to drift apart.
5. Optional **Notes**, then create.

### Assigning Counters

While the session is Draft/Open/Counting/Needs Recount, an assigner can **Assign Counter** — add counter users to the session. Only assigned counters can enter counts.

> By role: **Superadmin, manager, supervisor and inventory staff** create, assign, verify, post, and close sessions (only assigned counters enter counts). **Cashiers and inventory staff** hold the `stock_opname.count`/`stock_opname.submit` permissions and are the usual counters — an assigner adds them to the session before counting begins.

### Opening & Counting

- **Open Session** (from Draft) — requires a comment explaining *why this session is being opened*.
- **Start Counting** — begins the counting phase. A counter enters the **physical** quantity for each product using the **Count** button on each item row.
- Blind sessions hide system quantities during entry.

### Verification

- **Submit for Verification** (from Counting) — sends the results to a verifier.
- **Verify / Reject** (from Verification):
  - *"Verifying approves the count without changing inventory. Posting is a separate step."*
  - Rejecting returns the session to **Needs Recount**; counters then **Resume Counting** to re-enter it.
- **Request Recount** — returns the session to counting for corrections.
- **Resume Counting** — counters continue after a recount request.

### Posting Adjustments

After a session is **Approved**, an authorized user **Posts the Adjustment**:
- *"Posting applies the verified differences to inventory and creates an adjustment document (IA-…)."*
- The toast shows *"Adjustment IA-2026-000007 posted successfully"*.
- Stock is updated for every item with a difference, and the **Adjustments Report** records each IA- document.

Posting is deliberately separate from verification (separation of duties) — the person who verifies should not be the only one who posts.

### Closing & Cancelling

- **Close Session** (from Posted) — a confirmation dialog asks *"Close stock opname {number}?"*
- **Cancel** (from Draft/Open/Counting/Needs Recount) — a confirmation dialog asks *"Cancel stock opname {number}?"*

### Adjustments Report

**Stock Opname → Adjustments** (`/stock-opnames/adjustments`) lists all adjustment documents (IA-…):
- Search by adjustment/session number, filter **Posted/Reversed**.
- Columns: Adjustment number, Session, Status, Total Diff, Total Value, Created By, Created At.
- Rows link back to the source session.

> Sessions can be exported to CSV from the list (per-row **Export**) and from the detail page (**Export CSV**).

---

## 14. Konsinyasi Supplier

The **Konsinyasi** (Consignment Supplier) module (`/consignment`) manages consignment stock — goods owned by a supplier that sit on your shelves. You sell them at terms you agree on, and the supplier is paid only after the goods are sold and a settlement is processed.

### Core Concept — How Consignment Works

In a consignment arrangement, **the supplier owns the goods** until they are sold to a customer. Your store acts as the selling agent and keeps an agreed share (the "store share") of each sale. The supplier is paid only after you create a settlement for sold items.

Here is the lifecycle at a glance:

```
+-----------------------------------------------------------------------+
|                     CONSIGNMENT LIFECYCLE                              |
|                                                                       |
|  1. SETUP                                                             |
|     Mark supplier konsinyasi -> Create arrangement -> Set terms        |
|                                                                       |
|  2. RECEIVE STOCK                                                     |
|     Supplier delivers goods -> Record receipt (CR-YYYY-xxxxxx)        |
|     -> Available stock increases                                      |
|                                                                       |
|  3. SELL AT POS                                                       |
|     Customer buys consignment product -> Stock decreases              |
|     -> Sale item recorded as "unsettled"                              |
|                                                                       |
|  4. RETURN (if needed)                                                |
|     Damaged/expired items -> Record pending return                    |
|     -> Physically hand back -> Record formal return (RT-YYYY-xxxxxx)  |
|                                                                       |
|  5. SETTLE & PAY                                                      |
|     Review unsettled sales -> Create settlement (CS-YYYY-xxxxxx)      |
|     -> Record payout -> Supplier is paid                              |
+-----------------------------------------------------------------------+
```

**Key rules:**
- Only **one active arrangement** is allowed per supplier per store.
- An arrangement is auto-ended if the supplier has not visited for too long.
- Terms apply to stock not yet sold. They never change sales that already happened.
- Settlements cover **all** unsettled sales at once — there is no partial settlement.

### Prerequisites

Before using the consignment module, you need:

1. **A supplier marked as konsinyasi.** Go to **Suppliers** -> Add or Edit a supplier -> toggle **Supplier Konsinyasi** on. Only suppliers with this flag appear in the consignment module.
2. **Products available to consign.** Any active product can be added to the arrangement's Terms tab — no prior supplier-product link is needed. Products still held as store-owned stock, or already consigned by another supplier, are excluded from the picker.
3. **Permission.** Your role needs `consignment.view` at minimum. Creating, updating, settling, and paying each require separate permissions — see [Appendix A](#appendix-a-role--permission-matrix).

### Navigating the Module

Open **Konsinyasi** from the sidebar. You'll see the **Arrangements List** — a table of all consignment arrangements across your accessible stores.

**The list shows:**

| Column | Meaning |
|--------|---------|
| Supplier | Name of the consignment supplier |
| Status | **Aktif** (Active — can receive stock and sell) or **Berakhir** (Ended — read-only) |
| Terms | Number of products with agreed pricing terms |
| Last Visit | When the supplier last delivered goods |

**Filtering:**
- **Search bar** — type a supplier name to filter.
- **Status buttons** — toggle between **Semua** (All), **Aktif** (Active only), **Berakhir** (Ended only).

**Creating a new arrangement:**
1. Click **Kesepakatan Baru** (New Arrangement) (top-right).
2. In the modal, select the **Supplier** from the dropdown (only consignment suppliers appear).
3. The **Store** defaults to your current store (superadmin can change it; other roles are auto-assigned to their own store).
4. Click **Create**. The arrangement appears in the list with status **Aktif**.

> If no consignment suppliers appear in the dropdown, go to **Suppliers** first and toggle the **Supplier Konsinyasi** (Consignment Supplier) flag on the supplier you want to use.

**Opening an arrangement:**
Click the **Buka** (Open) button on any row. This opens the arrangement detail view with six tabs. A back arrow (< Kembali (Back)) at the top-left returns you to the list.

The arrangement header shows the supplier name, status badge (**Aktif** / **Berakhir**), and the last visit date.

> **First-time setup:** If the arrangement has no pricing terms yet, the Terms tab opens automatically with a warning banner: "Set pricing terms before receiving goods." Once terms are added, subsequent opens land on the Receiving tab as usual.

**View Suppliers:** the arrangement list header has a **Lihat Pemasok** (View Suppliers) button that navigates to the Suppliers page filtered to show only consignment suppliers. A back button on the Suppliers page returns you to the arrangements list.

---

### Tab 1: Penerimaan (Receiving Goods)

This tab records goods delivered by the supplier. It is the default tab when you open an existing arrangement that already has pricing terms set.

**The receipt history** shows all past receipts with columns: Receipt Number (CR-YYYY-xxxxxx), Date, Items count, and Total Value.

**Recording a new receipt:**

1. Click **Catat Penerimaan** (Record Receipt) (top-right).
2. The receipt form opens with one empty product line. For each line:
   - **Produk** (Product, required) — search and select the product.
   - **Jumlah Diterima** (Accepted Quantity, required, min 1) — the units you accept from the supplier (default: 1). Units you refuse are simply not entered; there is no separate rejected field.
3. If the product has a term, the agreed price per unit is displayed (e.g. "Rp 50,000 / unit"). If there is **no term** for the product, a yellow warning "Produk belum punya terms — tambahkan di tab Terms." (This product has no terms yet — add one in the Terms tab) appears — go to the Terms tab first to add one.
4. Click **+ Tambah Baris** (Add Row) to add more product lines.
5. Optionally add **Catatan** (Notes) at the bottom.
6. Click **Simpan** (Save) to save. A toast confirms the receipt number and the receipt appears in the history.

> **Stock impact:** accepted quantities are added to the supplier's consignment stock immediately.

---

### Tab 2: Terms (Price & Store Share)

Terms define the pricing agreement for each product on consignment. **You must set terms before receiving goods** for those products — otherwise the receipt form will warn you.

**The terms list** shows: Product (name + SKU), Price (per unit), and Store Share.

**Adding a term:**

1. Click **Tambah Term** (Add Term) (top-right).
2. Fill in the form:
   - **Produk** (Product, required) — search and select the product.
   - **Harga (Rp)** (Price, required) — the agreed consignment price for the unit, recorded on the receipt for stock valuation. It does **not** set the POS selling price; the settlement is calculated from the actual POS sale price.
   - **Jenis Share** (Share Type, required) — choose one:
     - **Persentase (%)** (Percentage) — the store keeps a percentage of each sale (e.g. 20%). Must be between 0 and 100 (exclusive).
     - **Nominal Tetap (Rp)** (Fixed Amount) — the store keeps a fixed Rp amount per unit sold. Must be greater than 0 **and less than the price**.
   - **Nilai Share** (Share Value) — the share value (percentage or Rp amount, depending on the type above).
3. Click **Simpan** (Save).

> **Example:** Price = Rp 50,000, Share Type = Persentase, Share Value = 20. When one unit is sold, the store keeps Rp 10,000 and the supplier is owed Rp 40,000.

> **Note:** Adding a term appends it to the list — existing terms are preserved. Remove one with **Hapus** (Delete) on its row. The system preserves all existing terms automatically when you add a new one — only the newly added term needs to be filled in.

---

### Tab 3: Retur Tertunda (Pending Return)

A **Retur Tertunda** (Pending Return) records items pulled off the display **before** they are physically handed back to the supplier. This removes them from available stock while keeping them as supplier ownership until the formal return.

**The list shows:** Product (name + SKU), Quantity, Reason, Status (**Terbuka** = Open; **Sudah Dikembalikan** = Returned, resolved by a formal return). Only open pending returns are listed., and Date.

**Recording a pending return:**

1. Click **Catat Retur Tertunda** (Record Pending Return) (top-right).
2. Fill in the form:
   - **Produk (dari stok konsinyasi)** (Product from Stock, required) — the dropdown shows only products with available consignment stock, along with their available quantity (e.g. "Kopi ABC (SKU-001) — Stok tersedia 50" (Available stock 50)).
   - **Jumlah** (Quantity, required) — how many units to pull from display. Cannot exceed the available stock (a "Maks: XX" (Max: XX) hint appears below the field).
   - **Alasan** (Reason, required) — pick one: **Rusak** (Damaged), **Kadaluarsa** (Expired), **Retur Pelanggan** (Customer Return), **Pengakhiran kesepakatan** (Arrangement termination), or **Lainnya** (Other).
   - **Catatan** (Notes, optional) — free-text notes.
3. Click **Simpan** (Save).

> **Stock impact:** the product's available stock decreases by the pending return quantity. The pending return quantity is tracked separately until a formal return is created.

**Clearing a pending return:** there is no cancel action. A pending return is cleared only by recording a formal return that references it (see Tab 4), which removes the quantity from the pending-return ledger.

---

### Tab 4: Retur (Formal Return)

A formal return records the **physical hand-back** of goods to the supplier. It generates an RT-YYYY-xxxxxx document and removes the items from supplier ownership entirely.

**The list shows:** Return Number (RT-YYYY-xxxxxx), Date, Item count, and Total quantity returned.

If there are open pending returns, a yellow notice appears at the top: *"X retur tertunda terbuka"* (X pending returns open) — reminding you to link them.

**Recording a formal return:**

1. Click **Catat Retur** (Record Return) (top-right).
2. The form opens with one empty product line. For each line:
   - **Produk** (Product, required) — search and select the product.
   - **Jml** (Quantity) — the quantity being returned.
   - **Alasan** (Reason) — pick one: **Rusak** (Damaged), **Kadaluarsa** (Expired), **Retur Pelanggan** (Customer Return), **Pengakhiran kesepakatan** (Arrangement termination), or **Lainnya** (Other).
   - **Tautkan ke retur tertunda (opsional)** (Link to Pending Return, optional) — if this return corresponds to an existing pending return, select it from the dropdown. This closes the pending return and reduces the pending_return_qty. You can leave it as "— Tanpa tautan —" (No link) if no pending return applies.
   - **Catatan** (Notes, optional) — notes for this line.
3. Click **+ Tambah Baris** (Add Row) to return multiple products at once.
4. Optionally add **Catatan Keseluruhan** (Overall Notes) at the bottom.
5. Click **Simpan** (Save). A toast confirms the return number (RT-YYYY-xxxxxx).

> **Stock impact:** the product's total consignment stock decreases by the returned quantity. If a pending return was linked, the pending_return_qty is also reduced.

---

### Tab 5: Penyelesaian (Settlement & Payout)

The settlement tab handles the financial side — calculating what you owe the supplier for sold goods and recording payments.

This tab has two sections:

#### Unsettled Sales Preview (top card)

This shows all completed POS sales of consignment items that have **not yet been settled** — i.e. items the supplier is still owed money for.

**The preview table shows per product:** Product name, Quantity sold, Unit Price, Subtotal, and Store Share amount.

**The footer row shows three totals:**
- **Total Penjualan** (Total Sales) — total sale value of unsettled items.
- **Hak Toko** (Store Share) — your store's total share.
- **Terhutang ke Pemasok** (Owed to Supplier) — the amount you owe the supplier (= Total Penjualan minus Hak Toko).

**Creating a settlement:**

1. Review the unsettled items in the preview.
2. Click **Buat Penyelesaian** (Create Settlement) (top-right). The button is disabled when there are no unsettled items.
3. A confirmation modal shows the number of items and the total payable amount.
4. Click **Buat Penyelesaian** (Create Settlement) to confirm. A toast confirms the settlement number (CS-YYYY-xxxxxx).
5. The unsettled preview clears (all items are now part of the settlement) and the settlement appears in the history below.

> Settlement covers **all** unsettled sales — you cannot settle only some items.

#### Settlement History (bottom card)

This lists all past settlements with columns: No. Penyelesaian (CS-YYYY-xxxxxx), Date, Total amount, and Status (**Menunggu Pembayaran** = Pending Payment / **Dibayar** = Paid).

**Recording a payout (paying the supplier):**

1. On a settlement with status **Menunggu Pembayaran** (Pending Payment), click **Bayar** (Pay).
2. The payout modal shows the outstanding amount at the top.
3. Fill in:
   - **Metode Pembayaran** (Payment Method, required) — select from the available payment methods (Cash, Card, E-Wallet, Transfer, QRIS, etc.).
   - **Nominal (Rp)** (Amount, required) — defaults to the full outstanding amount. You can enter a lower amount for partial payment; the settlement remains pending until fully paid.
   - **Referensi** (Reference Number, optional) — a reference number (e.g. transfer receipt number).
   - **Catatan** (Notes, optional) — notes about the payment.
4. Click **Bayar** (Pay). A toast confirms the payout number (CP-YYYY-xxxxxx).

> The settlement status changes to **Dibayar** (Paid) only when the total paid equals the total payable. Until then, the **Bayar** (Pay) button remains available for additional payments.

---

### Tab 6: Stok (Consignment Stock)

This is a read-only view of the consignment stock for this supplier.

**The stock table shows per product:**

| Column | Meaning |
|--------|---------|
| Product | Product name and SKU |
| Stok Tersedia (Available Stock) | Available quantity (can be sold) |
| Retur Tertunda (Pending Return) | Quantity pending return (pulled from display, not yet handed back) |

> **How stock changes:** Receipts increase available stock. POS sales decrease both total and available stock. Pending returns decrease available stock and increase pending_return_qty. Formal returns decrease total stock and decrease pending_return_qty.

---

### Quick Reference: Document Numbers

| Document | Format | Created When |
|----------|--------|-------------|
| Receipt (Penerimaan) | CR-YYYY-xxxxxx | You record goods received from the supplier |
| Return (Retur) | RT-YYYY-xxxxxx | You record goods physically handed back to the supplier |
| Settlement | CS-YYYY-xxxxxx | You create a settlement for sold, unsettled items |
| Payout (Pembayaran) | CP-YYYY-xxxxxx | You record a payment to the supplier |

Every number carries the **year it was created**, and the suffix starts at `000001` — so the first receipt of 2026 is `CR-2026-000001`.

### Quick Reference: Stock Math

| Event | Available Stock | Pending Return |
|-------|:---------------:|:--------------:|
| Receipt (goods received) | increases | — |
| POS Sale | decreases | — |
| Pending Return created | decreases | increases |
| Formal Return linked to a pending return | — | decreases |
| Formal Return with no pending return linked | decreases | — |

There is no separate "total stock" quantity — consignment stock is stored as exactly two numbers, **Available** and **Pending Return**. Their sum is the total.

### Complete Walkthrough — End-to-End Example

Here is a full example of a consignment flow for a supplier "Toko Kopi Maju":

**Step 1 — Setup**
1. Go to **Suppliers**. Create or edit "Toko Kopi Maju" and toggle **Supplier Konsinyasi** (Consignment Supplier) on.
2. Add the products this supplier will provide to the Terms tab (e.g. "Kopi Robusta 250g", "Teh Hijau 100g"). Any active product qualifies; no supplier-product link is required.
3. Go to **Konsinyasi** (Consignment Supplier). Click **Kesepakatan Baru** (New Arrangement), select "Toko Kopi Maju", and create.

**Step 2 — Set Terms**
1. Open the arrangement. Go to the **Terms** tab.
2. Add a term for "Kopi Robusta 250g": Price = Rp 45,000, Share = Persentase (Percentage) 25%.
3. Add a term for "Teh Hijau 100g": Price = Rp 25,000, Share = Nominal Tetap (Fixed Amount) Rp 5,000.

**Step 3 — Receive Goods**
1. Switch to the **Penerimaan** (Receiving) tab. Click **Catat Penerimaan** (Record Receipt).
2. Line 1: Kopi Robusta 250g, Jumlah Diterima = 98.
3. Line 2: Teh Hijau 100g, Jumlah Diterima = 200.
4. Click **Simpan** (Save). Receipt `CR-2026-000001` is created. Stock increases.

**Step 4 — Sell at POS**
1. A cashier sells 5 Kopi Robusta at the POS register. The sale completes normally.
2. The system automatically deducts 5 from consignment stock and records the sale as unsettled.
3. For each unit sold: store gets Rp 11,250 (25% of Rp 45,000), supplier is owed Rp 33,750.

**Step 5 — Handle a Return (if needed)**
1. 3 units of Teh Hijau are found expired on the shelf.
2. Go to the arrangement -> **Retur Tertunda** (Pending Return) tab -> **Catat Retur Tertunda** (Record Pending Return).
3. Product = Teh Hijau 100g, Jumlah (Quantity) = 3, Alasan (Reason) = Kadaluarsa (Expired). Save.
4. Later, when the supplier picks them up, go to **Retur** (Return) tab -> **Catat Retur** (Record Return).
5. Line: Teh Hijau 100g, Jumlah (Quantity) = 3, Alasan (Reason) = Kadaluarsa (Expired), Link = select the pending return. Save.

**Step 6 — Settle and Pay**
1. Go to the **Penyelesaian** (Settlement) tab. The preview shows 5 units of Kopi Robusta sold.
   - Total Penjualan (Total Sales): Rp 225,000 (5 x Rp 45,000)
   - Hak Toko (Store Share): Rp 56,250 (5 x Rp 11,250)
   - Terhutang ke Pemasok (Owed to Supplier): Rp 168,750
2. Click **Buat Penyelesaian** (Create Settlement) and confirm. Settlement `CS-2026-000001` is created.
3. Finance pays the supplier via bank transfer. Click **Bayar** (Pay) on `CS-2026-000001`.
4. Select Transfer, enter the full amount Rp 168,750, add the transfer reference number. Click **Bayar** (Pay).
5. Settlement status changes to **Dibayar** (Paid). Done.

---

## 15. Reports

The **Reports** page is the revenue analytics dashboard.

**Period selection**
- Quick periods: **Real-time**, **Yesterday**, **7 Days**, **30 Days**.
- Calendar periods: **Daily**, **Weekly**, **Monthly**, **Yearly** — pick the period on the calendar.
- All values use Jakarta time (GMT+07). The earliest available data is June 2023 and the maximum selectable period is yesterday.

**What you see**
- **KPI cards:** Total Revenue (with Peak hour or Projected), Total Orders, Avg Order Value, Peak Hour/Month or Avg per Day, and the **comparison %** against the previous period (e.g. *vs Yesterday*, *vs Previous 7 Days*).
- **Chart:** hourly (real-time/yesterday/daily), daily (7 days/30 days/weekly/monthly), or yearly (bar chart by month). Current period is sky blue, previous period is slate; the tooltip shows the difference.
- **Best/Worst** badges — the best and worst hour/date/month by revenue.
- **Data table** — period, revenue, previous period, change %, and orders.
- **Revenue by Pricing Type** — how much came from discount/wholesale/promotion/other rules.

**Export** — **Export to Excel** (`dashboard-YYYY-MM-DD.xlsx`) or **Export to PDF** (a formatted *Revenue Report* with chart, comparison, and data table).

---

## 16. Store Management

The **Stores** page (`/stores`, Indonesian UI) manages store branches.

- **Tambah Toko** (Add Store) launches the **onboarding wizard** — Details (name, address, phone) → Staff (creates one active user per required role: manager, supervisor, cashier, inventory_staff, finance) → Location (first storage location) → Stock (initial products with stock) → Readiness. Each step only advances once its API call succeeded, so a store is never left half-created.
- **Readiness** — every store row reflects `GET /api/stores/:id/readiness`: `ready: true` requires an active store with address + phone, at least one active user in each required role, ≥1 storage location, and ≥1 active product with stock. Blockers are listed per missing ingredient.
- **Edit** — change details and toggle **Aktif** (Active).
- **Delete** — the confirmation suggests deactivating instead of deleting.

`000_baseline.sql` seeds a placeholder **Default Store** (with five of the six default users assigned to it — superadmin is deliberately store-less). Replace its placeholder address/phone with real values, then add a storage location and some catalog stock: the readiness check only reports `storage_location` and `catalog` as blockers, because the seeded address and phone are already non-empty. [`docs/guides/first-time-installation.md`](first-time-installation.md) walks through the initial setup. Active stores are used elsewhere in the system (e.g. as a scope for storage locations and stock opname, and as the outlet filter for pricing rules).

---

## 17. Administration

The Administration group is shown to **manager** and **superadmin**, and to **finance** for Audit Logs only (requires the relevant `*.view` permissions; `audit.view` is held by superadmin, manager, and finance).

### Users

Manage login accounts:
- **Add User** — username (alphanumeric), email, **password** (min 8 characters), **role** (superadmin, manager, supervisor, cashier, inventory_staff, finance), **active** status, a **store** (required for operational roles), and an optional **reports-to** manager (or *None (top-level)*).
- **Edit** — change details, role, active status, or set a **new password** (leave blank to keep the current one).
- Deactivate or delete users. Deleting users is superadmin-only. The superadmin account's delete button is disabled in the UI, but this is not enforced server-side.

> Change your own password anytime at **`/account/password`** (lock icon in the sidebar). `000_baseline.sql` flags the six seeded accounts `must_change_password`, forcing them through it on first login. Administrators can still reset any user's password in User Management.

### Roles & Permissions

Custom roles let you grant exactly the right permissions:
- **Create Role** — Step 1: name + description. Step 2: tick permission checkboxes grouped by area (User & Role, Product, Category, Sales, Inventory, Customer, Report, Dashboard, POS, System), with group toggles, a permission counter, and search.
- **Edit / Duplicate** (`(copy)` suffix) / **Delete** via the row menu. System roles' Delete button is hidden in the UI (not enforced server-side), deleting roles requires superadmin, and manager can create and duplicate roles but not edit or delete them.
- Role permission changes take effect for members when their access token is next refreshed (or on their next login), not on their next request — permissions are baked into the token.

### Audit Logs

A read-only log of important actions (who did what and when), with filters for action, resource, and date range, plus export. Visible to superadmin, manager and finance; superadmin and manager can export.

---

## 18. Import & Export

Bulk import/export works across products, categories, brands, units, customers, pricing rules, and suppliers. Customer groups and stores have downloadable templates only — their import/export endpoints return **403** because no module permission is defined for them. The entry point is the **Bulk Actions** dropdown on each supported page.

**Exporting**
- **Export CSV** or **Export XLSX** downloads your current data.
- Use CSV for spreadsheet editing, XLSX if formatting matters.

**Importing**
1. **Download Template** — get the correct column structure.
2. **Fill Out the Template** — example filled files are available in [`docs/examples/`](../examples/).
3. **Upload and Preview** — the **Import Wizard** shows you a preview before anything is applied.
4. **Confirm the Import** — apply the valid rows; invalid rows are reported.
5. **Tracking Progress** — monitor the import in the wizard; finished imports appear under **Import History** (reachable from Bulk Actions → Import History).

Imports are processed with preview/validation before commit, so mistakes can be caught before data is changed.

---

## Appendix A: Role / Permission Matrix

Legend: ✓ full access · ◐ partial/limited · — no access

| Capability | Superadmin | Manager | Supervisor | Cashier | Inventory Staff | Finance |
|------------|:---:|:---:|:---:|:---:|:---:|:---:|
| Dashboard | ✓ | ✓ | ✓ | ✓ | — | ✓ |
| Point of Sale (create sale) | ✓ | ✓ | ✓ | ✓ | — | — |
| View transactions | ✓ | ✓ | ✓ | ✓ (own) | — | ✓ |
| Reports | ✓ | ✓ | ✓ | — | — | ✓ |
| Shifts — open/close own | ✓ | ✓ | ✓ | ✓ | — | — |
| Shifts — view/review all | ✓ | ✓ | ✓ | — | — | — |
| Products — view | ✓ | ✓ | ✓ | ✓ | — | — |
| Products — create/edit | ✓ | ✓ | ✓ | — | — | — |
| Products — delete | ✓ | ✓ | — | — | — | — |
| Inventory adjustment | ✓ | ✓ | ✓ | — | ✓ | — |
| Categories — view | ✓ | ✓ | ✓ | ✓ | — | — |
| Categories — create | ✓ | ✓ | ✓ | — | — | — |
| Categories — edit/delete | ✓ | ✓ | ✓ | — | — | — |
| Customers — view | ✓ | ✓ | ✓ | ✓ | — | — |
| Customers — create/update | ✓ | ✓ | ✓ | — | — | — |
| Customers — delete/export/import | ✓ | ✓ | ✓ | — | — | — |
| Customer groups — view | ✓ | ✓ | ✓ | ✓ | — | — |
| Customer groups — manage | ✓ | ✓ | ✓ | — | — | — |
| Suppliers (use module) | ✓ | ✓ | ✓ | — | — | — |
| Storage locations — view | ✓ | ✓ | ✓ | ◐ (URL only) | ✓ | — |
| Storage locations — manage (create/edit/delete) | ✓ | ✓ | — | — | ✓ | — |
| Pricing rules — create/manage | ✓ | ✓ | ◐ (view only) | ✓ (view) | — | — |
| Purchase orders — create/confirm/receive | ✓ | ✓ | ✓ | — | — | — |
| Stock opname — create/assign/verify/post/close | ✓ | ✓ | ✓ | — | ✓ | — |
| Stock opname — count/submit | ✓ | ✓ | ✓ | ✓ | ✓ | — |
| Stock opname — export/report | ✓ | ✓ | ✓ | — | ✓ | — |
| Konsinyasi — view | ✓ | ✓ | ✓ | — | — | ✓ |
| Konsinyasi — create/update terms | ✓ | ✓ | ✓ | — | — | — |
| Konsinyasi — settle | ✓ | ✓ | ✓ | — | — | — |
| Konsinyasi — pay supplier | ✓ | ✓ | — | — | — | ✓ |
| Stores — manage | ✓ | ◐ (update/delete, no create) | — | — | — | — |
| Users — create/edit | ✓ | ✓ | — | — | — | — |
| Users — delete | ✓ | — | — | — | — | — |
| Roles — create | ✓ | ✓ | — | — | — | — |
| Roles — update/delete | ✓ | — | — | — | — | — |
| Audit logs — view | ✓ | ✓ | — | — | — | ✓ |
| Audit logs — export | ✓ | ✓ | — | — | — | — |
| Application settings — view | ✓ | ✓ | — | — | — | — |
| Application settings — update | ✓ | — | — | — | — | — |
| Import/Export (product, category, customer) | ✓ | ✓ | ✓ (customer) | — | — | — |

> Permission codes are checked in real time. Even within a role, custom roles can be granted any subset of permissions (see [Roles & Permissions](#roles--permissions)). Exact permission codes per action: `dashboard.view`, `sale.create/view/lookup/detail/park`, `product.view/create/update/delete/export/import/history.view/cost.view`, `category.view/create/update/delete/export/import`, `customer.view/create/update/delete/export/import`, `customer_group.view/create/update/delete`, `pricing.view/create/update/approve/delete`, `purchase_order.view/create/update/confirm/receive/cancel/delete`, `shift.view/create/review/audit/cash_movement`, `report.view`, `inventory.adjust`, `stock_opname.view/create/assign/count/submit/verify/post/close/recount/cancel/export/report`, `storage_location.view/create/update/delete`, `consignment.view/create/update/settle/pay`, `supplier.view/create/update/delete`, `app_settings.view/update`, `store.view/create/update/delete`, `user.view/create/update/delete`, `role.view/create/update/delete`, `audit.view/export`, `receipt.print`. The Suppliers module has its own `supplier.*` codes: `supplier.view` is held by superadmin, manager, and supervisor; `supplier.create/update/delete` by superadmin and manager only.

---

## Appendix B: Status Reference

**Products:** `draft` · `active` · `inactive` · `discontinued` · `archived`

**Sales:** `completed` (plus internal cart states `open`/`held`/`checked_out`/`cancelled`/`expired`)

**Purchase Orders:** `draft` → `confirmed` → `partial_received` → `fully_received`, or `cancelled`. `waiting_approval` and `rejected` are defined in the domain and rendered by the UI, but no PO approval transition exists yet, so the backend never sets them

**Pricing Rules:** `pending` → `approved` / `rejected` (no `draft` — retired in `057`)

**Stock Opname:** `draft` · `open` · `counting` · `verification` · `needs_recount` · `approved` · `posted` · `closed` · `cancelled`

**Stock Opname Adjustments:** `posted` (the column is unconstrained; `reversed` appears only as a UI filter option and is never written)

**Shifts:** `open` · `closed` (review status: needs review / reviewed)

**Customers & Customer Groups:** `active` · `inactive`

**Stores, Storage Locations, Suppliers, Units of Measure:** `active` · `inactive`

**Konsinyasi Arrangements:** `active` · `ended`

**Konsinyasi Pending Returns:** `open` · `returned`

**Konsinyasi Settlements:** `pending_payment` · `paid`

**Konsinyasi Return Reasons:** `damaged` · `expired` · `customer_return` · `termination` · `other`

**Konsinyasi Share Types:** `percentage` · `fixed_amount`

