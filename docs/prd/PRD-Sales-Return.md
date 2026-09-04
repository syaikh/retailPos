# PRD: Sales Return (Retur Penjualan)

| Field | Nilai |
|-------|-------|
| Status | **Draft** |
| Date | 2026-09-04 |
| Author | Engineering Team |
| Scope | Customer return of purchased items at POS |

---

## 1. Purpose

Allow customers to return **damaged** items they have purchased and receive a **replacement** (same product, same price) when stock is available. If no stock is available (last item), a **full refund** to the original payment method is issued. This return-first policy ensures customer satisfaction while maintaining inventory accuracy, financial integrity, and fraud controls.

---

## 2. Business Context

- **Current policy** (roadmap `upcoming-features.md:576`): "Returns & Refund: Tidak akan diimplementasikan."
- **This PRD supersedes** that exclusion for the specific case of **damaged-item returns within 1 day**.
- The Find Transaction feature (`docs/design/find-transaction-requirements.md`) already provides the proof-of-purchase gate (invoice lookup) that is a prerequisite for returns.
- Return/void was previously marked **"On Hold — pending review"** (`find-transaction-requirements.md:144-148`). This PRD resolves that hold.

---

## 3. Scope

### 3.1 In Scope (Phase 1)

| Capability | Description |
|------------|-------------|
| Return initiation | Cashier looks up original sale via invoice number, selects items to return |
| Damaged item return | Only items with condition "damaged" qualify for return |
| **Replacement (preferred)** | Customer receives same product replacement when stock available |
| **Refund (fallback)** | Full refund to original payment method only when no stock available |
| Original payment method | Refund always goes back to original payment method (cash→cash, card→card) |
| Partial return | Customer can return some items from a sale without returning everything |
| Tax reversal | Reverse `tax_amount` from original sale items |
| Inventory restoration | Restock returned items into `product_stock` (if resellable condition) |
| Shift total adjustment | Reverse shift running totals atomically |
| Audit logging | Full audit trail for every return action |
| Manager approval | High-value returns require remote manager approval from another device |
| Configurable settings | Return window and approval threshold configurable via `app_settings` |

### 3.2 Out of Scope (Future Phases)

| Capability | Phase |
|------------|-------|
| Exchange (return + new sale in one flow) | Phase 2 |
| Store credit refund | Phase 2 |
| Receiptless returns (manager override) | Phase 2 |
| Return analytics dashboard | Phase 2 |
| High-frequency returner flagging | Phase 2 |

---

## 4. Terms & Conditions

### 4.1 Return Policy

| Rule | Value | Configurable |
|------|-------|--------------|
| **Return window** | 1 day from purchase | Yes (`return.window_days` in `app_settings`) |
| **Receipt required** | Yes — mandatory | No (enforced by Find Transaction flow) |
| **Restocking fee** | None (0%) | No |
| **Eligible condition** | Damaged items only | No |
| **Preferred outcome** | Replacement (same product, same price) | No |
| **Refund only when** | Last stock — no replacement available | No |
| **Replacement product** | Same product only (same SKU) | No |
| **Replacement price** | Same price (no price difference) | No |
| **Partial return** | Allowed (select specific items) | No |
| **Over-return** | Blocked (cannot exceed original qty) | No |
| **Tax reversal** | Yes (reverse `tax_amount`) | No |

### 4.2 Return Reason Codes

| Code | Label |
|------|-------|
| `damaged` | Barang Rusak |
| `defective` | Barang Cacat |
| `wrong_item` | Barang Salah |
| `other` | Lainnya |

> Note: Only `damaged` and `defective` are eligible for return in Phase 1. Other reasons exist for future use.

### 4.3 Item Condition Codes

| Code | Restock? | Outcome |
|------|----------|---------|
| `resellable` | Yes — add back to sellable stock | Replacement (preferred) |
| `damaged` | No — write off | Replacement if stock available; refund if last stock |
| `defective` | No — quarantine/dispose | Replacement if stock available; refund if last stock |

> Phase 1: Only `damaged` condition items are accepted for return.

### 4.4 Return Window Configuration

The return window is stored in `app_settings`:

