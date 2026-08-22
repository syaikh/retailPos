# Retail POS System — End-User Manual

This manual explains how to use every feature of the Retail POS System from a user's perspective. It covers logging in, selling at the Point of Sale (POS), managing products and inventory, purchasing, pricing, stock opname, shifts, reports, customers, suppliers, and administration.

All prices are shown in Indonesian Rupiah (Rp) and all date/time values are displayed in **Asia/Jakarta (UTC+7)**.

---

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
    - [Counting](#counting)
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
    - [Tab 5: Settlement & Payout](#tab-5-settlement--payout)
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

The system has five built-in roles. Your role determines which menus you see and which actions you can take.

| Role | Typical user | What they do |
|------|--------------|--------------|
| **Superadmin** | System owner | Everything, including user deletion, role management, and audit logs |
| **Admin** | Store administrator | Everything except deleting users/roles and audit logs (can create and edit users and roles) |
| **Manager** | Store/ops manager | Dashboard, transactions, reports, shifts, purchase orders, stock opname; manages products, categories, customers, pricing rules, suppliers; adjusts inventory; no POS register |
| **Cashier** | Front-line seller | POS, own transactions, own shifts, customer lookup, stock counting |
| **Staff** | Warehouse/counter staff | Products (view), stock opname counting |

A complete permission-to-role matrix is in [Appendix A](#appendix-a-role--permission-matrix).

### Logging In

1. Open the Retail POS application in your browser.
2. On the login page, enter your **Username** and **Password**.
3. Click **Login** (or press Enter).

After a successful login you are taken to the screen appropriate for your role:

- **Cashier** → the **Shifts** page (you must open a shift before using the POS).
- **Staff** → the **Products** page.
- **Superadmin / Admin / Manager** → the **Dashboard**.

> Your session is active for the current browser tab only. If you close the browser, you will need to log in again.

### The Main Screen (Navigation)

The left sidebar contains the main navigation. What you see depends on your role:

**Main menu**
- **Dashboard** — today's revenue and quick access tiles.
- **Point of Sale** — the cash register (not shown for manager/staff).
- **Transactions** — sales history.
- **Reports** — revenue analytics.
- **Shifts** — cash register shifts.
- **Purchase Orders** — purchasing from suppliers (not shown for cashier/staff).
- **Stock Opname** — physical stock counting.

**Master Data** (collapsible group)
- Products, Categories, Brands, Units, Customers, Pricing Rules, Customer Groups, Suppliers, Storage Locations.

**Administration** (shown for admin/superadmin)
- Stores, Users, Roles, Audit Logs (audit logs require superadmin).

Sidebar visibility by role:

- **Cashier** — Point of Sale, Transactions, Shifts.
- **Staff** — Stock Opname, and Master Data → Products.
- **Manager** — Dashboard, Transactions, Reports, Shifts, Purchase Orders, Stock Opname, Konsinyasi, and Master Data (Products, Categories, Brands, Units, Customers, Pricing Rules, Customer Groups, Suppliers).
- **Admin / Superadmin** — the full menu plus Administration (Stores, Users, Roles; Audit Logs and Settings are superadmin-only).

> The sidebar shows only the menus above, but a role can also navigate directly to a URL whose permission code it holds (for example a cashier who also has `stock_opname.view` can open the Stock Opname page).

At the top of the screen you'll find:
- The page title (breadcrumb).
- A **live Jakarta clock** and date.
- A **WebSocket status dot** (Online / Connecting… / Offline) — when it is Offline, live updates are paused.
- The **notification bell**.

At the bottom of the sidebar is your **username, role, and the Logout button**.

### Notifications

The bell shows live notifications as events happen, including:
- **Low stock alerts** — products below the critical threshold.
- **New sales** — when a transaction is completed.
- **Purchase order received** — when goods are received.
- **Stock opname events** — created / submitted / approved / needs recount / cancelled (requires `stock_opname.view`).

Clicking a notification jumps to the relevant page (e.g. the stock opname session, the transaction, or the product list filtered to low stock).

### Logging Out

1. Click **Logout** at the bottom of the sidebar.

> **Cashier note:** you cannot log out while a shift is open. Close your shift first (the Logout button shows the tooltip *"Close shift first"*).

### User Preferences

Each user can personalise their experience via the profile menu (click your username at the bottom of the sidebar):

- **Theme** — Switch between **Light** and **Dark** mode. The setting is saved per user and applied on every login.
- **Language** — Choose the display language. The setting is saved per user and applied on every login.

> These preferences are stored server-side and follow you across devices.

### Application Settings (Superadmin Only)

Superadmins can configure global branding under **Administration → Settings**:

- **Store Name** — displayed in the sidebar, login page, and receipts.
- **Store Jargon** — a subtitle/tagline (e.g. "Management System").
- **Logo** — uploaded image shown on receipts and the login page.
- **Receipt Header** — custom text printed at the top of receipts.
- **Receipt Footer** — custom text printed at the bottom of receipts (default: "Terima kasih atas kunjungan Anda!").

Admins can view these settings but only superadmins can edit them.

---

## 2. Dashboard

The Dashboard gives you a live summary of the day:

- **Today's Revenue** — total revenue so far today (updates live as sales are completed).
- **Transactions** — how many sales were completed today.
- **Total Products** — the number of units in the catalog.
- **Low Stock Alerts** — products that need attention ("Action required" or "All stock healthy").

Below the cards are **Quick Access** tiles that jump to Point of Sale, Inventory, Reports, and Administration.

---

## 3. Point of Sale (POS)

The POS is the cash register screen. It has two areas: a **product search panel** on the left and a **cart** on the right. On mobile the cart becomes a bottom sheet that you can show or hide.

### Before You Start: Open a Shift

If you are a **cashier**, you must open a shift first:

1. When you reach the POS without an open shift, you'll see *"Anda harus membuka shift terlebih dahulu"* and be redirected to **Shifts**.
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
- **Recall:** press **F5** (or click Recall) to open the **Held Sales** list. Each entry shows `Cart #id`, the total, and the item count. Click **Recall** on one to restore it to the cart (*"Sale resumed"*).

### Checkout & Payment

1. Press **F4** or click the green **Bayar [F4]** button. The **Pembayaran** (Payment) modal opens with a default **CASH** row of Rp 0.
2. Click the payment-method buttons to add an **Alokasi Pembayaran** (payment allocation) row for each method. The available methods are **Cash (CASH)**, **Card (CARD)**, **E-Wallet (E_WALLET)**, **Transfer (TRANSFER)**, and **QRIS**.
   - For non-cash methods a **No. Referensi** (reference number) field is pre-filled; you can edit it.
   - For cash, use the quick buttons **5rb / 10rb / 20rb / 50rb / 100rb** to add denominations, or press **F7 (Tepat)** to set cash to exactly the total.
   - Use **Reset** to zero a row and **Hapus semua** to remove all allocations.
3. Split payment is supported — add multiple allocations as long as they sum to the total.
4. Press **Enter** or click **Selesai** to complete the sale. This button is only enabled when the allocations equal the total.
5. Press **Esc** (or click **Batal**) to cancel the checkout and return to the cart.

On success you'll see *"Sale completed"*, the cart clears, and a receipt is printed automatically on the thermal receipt overlay.

### Customer Selection

By default the sale is to **Walk-in / General**. To attach a customer:

1. Click the customer row in the checkout modal.
2. In the **Pilih Customer** dialog, search by **name or phone**, or choose **Walk-in / Umum**.
3. Selecting a customer applies that customer's group pricing to the items in the cart.

### Reprinting a Receipt

After a sale, the cart footer shows **Print · {invoice number}**. Click it to reprint the last sale's receipt.

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| **F2** | Focus the product search |
| **Arrow Up / Down** | Move product selection |
| **Enter** | Add selected product (or first search result) |
| **F4** | Open checkout |
| **F5** | Open Held Sales / Recall |
| **F6** | Hold the current sale |
| **ALT+Delete** | Clear the cart |
| **F7** | Set cash to exact total (in checkout) |
| **Enter** | Finish checkout |
| **Esc** | Clear search / close modal / cancel checkout |

---

## 4. Transaction History

The **Transactions** page lists completed sales. It has two tabs:

- **My Transactions** (default) — your own sales only.
- **Find Transaction** — search across *all* cashiers' sales (cross-cashier). Available to roles granted the **`sale.lookup`** permission (cashier and manager by default). Results are a **redacted summary** — invoice number, cashier name, date/time, total, and status only. Items, cost, customer details, and payment tender/reference are **not** shown, so you can locate a co-worker's transaction (e.g. for a receipt reprint request) without exposing sensitive data.

**Filtering**
- **Search** — by invoice number, product, or customer (an `INV-` prefix is ignored).
- **Payment method** — multi-select list (All methods or a specific method).
- **Amount range** — min/max in Rupiah.
- **Date range** — presets: Today, Yesterday, Last 7 Days, Last 30 Days, This Month, This Year, or a custom range. The default is **Last 30 Days**. All dates use Jakarta time.

The **My Transactions** tab shows only the cashier's own sales. The **Find Transaction** tab is the only place cross-cashier sales appear, and only in redacted form.

**Viewing a transaction**
Click a row to open the **Transaction Details** drawer:
- Invoice number, date/time, customer, and payment methods (with per-method amounts and reference numbers).
- The item list with quantities, prices, and subtotals (original price struck through when discounted).
- Totals: **Hemat** (savings), **Subtotal (DPP)**, **PPN 11%**, and **TOTAL**.
- Actions: **Print Receipt** (thermal receipt) and **Download Invoice** (a PDF invoice).

**Exporting**
Use the Export dropdown to download **CSV** or **Excel** of the filtered results (`transactions-YYYY-MM-DD`).

> Note: there is currently no void/refund feature in the system. A transaction's status is `completed` and purchases cannot be returned through the app.

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
- **Review** — marks a closed shift as reviewed.
- **Audit** — a "Surprise Audit" that compares the system's expected cash against an entered actual balance, recording the difference.

You can also **export** shifts to CSV or Excel.

---

## 6. Products & Inventory

The **Products** page (Inventory → Products, or Master Data → Products) is your product catalog and your stock-level screen.

### Browsing Products

- **Search** — by name, SKU, or barcode.
- **Kategori** — filter by one or more categories.
- **Status** — All / Active / Inactive / Archived.
- **Low Stock** toggle — show only products at or below the critical threshold.
- **Supplier** — filter to a specific supplier's products (arrived via the Suppliers page).
- Sortable columns and pagination (20 per page).

Active filter chips appear below the toolbar; use the **X** on a chip or **Clear all** to reset.

### Adding / Editing a Product

Click **Add Product** (superadmin/admin only) and fill in:

- **Name** (required), **SKU** (required), **Barcode** (optional)
- **Category** (required) — type to search existing categories
- **Brand**, **Unit**, **Tax Class** (e.g. PPN 11%)
- **Price (IDR)** (required), **Cost (IDR)**, **Stock** (required)
- **Description** (optional)
- **Status** — Draft / Active / Inactive / Discontinued (Archived is available to admin/superadmin)

On **edit**, a **Pricing Rules** panel lists the rules currently attached to the product (inactive rules are dimmed).

**Deleting a product** is permanent and removes it from the catalog — only admin/superadmin can do it.

### Adjusting Stock

To change the on-hand quantity of a product:

1. On the product row, open the **Adjust Stock** action.
2. Enter a **Quantity Change** — positive adds stock, negative reduces it.
3. Enter **Notes** — a reason is required (e.g. "damaged", "return", "found on shelf").
4. Click **Adjust Stock**.

This records an inventory adjustment; the note is kept as the reason.

> Stock is also changed automatically when a sale completes (reduced), when a purchase order is received (increased), and when a stock opname is posted.

### Rack Stock (Stok Rak)

Opening a product's **detail drawer** shows a **Stok Rak (Lokasi)** panel listing how much of the product sits in each storage location (rack/shelf). Rack rows are a *sub-account* of the global stock — set/transfer operations never change the global stock number.

- **Tambah Stok / Set** — records the exact quantity of the product in a chosen location (upsert; overwrites the current rack figure).
- **Transfer** — moves a quantity from one location to another (requires the source to have enough stock).

Rack stock is reconciled automatically when a **stock opname scoped to a storage location** is posted: the rack row is corrected to the physical count, and the global stock is recomputed from that count (see §13), so a rack count reconciles the sub-account with the global number.

### Low Stock Alerts

Thresholds are configured system-wide (defaults: warning 10, critical 5). Products at or below the critical level are highlighted in red and trigger a dashboard alert and a notification.

### Bulk Actions

Tick the checkboxes on rows to select products, then use the bulk bar to:
- **Change Status** — set selected products to Active / Inactive / Archived.
- **Export / Import** — see [Import & Export](#17-import--export).

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

**Create:** **Tambah Group** → **Group Name** (required), **Description**, and an avatar **color**.
**Edit / Delete / Duplicate:** via the row's kebab menu (Duplicate pre-fills the name as `{name} (Salinan)`).
**View members:** kebab menu → **Lihat Anggota** jumps to the Customers page filtered to that group; a **Kembali** banner returns to the group list.

Clicking a row opens a drawer with group details and an **activity history** (created/updated/deleted by whom and when).

At the POS, when a customer in a group is attached to a cart, the group's pricing rules apply automatically.

---

## 9. Suppliers

The **Suppliers** page manages the vendors you purchase from.

**Create:** **Add Supplier** → **Supplier Name** (required) and **Supplier Code** (required), plus optional contact person, phone, email, address, and notes.

**Edit:** change details and toggle **Active**.

**View:** the detail drawer shows supplier info and their products. A **products** link filters the Products page to this supplier (with a **Kembali ke Suppliers** banner).

Suppliers are used by Purchase Orders — when creating a PO you pick a supplier and choose only products linked to that supplier.

---

## 10. Storage Locations

**Storage Locations** (`/storage-locations`, Indonesian UI) are master data for where products are physically kept (racks/shelves), scoped to a **warehouse** or a **store**.

- **Search** by code or name; filter **Semua / Aktif / Nonaktif**.
- **Tambah Lokasi** → **Kode** (required, e.g. `RAK-A-01`) and **Nama** (required, e.g. `Rak A - Baris 1`), a scope (**Gudang**/warehouse or **Toko**/store), and optional **Catatan** (notes).
- **Edit** and **Delete** via the row's action menu; bulk **Aktifkan / Nonaktifkan / Hapus**.

This is master data only for the location itself. Rack-level stock tracking is live — see **Rack Stock (Stok Rak)** in §6 — and rack-aware stock counts are available via the **Storage Location (Rack)** scope in §13.

---

## 11. Pricing Rules

Pricing Rules define special prices, promotions, and markups. They are applied automatically at the POS; the register shows the discounted price and the rule name.

**Rule types**
- **Default** — the product's base price.
- **Harga Khusus** (`special_price`) — a specific price.
- **Promosi** (`promotion`) — a discount or markup.

**Methods**
- **Harga Tetap** (`fixed_price`) — set an exact price.
- **Diskon (%)** (`discount_percent`) — percentage off.
- **Diskon (Rp)** (`discount_amount`) — fixed amount off.
- **Markup (%)** (`markup_percent`) — percentage added.

### Creating a Pricing Rule

Click **Tambah Rule** and complete the five-step form:

1. **Informasi Rule** — Name (required), Price Type, Method, and Value.
2. **Kondisi** (Conditions) — minimum/maximum quantity (empty = unlimited), customer group (All Groups), outlet (All Outlets).
3. **Target** — choose products, categories, and/or brands (at least one target is required; leave unused fields empty).
4. **Jadwal** (Schedule) — All Days / Weekdays / Weekend, active hours (Dari Jam–Sampai Jam), and validity dates (empty = always).
5. **Ringkasan Rule** — a live 12-row preview of the rule.

You can tick **"Boleh digabung (stacking)"** to allow the rule to combine with other rules.

On save, the system checks for **conflicts** with existing rules. If a conflict is found, a **Konflik Ditemukan** warning lists the conflicting rules and lets you choose **Tetap Simpan** (save anyway) or go back.

### Approval Workflow

Rules move through an approval workflow:

```
Draft → Pending → Approved
               ↘ Rejected
```

- **Ajukan** (Submit) — moves a draft to pending.
- **Approve** — approves a pending rule (making it active).
- **Reject** — rejects a pending rule (back to draft).

You can also **Edit**, **Duplikasi** (duplicate), **Hapus** (delete), and **Aktifkan/Nonaktifkan** (enable/disable) rules, and use the bulk bar.

**Filters:** search, All/Aktif/Nonaktif, approval status (Semua Approval), rule type, and method.

### Simulating a Price

The **Simulasi** (Simulation) tool answers "what will this cost?":

1. Click **Simulasi**.
2. Select a **product** (type at least 2 characters), **Jumlah** (quantity), **Customer Group**, and **Toko** (store).
3. Click **Hitung**.
4. The result shows the original price, the final price, and the rule applied (discounted, markup, or normal).

The Product edit form also shows the rules attached to each product.

---

## 12. Purchase Orders

The **Purchase Orders** page manages orders to suppliers. Statuses: **Draft → Confirmed → Partial Received → Fully Received**, or **Cancelled** (the backend also supports `waiting_approval`/`rejected` for the approval workflow).

### Creating a Purchase Order

Click **Create** (requires `purchase_order.create`). The form has two steps:

**Step 1 — PO Details**
- **Supplier** (required) — pick from your suppliers.
- **Expected Date** (required).
- **Payment Term** (required) — Cash on Delivery, Net 15/30/60/90, Due on Receipt, 50% Upfront 50% on Delivery, or a custom term.
- Optional: **Supplier Reference Number**, **Delivery Address**, **Notes**.

**Step 2 — Items**
- Choose products from the **supplier's product list** (products must be linked to the supplier). If none are linked, you'll see *"No products available for this supplier. Link products to the supplier first."*
- For each item: **Product**, **Qty**, **Unit Cost**, and **Discount** (Rp). The subtotal is calculated automatically and the **Total** shown in the footer.
- Click **Create Draft** to save. The PO is created in **Draft** status.

### Confirming a Purchase Order

While a PO is in **Draft** you can **Edit** it, **Confirm** it, or **Cancel** it. Confirming locks the order and makes it ready for receiving.

### Receiving Goods

When goods arrive (PO status **Confirmed** or **Partial Received**), click **Receive**:

1. The **Receive Goods** modal lists each item with **Ordered**, **Remaining** (= ordered − received), and fields for **Qty Good** and **Qty Damaged**.
2. Enter how many units arrived in good condition and how many were damaged. The two are constrained so the total never exceeds the remaining quantity.
3. Add optional **Notes**, then **Create Goods Receipt**.

On success:
- A **Goods Receipt** is recorded with a **DO number** (Delivery Order) that is generated automatically — you'll see the toast *"Goods receipt created (DO-…)"*. The DO number also appears on the PO detail drawer.
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

> By role: **Manager/admin/superadmin** create, assign, verify, post, and close sessions (they cannot enter counts). **Cashiers and staff** hold the `stock_opname.count`/`stock_opname.submit` permissions and are the usual counters — a manager assigns them to a session before counting begins.

### Opening & Counting

- **Open Session** (from Draft) — requires a comment explaining *why this session is being opened*.
- **Start Counting** — begins the counting phase. A counter enters the **physical** quantity for each product using the **Count** button on each item row.
- Blind sessions hide system quantities during entry.

### Verification

- **Submit for Verification** (from Counting) — sends the results to a verifier.
- **Verify / Reject** (from Verification):
  - *"Verifying approves the count without changing inventory. Posting is a separate step."*
  - Rejecting returns the session to counting.
- **Request Recount** — returns the session to counting for corrections.
- **Resume Counting** — counters continue after a recount request.

### Posting Adjustments

After a session is **Approved**, an authorized user **Posts the Adjustment**:
- *"Posting applies the verified differences to inventory and creates an adjustment document (IA-…)."*
- The toast shows *"Adjustment {number} posted — inventory adjusted"*.
- Stock is updated for every item with a difference, and the **Adjustments Report** records each IA- document.

Posting is deliberately separate from verification (separation of duties) — the person who verifies should not be the only one who posts.

### Closing & Cancelling

- **Close Session** (from Posted) — *"This finalises the record."*
- **Cancel** (from Draft/Open/Counting/Needs Recount) — *"This cannot be undone."*

### Adjustments Report

**Stock Opname → Adjustments** (`/stock-opnames/adjustments`) lists all adjustment documents (IA-…):
- Search by adjustment/session number, filter **Posted/Reversed**.
- Columns: Adjustment number, Session, Status, Total Diff, Total Value, Created By, Created At.
- Rows link back to the source session.

> Sessions can be exported to CSV from the list (per-row **Export CSV**) and from the detail page.

---

## 14. Konsinyasi Supplier

The **Konsinyasi Supplier** module (`/consignment`) manages consignment stock — goods owned by a supplier that sit on your shelves. You sell them at terms you agree on, and the supplier is paid only after the goods are sold and a settlement is processed.

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
|     Supplier delivers goods -> Record receipt (CR-xxxxxx)             |
|     -> Available stock increases                                      |
|                                                                       |
|  3. SELL AT POS                                                       |
|     Customer buys consignment product -> Stock decreases              |
|     -> Sale item recorded as "unsettled"                              |
|                                                                       |
|  4. RETURN (if needed)                                                |
|     Damaged/expired items -> Record pending return                    |
|     -> Physically hand back -> Record formal return (RT-xxxxxx)       |
|                                                                       |
|  5. SETTLE & PAY                                                      |
|     Review unsettled sales -> Create settlement (CS-xxxxxx)           |
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
2. **Products linked to that supplier.** The supplier must have products assigned to it (done via the Supplier detail or Product edit page).
3. **Permission.** Your role needs `consignment.view` at minimum. Creating, updating, settling, and paying each require separate permissions — see [Appendix A](#appendix-a-role--permission-matrix).

### Navigating the Module

Open **Konsinyasi Supplier** from the sidebar. You'll see the **Arrangements List** — a table of all consignment arrangements across your accessible stores.

**The list shows:**

| Column | Meaning |
|--------|---------|
| Supplier | Name of the konsinyasi supplier |
| Status | **Aktif** (active — can receive stock and sell) or **Berakhir** (ended — read-only) |
| Terms | Number of products with agreed pricing terms |
| Last Visit | When the supplier last delivered goods |

**Filtering:**
- **Search bar** — type a supplier name to filter.
- **Status buttons** — toggle between **Semua** (all), **Aktif** (active only), **Berakhir** (ended only).

**Creating a new arrangement:**
1. Click **Arrangement Baru** (top-right).
2. In the modal, select the **Supplier** from the dropdown (only konsinyasi suppliers appear).
3. The **Store** defaults to your current store (superadmin/admin can change it).
4. Click **Create**. The arrangement appears in the list with status **Aktif**.

> If no konsinyasi suppliers appear in the dropdown, go to **Suppliers** first and toggle the **Supplier Konsinyasi** flag on the supplier you want to use.

**Opening an arrangement:**
Click the **Buka** (Open) button on any row. This opens the arrangement detail view with six tabs. A back arrow (< Kembali) at the top-left returns you to the list.

The arrangement header shows the supplier name, status badge (**Aktif** / **Berakhir**), and the last visit date.

---

### Tab 1: Penerimaan (Receiving Goods)

This is the **default tab** when you open an arrangement. It records goods delivered by the supplier.

**The receipt history** shows all past receipts with columns: Receipt Number (CR-xxxxxx), Date, Items count, and Total Value.

**Recording a new receipt:**

1. Click **Catat Penerimaan** (top-right).
2. The receipt form opens with one empty product line. For each line:
   - **Produk** (required) — search and select the product.
   - **Dibawa** (Brought) — the quantity the supplier delivered (default: 1).
   - **Ditolak** (Rejected) — units you refuse (damaged, wrong item, etc.). Default: 0.
   - **Accepted** = Dibawa minus Ditolak (shown automatically below the fields).
3. If the product has a term, the agreed price per unit is displayed (e.g. "Terms: Rp 50,000 per unit"). If there is **no term** for the product, a yellow warning "Belum ada terms" appears — go to the Terms tab first to add one.
4. Click **+ Tambah Baris** to add more product lines.
5. Optionally add **Catatan** (notes) at the bottom.
6. Click **Simpan** to save. A toast confirms the receipt number and the receipt appears in the history.

> **Stock impact:** accepted quantities are added to the supplier's consignment stock immediately.

---

### Tab 2: Terms (Price & Store Share)

Terms define the pricing agreement for each product on consignment. **You must set terms before receiving goods** for those products — otherwise the receipt form will warn you.

**The terms list** shows: Product (name + SKU), Price (per unit), and Store Share.

**Adding a term:**

1. Click **Tambah Term** (top-right).
2. Fill in the form:
   - **Produk** (required) — search and select the product.
   - **Harga (Rp)** (required) — the agreed retail price per unit.
   - **Jenis Share** (required) — choose one:
     - **Persentase (%)** — the store keeps a percentage of each sale (e.g. 20%). Must be between 0 and 100 (exclusive).
     - **Nominal Tetap (Rp)** — the store keeps a fixed Rp amount per unit sold. Must be greater than 0.
   - **Nilai Share** — the share value (percentage or Rp amount, depending on the type above).
3. Click **Simpan**.

> **Example:** Price = Rp 50,000, Share Type = Persentase, Share Value = 20. When one unit is sold, the store keeps Rp 10,000 and the supplier is owed Rp 40,000.

> **Note:** When you save terms, the entire list is replaced. The system preserves all existing terms automatically when you add a new one — only the newly added term needs to be filled in.

---

### Tab 3: Retur Tertunda (Pending Return)

A **Retur Tertunda** (Pending Return) records items pulled off the display **before** they are physically handed back to the supplier. This removes them from available stock while keeping them as supplier ownership until the formal return.

**The list shows:** Product (name + SKU), Quantity, Reason, Status (**Terbuka** = open / **Diproses** = returned/fulfilled), and Date.

**Recording a pending return:**

1. Click **Catat Retur Tertunda** (top-right).
2. Fill in the form:
   - **Produk dari Stok** (required) — the dropdown shows only products with available consignment stock, along with their available quantity (e.g. "Kopi ABC (SKU-001) — Stok tersedia 50").
   - **Jumlah** (required) — how many units to pull from display. Cannot exceed the available stock (a "Maks: XX" hint appears below the field).
   - **Alasan** (required) — pick one: **Rusak** (damaged), **Kadaluarsa** (expired), **Retur Pelanggan** (customer return), or **Lainnya** (other).
   - **Catatan** (optional) — free-text notes.
3. Click **Simpan**.

> **Stock impact:** the product's available stock decreases by the pending return quantity. The pending return quantity is tracked separately until a formal return is created.

**Cancelling a pending return:** A pending return in **Terbuka** (open) status can be cancelled via the API, which restores the available stock.

---

### Tab 4: Retur (Formal Return)

A formal return records the **physical hand-back** of goods to the supplier. It generates an RT-xxxxxx document and removes the items from supplier ownership entirely.

**The list shows:** Return Number (RT-xxxxxx), Date, Item count, and Total quantity returned.

If there are open pending returns, a yellow notice appears at the top: *"X retur tertunda terbuka"* — reminding you to link them.

**Recording a formal return:**

1. Click **Catat Retur** (top-right).
2. The form opens with one empty product line. For each line:
   - **Produk** (required) — search and select the product.
   - **Jumlah** — the quantity being returned.
   - **Alasan** — pick one: **Rusak**, **Kadaluarsa**, **Retur Pelanggan**, or **Lainnya**.
   - **Link ke Retur Tertunda** (optional) — if this return corresponds to an existing pending return, select it from the dropdown. This closes the pending return and reduces the pending_return_qty. You can leave it as "Tidak ada link" if no pending return applies.
   - **Catatan** (optional) — notes for this line.
3. Click **+ Tambah Baris** to return multiple products at once.
4. Optionally add **Catatan Keseluruhan** (overall notes) at the bottom.
5. Click **Simpan**. A toast confirms the return number (RT-xxxxxx).

> **Stock impact:** the product's total consignment stock decreases by the returned quantity. If a pending return was linked, the pending_return_qty is also reduced.

---

### Tab 5: Settlement & Payout

The settlement tab handles the financial side — calculating what you owe the supplier for sold goods and recording payments.

This tab has two sections:

#### Unsettled Sales Preview (top card)

This shows all completed POS sales of consignment items that have **not yet been settled** — i.e. items the supplier is still owed money for.

**The preview table shows per product:** Product name, Quantity sold, Unit Price, Subtotal, and Store Share amount.

**The footer row shows three totals:**
- **Total Penjualan** — total sale value of unsettled items.
- **Hak Toko** — your store's total share.
- **Terhutang ke Supplier** — the amount you owe the supplier (= Total Penjualan minus Hak Toko).

**Creating a settlement:**

1. Review the unsettled items in the preview.
2. Click **Buat Settlement** (top-right). The button is disabled when there are no unsettled items.
3. A confirmation modal shows the number of items and the total payable amount.
4. Click **Buat Settlement** to confirm. A toast confirms the settlement number (CS-xxxxxx).
5. The unsettled preview clears (all items are now part of the settlement) and the settlement appears in the history below.

> Settlement covers **all** unsettled sales — you cannot settle only some items.

#### Settlement History (bottom card)

This lists all past settlements with columns: Settlement Number (CS-xxxxxx), Date, Total amount, and Status (**Menunggu Pembayaran** = pending payment / **Dibayar** = paid).

**Recording a payout (paying the supplier):**

1. On a settlement with status **Menunggu Pembayaran**, click **Bayar**.
2. The payout modal shows the outstanding amount at the top.
3. Fill in:
   - **Metode Pembayaran** (required) — select from the available payment methods (Cash, Card, E-Wallet, Transfer, QRIS, etc.).
   - **Jumlah** (required) — defaults to the full outstanding amount. You can enter a lower amount for partial payment; the settlement remains pending until fully paid.
   - **No. Referensi** (optional) — a reference number (e.g. transfer receipt number).
   - **Catatan** (optional) — notes about the payment.
4. Click **Bayar**. A toast confirms the payout number (CP-xxxxxx).

> The settlement status changes to **Dibayar** only when the total paid equals the total payable. Until then, the **Bayar** button remains available for additional payments.

---

### Tab 6: Stok (Consignment Stock)

This is a read-only view of the consignment stock for this supplier.

**The stock table shows per product:**

| Column | Meaning |
|--------|---------|
| Product | Product name and SKU |
| Stok Tersedia | Available quantity (can be sold) |
| Retur Tertunda | Quantity pending return (pulled from display, not yet handed back) |

> **How stock changes:** Receipts increase available stock. POS sales decrease both total and available stock. Pending returns decrease available stock and increase pending_return_qty. Formal returns decrease total stock and decrease pending_return_qty.

---

### Quick Reference: Document Numbers

| Document | Format | Created When |
|----------|--------|-------------|
| Receipt (Penerimaan) | CR-xxxxxx | You record goods received from the supplier |
| Return (Retur) | RT-xxxxxx | You record goods physically handed back to the supplier |
| Settlement | CS-xxxxxx | You create a settlement for unsold items |
| Payout (Pembayaran) | CP-xxxxxx | You record a payment to the supplier |

### Quick Reference: Stock Math

| Event | Total Stock | Available Stock | Pending Return |
|-------|:-----------:|:---------------:|:--------------:|
| Receipt (goods received) | increases | increases | — |
| POS Sale | decreases | decreases | — |
| Pending Return created | — | decreases | increases |
| Pending Return cancelled | — | increases | decreases |
| Formal Return | decreases | — | decreases |

### Complete Walkthrough — End-to-End Example

Here is a full example of a consignment flow for a supplier "Toko Kopi Maju":

**Step 1 — Setup**
1. Go to **Suppliers**. Create or edit "Toko Kopi Maju" and toggle **Supplier Konsinyasi** on.
2. Link the products this supplier will provide (e.g. "Kopi Robusta 250g", "Teh Hijau 100g").
3. Go to **Konsinyasi Supplier**. Click **Arrangement Baru**, select "Toko Kopi Maju", and create.

**Step 2 — Set Terms**
1. Open the arrangement. Go to the **Terms** tab.
2. Add a term for "Kopi Robusta 250g": Price = Rp 45,000, Share = Persentase 25%.
3. Add a term for "Teh Hijau 100g": Price = Rp 25,000, Share = Nominal Tetap Rp 5,000.

**Step 3 — Receive Goods**
1. Switch to the **Penerimaan** tab. Click **Catat Penerimaan**.
2. Line 1: Kopi Robusta 250g, Dibawa = 100, Ditolak = 2. Accepted = 98.
3. Line 2: Teh Hijau 100g, Dibawa = 200, Ditolak = 0. Accepted = 200.
4. Click **Simpan**. Receipt CR-000001 is created. Stock increases.

**Step 4 — Sell at POS**
1. A cashier sells 5 Kopi Robusta at the POS register. The sale completes normally.
2. The system automatically deducts 5 from consignment stock and records the sale as unsettled.
3. For each unit sold: store gets Rp 11,250 (25% of Rp 45,000), supplier is owed Rp 33,750.

**Step 5 — Handle a Return (if needed)**
1. 3 units of Teh Hijau are found expired on the shelf.
2. Go to the arrangement -> **Retur Tertunda** tab -> **Catat Retur Tertunda**.
3. Product = Teh Hijau 100g, Jumlah = 3, Alasan = Kadaluarsa. Save.
4. Later, when the supplier picks them up, go to **Retur** tab -> **Catat Retur**.
5. Line: Teh Hijau 100g, Jumlah = 3, Alasan = Kadaluarsa, Link = select the pending return. Save.

**Step 6 — Settle and Pay**
1. Go to the **Settlement** tab. The preview shows 5 units of Kopi Robusta sold.
   - Total Penjualan: Rp 225,000 (5 x Rp 45,000)
   - Hak Toko: Rp 56,250 (5 x Rp 11,250)
   - Terhutang ke Supplier: Rp 168,750
2. Click **Buat Settlement** and confirm. Settlement CS-000001 is created.
3. Finance pays the supplier via bank transfer. Click **Bayar** on CS-000001.
4. Select Transfer, enter the full amount Rp 168,750, add the transfer reference number. Click **Bayar**.
5. Settlement status changes to **Dibayar**. Done.

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

- **Tambah Toko** → **Nama Toko** (required, e.g. "Cabang Bandung"), optional **Alamat** (address) and **Telepon** (phone).
- **Edit** — change details and toggle **Aktif**.
- **Delete** — the confirmation suggests deactivating instead of deleting.

Active stores are used elsewhere in the system (e.g. as a scope for storage locations and stock opname, and as the outlet filter for pricing rules).

---

## 17. Administration

The Administration group is shown only to **admin** and **superadmin** (Audit Logs is superadmin-only).

### Users

Manage login accounts:
- **Add User** — username (alphanumeric), email, **password** (min 6 characters), **role** (superadmin/admin/cashier/manager/staff), **active** status, and an optional **reports-to** manager (or *None (top-level)*).
- **Edit** — change details, role, active status, or set a **new password** (leave blank to keep the current one).
- Deactivate or delete users. The superadmin account cannot be deleted and deleting users is superadmin-only.

> There is no self-service "change password" screen. Passwords are set/reset by an administrator through User Management.

### Roles & Permissions

Custom roles let you grant exactly the right permissions:
- **Create Role** — Step 1: name + description. Step 2: tick permission checkboxes grouped by area (User & Role, Product, Category, Sales, Inventory, Customer, Report, Dashboard, POS, System), with group toggles, a permission counter, and search.
- **Edit / Duplicate** (`(copy)` suffix) / **Delete** via the row menu. System roles cannot be deleted, and deleting roles requires superadmin (admin can create, edit, and duplicate roles but not delete them).
- Role permission changes take effect for members on their next request.

### Audit Logs

A read-only log of important actions (who did what and when), with filters for action, resource, and date range, plus export. Superadmin only.

---

## 18. Import & Export

Bulk import/export works across several modules (products, categories, brands, units, customers, stores, and more where supported). The entry point is the **Bulk Actions** dropdown on each supported page.

**Exporting**
- **Export CSV** or **Export XLSX** downloads your current data.
- Use CSV for spreadsheet editing, XLSX if formatting matters.

**Importing**
1. **Download Template** — get the correct column structure.
2. **Fill Out the Template** — example filled files are available in `docs/examples/`.
3. **Upload and Preview** — the **Import Wizard** shows you a preview before anything is applied.
4. **Confirm the Import** — apply the valid rows; invalid rows are reported.
5. **Tracking Progress** — monitor the import in the wizard; finished imports appear under **Import History** (reachable from Bulk Actions → Import History).

Imports are processed with preview/validation before commit, so mistakes can be caught before data is changed.

---

## Appendix A: Role / Permission Matrix

Legend: ✓ full access · ◐ partial/limited · — no access

| Capability | Superadmin | Admin | Manager | Cashier | Staff |
|------------|:---:|:---:|:---:|:---:|:---:|
| Dashboard | ✓ | ✓ | ✓ | — | — |
| Point of Sale (create sale) | ✓ | ✓ | — | ✓ | — |
| View transactions | ✓ | ✓ | ✓ | ✓ (own) | — |
| Reports | ✓ | ✓ | ✓ | — | — |
| Shifts — open/close own | ✓ | ✓ | ✓ | ✓ | — |
| Shifts — view/review all | ✓ | ✓ | ✓ | — | — |
| Products — view | ✓ | ✓ | ✓ | ✓ | ✓ |
| Products — create/edit | ✓ | ✓ | ✓ (edit) | — | — |
| Products — delete | ✓ | ✓ | — | — | — |
| Inventory adjustment | ✓ | ✓ | ✓ | — | — |
| Categories — view/create | ✓ | ✓ | ✓ | — | — |
| Categories — edit/delete | ✓ | ✓ | — | — | — |
| Customers — view | ✓ | ✓ | ✓ | ✓ | — |
| Customers — create/update | ✓ | ✓ | ✓ | — | — |
| Customers — delete | ✓ | ✓ | — | — | — |
| Customer groups — view | ✓ | ✓ | ✓ | — | — |
| Customer groups — manage | ✓ | ✓ | — | — | — |
| Suppliers (use module) | ✓ | ✓ | ✓ | — | — |
| Storage locations — manage | ✓ | ✓ | — | — | — |
| Pricing rules — create/manage | ✓ | ✓ | ✓ | — | — |
| Purchase orders — create/confirm/receive | ✓ | ✓ | ✓ | — | — |
| Stock opname — create/assign/verify/post/close | ✓ | ✓ | ✓ | — | — |
| Stock opname — count/submit | ✓ | ✓ | — | ✓ | ✓ |
| Stock opname — export/report | ✓ | ✓ | ✓ | — | — |
| Konsinyasi — view | ✓ | ✓ | ✓ | — | — |
| Konsinyasi — create/update terms | ✓ | ✓ | ✓ | — | — |
| Konsinyasi — settle | ✓ | ✓ | ✓ | — | — |
| Konsinyasi — pay supplier | ✓ | ✓ | — | — | — |
| Stores — manage | ✓ | ✓ | — | — | — |
| Users — create/edit | ✓ | ✓ | — | — | — |
| Users — delete | ✓ | — | — | — | — |
| Roles — create/edit | ✓ | ✓ | — | — | — |
| Roles — delete | ✓ | — | — | — | — |
| Audit logs | ✓ | — | — | — | — |
| Application settings — view | ✓ | ✓ | — | — | — |
| Application settings — update | ✓ | — | — | — | — |
| Import/Export | ✓ | ✓ | — | — | — |

> Permission codes are checked in real time. Even within a role, custom roles can be granted any subset of permissions (see [Roles & Permissions](#roles--permissions)). Exact permission codes per action: `dashboard.view`, `sale.create/view/lookup`, `product.view/create/update/delete`, `category.view/create/update/delete`, `customer.view/create/update/delete`, `customer_group.view/create/update/delete`, `pricing.view/create/update/delete`, `purchase_order.view/create/update/confirm/receive/cancel`, `shift.view/create`, `report.view`, `inventory.adjust`, `stock_opname.view/create/assign/count/submit/verify/post/close/recount/cancel/export/report`, `storage_location.view/create/update/delete`, `consignment.view/create/update/settle/pay`, `app_settings.view/update`, `store.view/create/update/delete`, `user.view/create/update/delete`, `role.view/create/update/delete`, `audit.view`, `product.export/import`, `product.history.view`, `product.cost.view`, `category.export/import`, `customer.export/import`. The Suppliers module has no dedicated permission code — its page is gated by `pricing.view`, so superadmin, admin, and manager can use it.

---

## Appendix B: Status Reference

**Products:** `draft` · `active` · `inactive` · `discontinued` · `archived`

**Sales:** `completed` (plus internal cart states `open`/`held`/`checked_out`/`cancelled`/`expired`)

**Purchase Orders:** `draft` → `confirmed` → `partial_received` → `fully_received`, or `cancelled` (approval workflow: `waiting_approval`, `rejected`)

**Pricing Rules:** `draft` → `pending` → `approved` / `rejected`

**Stock Opname:** `draft` · `open` · `counting` · `verification` · `needs_recount` · `approved` · `posted` · `closed` · `cancelled`

**Stock Opname Adjustments:** `posted` · `reversed`

**Shifts:** `open` · `closed` (review status: needs review / reviewed)

**Customers & Customer Groups:** `active` · `inactive`

**Stores, Storage Locations, Suppliers, Units of Measure:** `active` · `inactive`

**Konsinyasi Arrangements:** `active` · `ended`

**Konsinyasi Pending Returns:** `open` · `returned`

**Konsinyasi Settlements:** `pending_payment` · `paid`

**Konsinyasi Return Reasons:** `damaged` · `expired` · `customer_return` · `other`

**Konsinyasi Share Types:** `percentage` · `fixed_amount`