| Key | Default | Description |
|-----|---------|-------------|
| `return.window_days` | `1` | Number of calendar days after `sale.created_at` that a return is accepted |
| `return.approval_threshold` | `500000` | Amount in IDR at or above which manager approval is required |

### 4.5 Partial Returns

A sale may contain multiple items (e.g., 10 items). A customer can return **some items** from the sale without returning everything. This is the most common scenario.

#### Return Outcome Decision Tree

```
Customer returns damaged item
         │
         ▼
    ┌─────────────────────────────────────────┐
    │  Check stock of SAME product            │
    │  (product_stock.quantity > 0?)           │
    └─────────────────────────────────────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
 STOCK      NO STOCK
 AVAILABLE  (LAST ITEM)
    │         │
    ▼         ▼
 REPLACE    REFUND
 (default)  (fallback)
```

#### Example Scenario — Replacement (Stock Available)

```
Sale #123:
  Item A: Indomie    ×5 @Rp3,500 = Rp17,500 (tax: Rp1,750)
  Item B: Kopi ABC   ×2 @Rp8,500 = Rp17,000 (tax: Rp1,700)
  Item C: Teh Pucuk  ×3 @Rp4,000 = Rp12,000 (tax: Rp1,200)
  ─────────────────────────────────────────
  Total: Rp46,500 + Rp4,650 tax = Rp51,150
  Payment: Cash Rp30,000 + QRIS Rp21,150

Customer returns: Item A × 1 (damaged)
Stock check: Indomie has 12 in stock → REPLACEMENT
```

**Result:**
- Return damaged Indomie ×1 (stock +1 → 13)
- Give replacement Indomie ×1 (stock -1 → 12)
- Net stock change: 0
- No money changes hands
- Return record: `return_type = 'replacement'`

#### Example Scenario — Refund (Last Stock)

```
Sale #124:
  Item D: Teh Pucuk  ×1 @Rp4,000 = Rp4,000 (tax: Rp400)
  ─────────────────────────────────────────
  Total: Rp4,000 + Rp400 tax = Rp4,400
  Payment: Cash Rp4,400

Customer returns: Teh Pucuk ×1 (damaged)
Stock check: Teh Pucuk has 0 in stock → REFUND
```

**Result:**
- Return damaged Teh Pucuk ×1 (stock +1 → 1)
- No replacement available
- Refund Rp4,400 to Cash (original payment method)
- Return record: `return_type = 'refund'`

#### How Partial Return Works

1. **Cashier selects specific items** to return from the original sale (checkbox + quantity input)
2. **Each return line links to the original sale item** via `sale_return_items.sale_item_id`
3. **System checks stock** for each returned item to determine replacement vs refund
4. **Replacement**: stock restored + new item deducted (net 0 change)
5. **Refund**: stock restored + money returned to original payment method

#### Over-Return Prevention

The system tracks returned quantities per sale item line by querying `SUM(quantity)` from `sale_return_items` grouped by `sale_item_id`. No extra column is added to `sale_items`.

```
Before return:  Item A returned = 0 of 5
After return:   Item A returned = 1 of 5 (4 remaining)

If customer tries to return 5 more Item A:
  → System rejects: "Hanya 4 unit yang tersisa untuk diretur"
```

Validation query:

```sql
SELECT si.quantity - COALESCE(SUM(sri.quantity), 0) AS remaining
FROM sale_items si
LEFT JOIN sale_return_items sri ON sri.sale_item_id = si.id
WHERE si.id = :sale_item_id
GROUP BY si.quantity;
```

#### Partial Return UI Flow

**Replacement (stock available):**
```
Return Modal:
  ┌─────────────────────────────────────────────────┐
  │  Retur Barang — INV-2026-000123                  │
  ├─────────────────────────────────────────────────┤
  │  Barang:                                         │
  │  ☑ Indomie ×5    Qty retur: [1]  @Rp3,500       │
  │  ├─ Kondisi: Rusak                               │
  │  └─ Penggantian: Ya (stok tersedia: 12)          │
  │  ☐ Kopi ABC ×2   Qty retur: [ ]  @Rp8,500       │
  │  ☐ Teh Pucuk ×3  Qty retur: [ ]  @Rp4,000       │
  ├─────────────────────────────────────────────────┤
  │  Retur: Rp0 (penggantian, tanpa uang)            │
  │  Stok: +1 rusak, -1 penggantian                  │
  └─────────────────────────────────────────────────┘
```

**Refund (last stock):**
```
Return Modal:
  ┌─────────────────────────────────────────────────┐
  │  Retur Barang — INV-2026-000124                  │
  ├─────────────────────────────────────────────────┤
  │  Barang:                                         │
  │  ☑ Teh Pucuk ×1   Qty retur: [1]  @Rp4,000      │
  │  ├─ Kondisi: Rusak                               │
  │  └─ Penggantian: Tidak (stok habis)              │
  ├─────────────────────────────────────────────────┤
  │  Pengembalian: Rp4,400                           │
  │  Metode: Cash (original)                         │
  │  Stok: +1 rusak (write off)                      │
  └─────────────────────────────────────────────────┘
```

#### Partial Return Edge Cases

| Case | Behavior |
|------|----------|
| Return 1, then try return 4 more | Only 4 remaining → allow up to 4 |
| Return all 5 of Item A | Fully returned, cannot return Item A again |
| Return all items in sale | Sale fully returned, "Return Items" button disabled |
| Same product sold twice (2 lines) | Each line tracked separately by `sale_item_id` |
| Item with pricing rule discount | Refund uses snapshot price (not current price) |
| Tax reversal | Proportional to returned quantity |
| Consignment-owned item | Restore both `product_stock` and `consignment_stock.available` |
| Stock available → replacement | Return damaged item, give new item, net 0 change |
| Stock unavailable → refund | Return damaged item, refund to original payment method |

---

## 5. Roles & Permissions

### 5.1 Role Matrix

| Role | Process Return? | Approve Return? | View Returns? |
|------|-----------------|-----------------|---------------|
| **Cashier** | ✅ Yes | ❌ No | ✅ Own returns |
| **Manager** | ✅ Yes | ✅ Yes | ✅ All returns (own store) |
| **Admin** | ✅ Yes | ✅ Yes | ✅ All returns |
| **Superadmin** | ✅ Yes | ✅ Yes | ✅ All returns |
| **Staff** | ❌ No | ❌ No | ❌ No |

### 5.2 New Permission Codes

| Code | Name | Description | Granted To |
|------|------|-------------|------------|
| `sale.return` | Retur Penjualan | Process customer return of sold items | cashier, manager, admin, superadmin |
| `sale.return.approve` | Setujui Retur | Approve high-value returns | manager, admin, superadmin |

---

## 6. Approval Workflow

### 6.1 Auto-Approve Path

When `return_amount ≤ return.approval_threshold`:

```
Cashier initiates return
    → System validates (return window, quantities, item eligibility)
    → Return processed immediately
    → Audit logged with cashier as processor
```

### 6.2 Manager Approval Path

When `return_amount > return.approval_threshold`:

```
Cashier initiates return
    → System validates (return window, quantities, item eligibility)
    → System creates pending approval record (expires in 5 minutes)
    → WebSocket notification sent to all managers/admins in same store
    → Manager opens approval modal on their device
    → Manager reviews: original sale details, items, reason, condition, total
    → Manager clicks Approve or Reject
    → If Approve: return processed, audit logged with manager as approver
    → If Reject: return cancelled, audit logged with rejection reason
    → If expired: return cancelled, audit logged as expired
```

### 6.3 Approval Timeout

| Setting | Default | Description |
|---------|---------|-------------|
| `return.approval_timeout_minutes` | `5` | Minutes before pending approval expires |

---

## 7. Workflow Diagram

### 7.1 Happy Path — Replacement (Stock Available)

```
┌─────────────────────────────────────────────────────────────┐
│  1. Customer presents receipt (damaged item)                 │
│  2. Cashier: Find Transaction → enter invoice #              │
│  3. Sale detail drawer shows → click [Return Items]          │
│  4. Return modal: select items, choose reason, condition     │
│  5. System validates: return window OK, qty OK, eligible     │
│  6. System checks stock: same product has stock → REPLACE    │
│  7. Total ≤ threshold → auto-approve                         │
│  8. Unit of Work begins:                                     │
│     a. Restore stock (returned item)                         │
│     b. Deduct stock (replacement item)                       │
│     c. Create sale_return record (type='replacement')        │
│     d. Create sale_return_items records                      │
│     e. Create sale_return_replacements records               │
│     f. Audit log                                             │
│     g. Publish SaleReturned event                            │
│  9. Return receipt printed                                   │
│ 10. Customer receives replacement item (no money exchange)   │
└─────────────────────────────────────────────────────────────┘
```

### 7.2 Refund Path — Last Stock

```
┌─────────────────────────────────────────────────────────────┐
│  1-5. Same as happy path                                     │
│  6. System checks stock: same product has NO stock → REFUND  │
│  7. Total ≤ threshold → auto-approve                         │
│  8. Unit of Work begins:                                     │
│     a. Restore stock (returned item)                         │
│     b. Create sale_return record (type='refund')             │
│     c. Create sale_return_items records                      │
│     d. Create sale_return_payments records (proportional)    │
│     e. Reverse shift totals                                  │
│     f. Audit log                                             │
│     g. Publish SaleReturned event                            │
│  9. Return receipt printed                                   │
│ 10. Customer receives refund to original payment method      │
└─────────────────────────────────────────────────────────────┘
```

### 7.3 Approval Path — Manager Required

```
┌─────────────────────────────────────────────────────────────┐
│  1-6. Same as happy/refund path                              │
│  7. Total > threshold → pending approval created             │
│  8. WebSocket notification → manager's device                │
│  9. Manager reviews and clicks [Approve]                     │
│ 10. Same Unit of Work as happy/refund path                   │
│ 11. Cashier's terminal receives approval confirmation        │
│ 12. Return/replacement processed, receipt printed            │
└─────────────────────────────────────────────────────────────┘
```

---

## 8. Database Schema

### 8.1 New Tables

```sql
-- Return header
CREATE TABLE sale_returns (
    id SERIAL PRIMARY KEY,
    return_number VARCHAR(30) NOT NULL UNIQUE,
    original_sale_id INTEGER NOT NULL REFERENCES sales(id),
    cashier_id INTEGER NOT NULL REFERENCES users(id),
    store_id INTEGER REFERENCES stores(id),
    shift_id INTEGER REFERENCES shifts(id),
    customer_id INTEGER REFERENCES customers(id),
    return_reason VARCHAR(50) NOT NULL,
    return_type VARCHAR(20) NOT NULL DEFAULT 'refund',  -- 'replacement' or 'refund'
    refund_method VARCHAR(20),                           -- NULL for replacement
    total_refund_amount INTEGER NOT NULL DEFAULT 0,      -- 0 for replacement
    restocking_fee INTEGER NOT NULL DEFAULT 0,
    tax_reversal INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'completed',
    approved_by INTEGER REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Return line items
CREATE TABLE sale_return_items (
    id SERIAL PRIMARY KEY,
    sale_return_id INTEGER NOT NULL REFERENCES sale_returns(id) ON DELETE CASCADE,
    sale_item_id INTEGER NOT NULL REFERENCES sale_items(id),
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price INTEGER NOT NULL,
    subtotal INTEGER NOT NULL,
    tax_amount INTEGER NOT NULL DEFAULT 0,
    refund_amount INTEGER NOT NULL DEFAULT 0,            -- 0 for replacement
    condition VARCHAR(20) NOT NULL DEFAULT 'damaged',
    outcome VARCHAR(20) NOT NULL DEFAULT 'refund',       -- 'replacement' or 'refund'
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Replacement tracking (only for replacement outcomes)
CREATE TABLE sale_return_replacements (
    id SERIAL PRIMARY KEY,
    sale_return_id INTEGER NOT NULL REFERENCES sale_returns(id) ON DELETE CASCADE,
    sale_return_item_id INTEGER NOT NULL REFERENCES sale_return_items(id) ON DELETE CASCADE,
    original_product_id INTEGER NOT NULL,
    replacement_product_id INTEGER NOT NULL,             -- same product for Phase 1
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price INTEGER NOT NULL,
    stock_deducted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Refund payments (only for refund outcomes)
CREATE TABLE sale_return_payments (
    id SERIAL PRIMARY KEY,
    sale_return_id INTEGER NOT NULL REFERENCES sale_returns(id) ON DELETE CASCADE,
    payment_method_id INTEGER NOT NULL REFERENCES payment_methods(id),
    payment_method_code VARCHAR(30) NOT NULL,
    amount INTEGER NOT NULL CHECK (amount > 0),
    reference_number VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Manager approval requests
CREATE TABLE return_approvals (
    id SERIAL PRIMARY KEY,
    sale_return_id INTEGER REFERENCES sale_returns(id),
    requested_by INTEGER NOT NULL REFERENCES users(id),
    approved_by INTEGER REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total_amount INTEGER NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    rejection_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

-- Sequences
CREATE SEQUENCE IF NOT EXISTS return_seq START 1;

-- Indexes
CREATE INDEX idx_sale_returns_original_sale ON sale_returns (original_sale_id);
CREATE INDEX idx_sale_returns_cashier ON sale_returns (cashier_id);
CREATE INDEX idx_sale_returns_store ON sale_returns (store_id);
CREATE INDEX idx_sale_returns_created ON sale_returns (created_at);
CREATE INDEX idx_sale_return_items_product ON sale_return_items (product_id);
CREATE INDEX idx_sale_return_items_outcome ON sale_return_items (outcome);
CREATE INDEX idx_sale_return_replacements_return ON sale_return_replacements (sale_return_id);
CREATE INDEX idx_return_approvals_status ON return_approvals (status, expires_at);
```

### 8.2 Permission Migration

```sql
INSERT INTO permissions (code, name, description) VALUES
  ('sale.return', 'Retur Penjualan', 'Memproses retur barang yang dibeli pelanggan'),
  ('sale.return.approve', 'Setujui Retur', 'Menyetujui retur bernilai tinggi')
ON CONFLICT (code) DO NOTHING;

-- Grant sale.return to cashier, manager, admin, superadmin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('cashier', 'manager', 'admin', 'superadmin')
  AND p.code = 'sale.return'
ON CONFLICT DO NOTHING;

-- Grant sale.return.approve to manager, admin, superadmin only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name IN ('manager', 'admin', 'superadmin')
  AND p.code = 'sale.return.approve'
ON CONFLICT DO NOTHING;
```

### 8.3 Settings Migration

```sql
INSERT INTO app_settings (key, value, description) VALUES
  ('return.window_days', '1', 'Jumlah hari setelah pembelian yang diperbolehkan untuk retur'),
  ('return.approval_threshold', '500000', 'Batas jumlah retur yang memerlukan persetujuan manager (Rp)'),
  ('return.approval_timeout_minutes', '5', 'Batas waktu persetujuan retur (menit)')
ON CONFLICT (key) DO NOTHING;
```

---

## 9. API Endpoints

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| `POST` | `/api/sales/:id/return` | `sale.return` | Initiate return for a sale |
| `GET` | `/api/sales/:id/returns` | `sale.view` | List returns for a specific sale |
| `GET` | `/api/sales/returns/:returnId` | `sale.view` | Get return detail |
| `POST` | `/api/sales/returns/:returnId/approve` | `sale.return.approve` | Manager approves pending return |
| `POST` | `/api/sales/returns/:returnId/reject` | `sale.return.approve` | Manager rejects pending return |
| `GET` | `/api/sales/returns/pending` | `sale.return.approve` | List pending approval requests |

---

## 10. Frontend Components

### 10.1 Return Flow Entry Point

- **Find Transaction drawer** → Sale detail → "Return Items" button (visible if `sale.return` permission held)
- Button disabled if: sale is not `completed`, return window expired, or sale already fully returned

### 10.2 Return Processing Modal

```
┌─────────────────────────────────────────────────┐
│  Retur Barang — INV-2026-000123                  │
├─────────────────────────────────────────────────┤
│  Penjualan Asli: 15 Jul 2026, Kasir: Budi       │
│  Pelanggan: Walk-in                              │
├─────────────────────────────────────────────────┤
│  Barang:                                         │
│  ☑ Indomie Goreng  ×2  @Rp3,500   =Rp7,000     │
│  ├─ Kondisi: [Rusak ▼]                           │
│  └─ Penggantian: Ya (stok: 12)                   │
│  ☐ Teh Pucuk 500ml ×1  @Rp4,000   =Rp4,000     │
│  ☑ Kopi ABC  ×1    @Rp8,500   =Rp8,500         │
│  ├─ Kondisi: [Rusak ▼]                           │
│  └─ Penggantian: Tidak (stok habis)              │
├─────────────────────────────────────────────────┤
│  Alasan Retur: [Barang Rusak ▼]                  │
│  Catatan: [___________]                          │
├─────────────────────────────────────────────────┤
│  Penggantian: 1 item (Rp0, tanpa uang)           │
│  Pengembalian: 1 item (Rp10,050)                 │
│  Metode Pengembalian: [Original (Cash) ▼]        │
├─────────────────────────────────────────────────┤
│              [Batal]  [Proses Retur]             │
└─────────────────────────────────────────────────┘
```

### 10.3 Manager Approval Modal (Remote)

```
┌─────────────────────────────────────────────────┐
│  Persetujuan Retur Diperlukan                    │
├─────────────────────────────────────────────────┤
│  Kasir: Budi — INV-2026-000123                   │
│  Jumlah: Rp750.000                               │
│  Alasan: Barang Rusak                             │
├─────────────────────────────────────────────────┤
│  Barang yang Diretur:                             │
│  • Indomie Goreng ×2 — Penggantian (stok: 12)    │
│  • Kopi ABC ×1 — Pengembalian Rp8.500            │
├─────────────────────────────────────────────────┤
│  [Tolak]                          [Setujui]      │
└─────────────────────────────────────────────────┘
```

### 10.4 Frontend Files

```
web/src/modules/salereturn/
├── components/
│   ├── ReturnModal.svelte              -- Main return processing modal
│   ├── ReturnItemRow.svelte            -- Individual item row with qty selector + outcome
│   ├── ReturnReasonSelect.svelte       -- Reason dropdown
│   ├── ReturnSummary.svelte            -- Financial summary (replacement vs refund)
│   ├── ApprovalNotification.svelte     -- Manager approval request toast
│   └── ApprovalModal.svelte            -- Manager approval detail modal
├── services/
│   └── salereturn-service.ts           -- API calls
├── types/
│   └── index.ts                        -- TypeScript types
└── stores/
    └── salereturn-store.svelte.ts      -- State management
```

---

## 11. Backend Architecture

### 11.1 Module Structure

```
internal/salereturn/
├── domain.go        -- SaleReturn, SaleReturnItem, SaleReturnReplacement, SaleReturnPayment, ReturnApproval entities + errors
├── repository.go    -- DB operations (CRUD return, items, replacements, payments, approvals)
├── service.go       -- Business logic (validate, process return/replacement, coordinate stock + shift)
├── handler.go       -- HTTP handlers (gin)
├── ports.go         -- StockRestorer, StockDeducer, SaleLookup interfaces
├── presenter.go     -- JSON response formatting
└── *_test.go        -- Unit + integration tests
```

### 11.2 Port Interfaces

```go
// StockRestorer restores stock for returned items.
// Mirrors StockDeducer — runs inside the return Unit of Work.
type StockRestorer interface {
    RestoreStock(ctx context.Context, tx pgx.Tx, items []shared.StockRestoreItem) error
}

// StockDeducer deducts stock for replacement items.
// Reuses existing inventory module implementation.
type StockDeducer interface {
    DeductStock(ctx context.Context, tx pgx.Tx, items []shared.StockDeductItem) error
}

// SaleLookup fetches original sale data for return validation.
type SaleLookup interface {
    GetSaleByID(ctx context.Context, id int) (*sale.Sale, error)
}

// StockChecker checks available stock for replacement decisions.
type StockChecker interface {
    GetAvailableStock(ctx context.Context, productID int) (int, error)
}
```

### 11.3 Cross-Module Impact

| Module | Change | Type |
|--------|--------|------|
| `internal/sale` | Add `returnable` status check; export `SaleItem` for validation | Read-only |
| `internal/inventory` | New `RestoreStock` method + `sale_return` movement type | **New port** |
| `internal/inventory` | Reuse existing `DeductStock` for replacement items | Existing port |
| `internal/shift` | Reverse shift totals (`UpdateShiftTotals` with negative values) | Existing port |
| `internal/consignment` | Restore `consignment_stock` for consignment-owned items | **New port** |
| `internal/audit` | Log return actions | No change |
| `cmd/server/main.go` | Wire new routes, permissions, ports | Composition root |
| `internal/permissions` | Add `SaleReturn`, `SaleReturnApprove` codes | Constants |

### 11.4 Unit of Work — Replacement Processing

```
BEGIN TX
  ├── Validate (return window, qty, eligibility)
  ├── Lock original sale (SELECT FOR UPDATE)
  ├── Check stock for replacement decision
  ├── IF replacement:
  │   ├── RestoreStock (returned item) — inventory module port
  │   │   └── UPDATE product_stock + INSERT inventory_movements (type='sale_return')
  │   ├── DeductStock (replacement item) — existing inventory module
  │   │   └── UPDATE product_stock + INSERT inventory_movements (type='sale')
  │   ├── RestoreConsignmentStock (if applicable)
  │   ├── INSERT sale_returns (type='replacement', total_refund_amount=0)
  │   ├── INSERT sale_return_items (outcome='replacement')
  │   └── INSERT sale_return_replacements
  ├── IF refund:
  │   ├── RestoreStock (returned item) — inventory module port
  │   ├── RestoreConsignmentStock (if applicable)
  │   ├── INSERT sale_returns (type='refund', total_refund_amount=X)
  │   ├── INSERT sale_return_items (outcome='refund')
  │   ├── INSERT sale_return_payments (proportional split)
  │   └── UpdateShiftTotals (negative contribution)
  ├── INSERT audit_logs (action='return')
  ├── Publish SaleReturned event (outbox)
  └── COMMIT TX
```

---

## 12. Edge Cases

### 12.1 Critical Edge Cases (Must Handle)

| # | Case | Behavior |
|---|------|----------|
| 1 | **Return after shift closed** | Skip shift total adjustment; log adjustment in audit trail. Cash drawer drift is accepted for post-close returns. |
| 2 | **Concurrent return + sale** | StockRestorer uses `SELECT FOR UPDATE` before incrementing stock row (same pattern as StockDeducer). |
| 3 | **Consignment-owned item return** | Restore both `product_stock` AND `consignment_stock.available`. Look up `consignment_sale_items` to credit correct supplier. Reject if settlement already paid. |
| 4 | **Discontinued/archived product** | Allow return; skip stock restoration if product is archived/discontinued. Use snapshotted product data from `sale_items`. |
| 5 | **Split payment refund** | Refund proportional to original allocation per payment method. Track each refund line separately. |

### 12.2 Important Edge Cases (Should Handle)

| # | Case | Behavior |
|---|------|----------|
| 6 | **Cross-store return** | Restrict to same-store only. Customer must return at original purchase store. |
| 7 | **Registered customer total_spent** | Decrement `customers.total_spent` and `loyalty_points` by return amount. |
| 8 | **Pricing rule discount** | Use snapshotted `unit_price` from `sale_items` (immutable), never re-resolve from pricing engine. |
| 9 | **Tax class deleted** | Use snapshotted `tax_rate` from `sale_items`, not current `tax_classes` row. |
| 10 | **Replacement stock race condition** | Use `SELECT FOR UPDATE` on stock row before deciding replacement vs refund. |

### 12.3 Validation Edge Cases

| # | Case | Behavior |
|---|------|----------|
| 11 | Return window expired | Reject with error: "Batas waktu retur sudah berlalu" |
| 12 | Return qty > original qty | Reject: "Jumlah retur melebihi jumlah pembelian" |
| 13 | Already partially returned | Query `sale_return_items` for sum of returned qty; allow remaining |
| 14 | Sale not completed | Reject: "Hanya penjualan selesai yang dapat diretur" |
| 15 | Sale from another store | Reject (store scoping enforced) |
| 16 | Manager approval expired | Auto-cancel, audit log as expired |
| 17 | Shift not open | Reject: "Shift sudah ditutup, retur tidak dapat diproses" |

### 12.4 Replacement/Refund Edge Cases

| # | Case | Behavior |
|---|------|----------|
| 18 | Stock available → replacement | Return damaged item, give new item, net 0 stock change |
| 19 | Stock unavailable → refund | Return damaged item, refund to original payment method |
| 20 | Replacement same product only | Phase 1: only same SKU replacement allowed |
| 21 | Replacement same price only | No price difference handling in Phase 1 |
| 22 | Multiple returns on same item | Track cumulative returned qty; allow until fully returned |
| 23 | Walk-in customer (id=1) | No customer field updates needed; no loyalty points to reverse |
| 24 | QRIS payment refund | No reference number; store original `sale_payment.id` on refund record |
| 25 | Gift cards / store credit | Not yet implemented; design API to support future `refund_method='store_credit'` |

### 12.5 Reporting Edge Cases

| # | Case | Behavior |
|---|------|----------|
| 26 | Materialized views | Keep original sale `status='completed'`; returns tracked in separate `sale_returns` table |
| 27 | Product sold twice (2 lines) | Each `sale_return_item` links to specific `sale_item_id` |
| 28 | Rounding on partial returns | Compute line-by-line amounts (integer arithmetic), not from sale-level totals |

---

## 13. Audit Trail

Every return action is logged with:

| Field | Value |
|-------|-------|
| `action` | `return` / `return_approve` / `return_reject` / `return_expire` |
| `entity_type` | `sale_return` |
| `entity_id` | Return ID |
| `new_values` | Return details (items, amounts, reason) |
| `user_id` | Cashier (for return) or Manager (for approve/reject) |
| `description` | "Returned {N} items from INV-XXXX — RpX,XXX" |

---

## 14. Reporting Impact

- **Sales reports**: Returns should be deducted from gross sales to show net sales
  - Replacement: no financial impact (stock only)
  - Refund: financial impact (reverse payment)
- **Dashboard**: Return count, replacement count, and refund total visible in dashboard stats
- **Inventory**: `sale_return` and `sale` (replacement) movements appear in product history
- **Shift summary**: Refund amounts deducted from shift totals (replacements don't affect shift totals)

Implementation via `SaleReturned` domain event consumed by reporting module (eventual consistency per ADR_Cross_Module_Transaction_Strategy).

---

## 15. Test Plan

### 15.1 Unit Tests

| Test | Scope |
|------|-------|
| Return within window — success | Service layer |
| Return outside window — rejected | Service layer |
| Partial return — success | Service layer |
| Over-return — rejected | Service layer |
| **Replacement — stock available** | Service + inventory |
| **Refund — last stock** | Service + inventory |
| Stock restoration — resellable item | Service + inventory |
| Stock restoration — damaged item (no restock) | Service + inventory |
| **Stock deduction for replacement** | Service + inventory |
| Shift total reversal (refund only) | Service + shift |
| Consignment stock restoration | Service + consignment |
| Manager approval — below threshold (auto-approve) | Service layer |
| Manager approval — above threshold (pending) | Service layer |
| Manager approve — success | Service layer |
| Manager reject — success | Service layer |
| Approval timeout — auto-expire | Service layer |
| Audit log created for every action | Repository |

### 15.2 Integration Tests

| Test | Scope |
|------|-------|
| POST /api/sales/:id/return — full flow (replacement) | Handler |
| POST /api/sales/:id/return — full flow (refund) | Handler |
| GET /api/sales/:id/returns — list returns | Handler |
| POST approve/reject endpoints | Handler |
| Permission gating (cashier cannot approve) | Handler + middleware |
| Store scoping (cross-store blocked) | Handler |

### 15.3 E2E Tests

| Test | Scope |
|------|-------|
| Cashier processes replacement end-to-end | Playwright |
| Cashier processes refund end-to-end | Playwright |
| Manager approval flow end-to-end | Playwright |
| Return window validation | Playwright |
| Receipt reprint after return | Playwright |

---

## 16. Migration from Current Roadmap

Update `docs/roadmap/upcoming-features.md`:

**Before:**
> Returns & Refund: Tidak akan diimplementasikan. Kebijakan toko: barang yang sudah dibeli tidak dapat direfund atau dikembalikan.

**After:**
> Returns & Replacement (Damaged Items): Phase 1 implemented. Customer can return damaged items within configurable window (default 1 day). **Replacement preferred** when stock available (same product, same price). **Refund fallback** only when no stock available. Manager approval required for high-value returns. See `docs/prd/PRD-Sales-Return.md`.
