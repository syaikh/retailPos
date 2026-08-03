# Retail POS System

Sistem Point of Sale (POS) modern untuk toko retail dengan manajemen inventory, penjualan, purchase order, pricing engine, reporting, shift management, dan kontrol akses berbasis role.

## Features

- **Point of Sale (POS)** — Transaksi penjualan dengan scanner, diskon, split payment (multi metode pembayaran), dan hold & recall (parked sales)
- **Purchase Order & Goods Receiving** — Alur pembelian dari supplier: draft → confirmed → received, penerimaan barang parsial, auto-generate nomor PO/GR/DO
- **Stock Opname** — Sesi perhitungan stok fisik (draft → counting → pending approval → approved/cancelled) dengan snapshot inventori, multi-counter assignment, blind count, recount workflow, dan auto-adjustment stok saat approval (FR-001 s.d. FR-044)
- **Shift Management** — Buka/tutup shift kasir, opening/closing balance, discrepancy review & audit
- **Pricing Engine** — Aturan harga (special price / promotion) berbasis produk, kategori, brand, customer group, dan store; workflow approval (draft → pending → approved/rejected); resolver harga real-time
- **Supplier Management** — CRUD supplier, tautan produk-supplier, preferred supplier, bulk actions
- **Customer & Customer Groups** — Manajemen pelanggan, grup pelanggan (Walk-in, Member, VIP), bulk actions
- **Multi-Warehouse & Multi-Store** — Inventori per warehouse/store dengan kunci unik komposit, manajemen toko
- **Inventory Management** — Tracking stok, movement, low stock alerts, stock thresholds, multi-category filter
- **Import & Export Framework** — Schema-driven reusable import/export untuk Products, Categories, Brands, UOMs, Customers, Pricing Rules, Suppliers dengan XLSX templates, preview, validasi, reference dropdowns, import history (async job), dan cancel
- **User Management** — RBAC (Role-Based Access Control) dengan permissions dot-notation, hierarki manajer-bawahan (org chart), soft delete
- **Audit Logging** — Full audit trail untuk semua aksi (termasuk login/logout, import, change-password)
- **Real-time Dashboard** — Statistik penjualan, revenue, analytics + live updates via WebSocket, chart harian/mingguan/bulanan, period comparison, pricing breakdown
- **WebSocket Support** — Notifikasi real-time (dashboard live, update PO)
- **Swagger/OpenAPI** — API documentation via swaggo annotations
- **Structured Logging** — JSON (production) / text (development) via `log/slog`
- **EventBus Observability** — Atomic metrics for published/consumed/failed events
- **Dead-Letter Queue** — Failed events stored to PostgreSQL for retry
- **Materialized Views** — Pre-aggregasi data penjualan harian/jam-an untuk query reporting cepat, refresh otomatis per jam

### Security Features

- JWT authentication dengan refresh token (HTTP-only cookie, terpisah secret refresh)
- CSRF protection pada state-changing endpoints (validate, logout, change-password)
- Rate limiting dengan per-entry TTL (terpisah untuk login, refresh, dan API umum)
- IP spoofing protection (menggunakan `RemoteAddr` bukan `X-Forwarded-For`)
- Product search via tsvector (menghindari ILIKE full table scan)
- Inventory adjustments menggunakan `SELECT ... FOR UPDATE` untuk concurrency safety
- Security headers middleware (CSP, X-Frame-Options, dll), body limit, gzip

---

## Architecture

### Development

```
┌──────────────────────────────────────────────────────────┐
│  Frontend (Vite dev server)   http://localhost:5173      │
│  Svelte 5 + Tailwind CSS 4 + Vite 6 (HMR)               │
└──────────────┬───────────────────────────────────────────┘
               │ /api/* → Backend, /ws/* → WebSocket
┌──────────────┴───────────────────────────────────────────┐
│  Go Backend (Gin)        http://localhost:9095           │
│  PostgreSQL              localhost:5433 (postgres-dev)   │
│  `./run-dev.sh` rebuild + restart otomatis (tekan r)     │
└──────────────────────────────────────────────────────────┘
```

### Production (Podman / Docker Compose)

```
┌──────────────────────────────────────────────────────────┐
│  Nginx Frontend            Port 80 / 443                │
│  Go Backend                Port 8080 (internal)         │
│  PostgreSQL 18             Volume retail-pos-postgres-data│
│  Jaringan: retail-pos-network                            │
│  `./deploy/podman-deploy.sh start`                       │
└──────────────────────────────────────────────────────────┘
```

**Tech Stack:**
- **Backend:** Go (Gin), PostgreSQL 18 (pgx), JWT Auth, WebSocket (gorilla/websocket), structured logging (slog)
- **Frontend:** Svelte 5, Tailwind CSS 4, Vite 6, Chart.js, jsPDF, Playwright, Vitest
- **Infrastructure:** Podman (rootless), Docker Compose, Nginx, systemd

---

## Quick Start

### Prerequisites

```bash
# Backend
go version  # 1.22+

# Frontend
cd web && npm install

# Database (dev)
podman run -d --name postgres-dev -p 5433:5432 \
  -e POSTGRES_USER=pos -e POSTGRES_PASSWORD=admin123 -e POSTGRES_DB=retail_pos \
  postgres:18-alpine
```

### Development

```bash
# 1. Salin konfigurasi
cp .env.example .env

# 2. Seed database (dummy data)
./seed-dev.sh            # data besar (produk, transaksi)
./seed-daily-dev.sh      # transaksi harian saja

# 3. Start backend (port 9095, auto-reload via tombol r/q)
./run-dev.sh

# 4. Start frontend (port 5173)
cd web && npm run dev

# 5. Buka http://localhost:5173
```

### Production

```bash
make build-all                 # Build image backend + frontend
./deploy/podman-deploy.sh start   # Start semua service
./deploy/podman-deploy.sh migrate # Jalankan migrasi (opsional, otomatis saat startup)
./deploy/podman-deploy.sh seed    # Seed data awal (opsional)
```

---

## Backend

### Module Structure

```
internal/
├── audit/             # Audit logging (domain events + listener)
├── brand/             # Brand CRUD + import adapter
├── category/          # Category CRUD + import adapter
├── config/            # App configuration (env, timezone)
├── customergroup/     # Customer group CRUD + bulk actions
├── customer/          # Customer CRUD + bulk actions + import adapter
├── eventbus/          # In-process event bus (retry, dead-letter, metrics)
├── inventory/         # Stock tracking, adjustments, low stock
├── middleware/        # Auth (JWT), CORS, rate limit, CSRF, security headers
├── platform/
│   └── importexport/  # Schema-driven import/export framework
├── pricing/           # Pricing rules engine + resolver + approval workflow
├── product/           # Product CRUD (repository + query + bulk)
├── purchase/          # Purchase orders + goods receipts
├── report/            # Dashboard stats, charts, comparisons
├── sale/              # POS transaction, split payment, parked sales, export
├── shared/            # Shared types, logger, response helpers
├── shift/             # Cashier shift management
├── store/             # Store CRUD
├── supplier/          # Supplier CRUD + product-supplier links
├── uom/               # Unit of Measure CRUD + import adapter
├── user/              # User & role management + auth (login/refresh)
├── wiring/            # Dependency injection / wiring
└── pkg/
    └── websocket/     # WebSocket hub
```

### Key Files

| File | Description |
|------|-------------|
| `cmd/server/main.go` | HTTP + WebSocket server entry point (routing, middleware, graceful shutdown) |
| `cmd/server/e2e_test.go` | End-to-end API tests |
| `internal/wiring/wiring.go` | Dependency wiring |
| `internal/eventbus/bus.go` | Event bus with retry, dead-letter, observability |
| `internal/pricing/resolver.go` | Harga final resolver (rule → harga efektif) |
| `internal/purchase/service.go` | Purchase order & goods receipt logic |
| `internal/shift/service.go` | Shift lifecycle (open/close/review/audit) |
| `internal/sale/service.go` | POS transaction, parked sales, split payment |
| `internal/shared/logger.go` | Structured logging (slog) |
| `database/migrations/000_squash.sql` | Baseline schema (role, user, product, sale, inventory, dll) |
| `docs/swagger.go` | OpenAPI annotations |

### Run Tests

```bash
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos TEST_DB_PASSWORD=admin123 \
DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only \
go test -p 1 -count=1 ./...
```

> `-p 1` memaksa eksekusi sekuensial untuk menghindari deadlock dari TRUNCATE/INSERT bersamaan antar package. Tests terhubung ke database `retail_pos_test`.

### API Documentation

Swagger annotations ada di endpoint kunci. Untuk generate spec:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/server/main.go -o docs/swagger
```

Spec bisa diakses di `/swagger/*any` saat server berjalan.

### API Endpoints

Base path: `/api`. Semua endpoint require JWT (via `Authorization: Bearer` atau cookie) kecuali dinyatakan "Public".

#### Auth

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/login` | Login (set cookie refresh_token) | No |
| POST | `/refresh` | Refresh access token | No |
| POST | `/validate` | Validasi session + daftar permission | Yes |
| POST | `/logout` | Logout (revoke refresh token) | Yes |
| POST | `/change-password` | Ganti password sendiri | Yes |

#### Dashboard & Reports

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/dashboard/stats` | Statistik dashboard (revenue hari ini, dsb) | `dashboard.view` |
| GET | `/dashboard/live` | Statistik live (WebSocket) | `dashboard.view` |
| GET | `/dashboard/chart` | Data chart penjualan | `report.view` |
| GET | `/dashboard/chart/weekly` | Laporan mingguan | `report.view` |
| GET | `/dashboard/chart/monthly` | Laporan bulanan | `report.view` |
| GET | `/dashboard/comparison` | Perbandingan periode | `report.view` |
| POST | `/dashboard/export` | Export dashboard (CSV/XLSX) | `report.view` |
| GET | `/dashboard/years` | Tahun tersedia | `report.view` |
| GET | `/dashboard/pricing-breakdown` | Rincian harga | `report.view` |

#### Products, Categories, Brands, UOM

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/products` | List produk (search + filter multi-category) | Public |
| GET | `/products/next-sku` | Next SKU generator | Public |
| GET | `/products/:id` | Detail produk | Yes |
| POST | `/products` | Buat produk | `product.create` |
| PUT | `/products/:id` | Update produk | `product.update` |
| DELETE | `/products/:id` | Hapus produk | `product.delete` |
| POST | `/products/bulk/status` | Bulk update status | `product.update` |
| GET | `/categories` | List kategori | Public |
| GET | `/categories/manage` | List kategori (management, paginated) | `category.view` |
| POST | `/categories` | Buat kategori | `category.create` |
| PUT | `/categories/:id` | Update kategori | `category.update` |
| DELETE | `/categories/:id` | Hapus kategori | `category.delete` |
| GET | `/brands` | List brand | Public |
| POST | `/brands` | Buat brand | `product.create` |
| PUT | `/brands/:id` | Update brand | `product.update` |
| DELETE | `/brands/:id` | Hapus brand | `product.delete` |
| GET | `/units-of-measure` | List UOM | Public |
| POST | `/units-of-measure` | Buat UOM | `product.create` |
| PUT | `/units-of-measure/:id` | Update UOM | `product.update` |
| DELETE | `/units-of-measure/:id` | Hapus UOM | `product.delete` |
| GET | `/tax-classes` | List tax classes | Public |
| GET | `/stock-thresholds` | Stock thresholds | Public |

#### Sales

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| POST | `/sales` | Buat transaksi (split payment, parked recall) | `sale.create` |
| GET | `/sales` | Riwayat penjualan | `sale.view` |
| GET | `/sales/:id` | Detail penjualan | `sale.view` |
| GET | `/sales/export` | Export penjualan (CSV/XLSX) | `report.view` |
| POST | `/sales/parked` | Park (hold) transaksi | `sale.park` |
| GET | `/sales/parked` | List parked sales | `sale.park` |
| GET | `/sales/parked/:id` | Detail parked sale | `sale.park` |
| POST | `/sales/parked/:id/recall` | Recall parked sale | `sale.park` |
| DELETE | `/sales/parked/:id` | Batalkan parked sale | `sale.park` |
| GET | `/payment-methods` | List metode pembayaran | Public |
| GET | `/payment-methods/:code` | Detail metode pembayaran | Yes |

#### Inventory

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| POST | `/inventory/adjust` | Penyesuaian stok manual | `inventory.adjust` |

#### Stock Opname

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| POST | `/stock-opnames` | Buat sesi Stock Opname (multi-scope snapshot) | `stock_opname.create` |
| GET | `/stock-opnames` | List sesi (filter status/scope, pagination) | `stock_opname.view` |
| GET | `/stock-opnames/:id` | Detail sesi | `stock_opname.view` |
| GET | `/stock-opnames/assignable-users` | Daftar user yang bisa di-assign | `stock_opname.assign` |
| POST | `/stock-opnames/:id/open` | Buka sesi (Draft → Open) | `stock_opname.create` |
| POST | `/stock-opnames/:id/cancel` | Batalkan sesi (draft/open/counting/needs_recount) | `stock_opname.cancel` |
| POST | `/stock-opnames/:id/assignments` | Assign counter/supervisor | `stock_opname.assign` |
| GET | `/stock-opnames/:id/assignments` | List assignment sesi | `stock_opname.view` |
| PUT | `/stock-opnames/:id/assignments/:assignmentId` | Reassign counter | `stock_opname.assign` |
| PUT | `/stock-opnames/items/:itemId/count` | Simpan hasil counting (autosave) | `stock_opname.count` |
| GET | `/stock-opnames/items/:itemId/counts` | Riwayat counting per item | `stock_opname.view` |
| POST | `/stock-opnames/:id/start` | Mulai counting (Draft/Open → Counting) | `stock_opname.count` |
| POST | `/stock-opnames/:id/submit` | Submit hasil counting (Counting → Verification) | `stock_opname.submit` |
| POST | `/stock-opnames/:id/verify` | Verifikasi (persist selisih, belum ubah stok; Verification → Approved) | `stock_opname.verify` |
| POST | `/stock-opnames/:id/reject` | Reject sesi (Verification → Needs Recount) | `stock_opname.verify` |
| POST | `/stock-opnames/:id/recount` | Request recount (Verification → Needs Recount) | `stock_opname.recount` |
| POST | `/stock-opnames/:id/resume` | Resume counting (Needs Recount → Counting) | `stock_opname.count` |
| POST | `/stock-opnames/:id/post-adjustment` | Posting penyesuaian ke stok + buat dokumen IA- (Approved → Posted) | `stock_opname.post` |
| POST | `/stock-opnames/:id/close` | Tutup sesi (Posted → Closed) | `stock_opname.close` |
| GET | `/stock-opnames/:id/summary` | Ringkasan progres sesi | `stock_opname.view` |
| GET | `/stock-opnames/:id/difference` | Laporan selisih stok | `stock_opname.view` |
| GET | `/stock-opnames/:id/export` | Export laporan (CSV/Excel/PDF) | `stock_opname.export` |
| GET | `/stock-opnames/adjustments` | Laporan penyesuaian (dokumen IA-) | `stock_opname.report` |
| GET | `/stock-opnames/adjustments/:id` | Detail dokumen penyesuaian | `stock_opname.report` |

#### Customers & Customer Groups

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/customers` | List pelanggan | `customer.view` |
| GET | `/customers/:id` | Detail pelanggan | `customer.view` |
| POST | `/customers` | Buat pelanggan | `customer.create` |
| PUT | `/customers/:id` | Update pelanggan | `customer.update` |
| DELETE | `/customers/:id` | Hapus pelanggan | `customer.delete` |
| POST | `/customers/bulk/status` | Bulk update status | `customer.update` |
| POST | `/customers/bulk/delete` | Bulk delete | `customer.delete` |
| GET | `/customer-groups` | List grup pelanggan | `customer_group.view` |
| GET | `/customer-groups/:id` | Detail grup | `customer_group.view` |
| POST | `/customer-groups` | Buat grup | `customer_group.create` |
| PUT | `/customer-groups/:id` | Update grup | `customer_group.update` |
| DELETE | `/customer-groups/:id` | Hapus grup | `customer_group.delete` |
| PUT | `/customer-groups/bulk` | Bulk update | `customer_group.update` |
| DELETE | `/customer-groups/bulk` | Bulk delete | `customer_group.delete` |

#### Stores

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/stores` | List toko | `store.view` |
| GET | `/stores/active` | List toko aktif | `store.view` |
| GET | `/stores/:id` | Detail toko | `store.view` |
| POST | `/stores` | Buat toko | `store.create` |
| PUT | `/stores/:id` | Update toko | `store.update` |
| DELETE | `/stores/:id` | Hapus toko | `store.delete` |

#### Purchase Orders & Goods Receiving

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| POST | `/purchase-orders` | Buat draft PO | `purchase_order.create` |
| GET | `/purchase-orders` | List PO (filter status/supplier) | `purchase_order.view` |
| GET | `/purchase-orders/:id` | Detail PO | `purchase_order.view` |
| PUT | `/purchase-orders/:id` | Update draft PO | `purchase_order.update` |
| DELETE | `/purchase-orders/:id` | Hapus draft PO | `purchase_order.delete` |
| POST | `/purchase-orders/:id/confirm` | Konfirmasi PO | `purchase_order.confirm` |
| POST | `/purchase-orders/:id/cancel` | Batalkan PO | `purchase_order.cancel` |
| GET | `/purchase-orders/:id/receipts` | List goods receipts PO | `purchase_order.view` |
| POST | `/goods-receipts` | Terima barang (auto-generate GR & DO number) | `purchase_order.receive` |

#### Pricing Engine & Suppliers

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/pricing-rules` | List aturan harga | `pricing.view` |
| GET | `/pricing-rules/:id` | Detail aturan | `pricing.view` |
| POST | `/pricing-rules` | Buat aturan | `pricing.create` |
| PUT | `/pricing-rules/:id` | Update aturan | `pricing.update` |
| DELETE | `/pricing-rules/:id` | Hapus aturan | `pricing.delete` |
| POST | `/pricing-rules/check-conflicts` | Cek konflik aturan | `pricing.view` |
| POST | `/pricing-rules/:id/submit` | Submit untuk approval | `pricing.update` |
| POST | `/pricing-rules/:id/approve` | Approve aturan | `pricing.update` |
| POST | `/pricing-rules/:id/reject` | Reject aturan | `pricing.update` |
| POST | `/pricing/resolve` | Resolve harga final | `pricing.view` |
| GET | `/products/search` | Search produk (untuk pricing) | `pricing.view` |
| GET | `/suppliers` | List supplier | `pricing.view` |
| GET | `/suppliers/:id` | Detail supplier | `pricing.view` |
| POST | `/suppliers` | Buat supplier | `pricing.create` |
| PUT | `/suppliers/:id` | Update supplier | `pricing.update` |
| DELETE | `/suppliers/:id` | Hapus supplier | `pricing.delete` |
| PUT | `/suppliers/bulk` | Bulk update | `pricing.update` |
| DELETE | `/suppliers/bulk` | Bulk delete | `pricing.delete` |
| GET | `/suppliers/:id/products` | Produk dari supplier | `pricing.view` |
| POST | `/suppliers/:id/products` | Tautkan produk ke supplier | `pricing.update` |
| DELETE | `/suppliers/:id/products/:productId` | Lepas tautan produk | `pricing.update` |
| PUT | `/suppliers/:id/products/:productId` | Update relasi (unit_cost) | `pricing.update` |
| POST | `/suppliers/:id/products/:productId/preferred` | Set preferred supplier | `pricing.update` |
| GET | `/products/:id/suppliers` | Supplier dari produk | `pricing.view` |

#### Shifts

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| POST | `/shifts/open` | Buka shift | `shift.create` |
| POST | `/shifts/:id/close` | Tutup shift | `shift.create` |
| POST | `/shifts/close-all` | Tutup semua shift aktif | `shift.create` |
| POST | `/shifts/:id/review` | Review selisih shift | `shift.review` |
| POST | `/shifts/:id/audit` | Audit fisik cash | `shift.audit` |
| GET | `/shifts/active` | Shift aktif saat ini | Yes |
| GET | `/shifts` | List shift | `shift.view` |
| GET | `/shifts/export` | Export shift | `shift.view` |
| GET | `/shifts/:id` | Detail shift | `shift.view` |

#### User & Role Management

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/admin/users` | List users | `user.view` |
| POST | `/admin/users` | Buat user | `user.create` |
| PUT | `/admin/users/:id` | Update user | `user.update` |
| DELETE | `/admin/users/:id` | Hapus user (soft delete) | `user.delete` |
| GET | `/admin/users/:id/subordinates` | Bawahan user | `user.view` |
| GET | `/admin/users/:id/manager` | Manager user | `user.view` |
| GET | `/admin/users/org-chart` | Org chart | `user.view` |
| GET | `/admin/roles` | List roles | `role.view` |
| POST | `/admin/roles` | Buat role | `role.create` |
| PUT | `/admin/roles/:id` | Update role | `role.update` |
| PUT | `/admin/roles/:id/permissions` | Update permission role | `role.update` |
| DELETE | `/admin/roles/:id` | Hapus role | `role.delete` |
| GET | `/admin/permissions` | List semua permissions | `role.view` |

#### Audit Logs

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/audit-logs` | List audit logs (filter tanggal, aksi, entity) | `audit.view` |
| GET | `/audit-logs/:id` | Detail audit log | `audit.view` |
| GET | `/audit-logs/export` | Export audit logs | `audit.view` |
| GET | `/audit-logs/entity-types` | List entity types | `audit.view` |

#### Import & Export

Module yang didukung: `products`, `categories`, `brands`, `uoms`, `customers`, `pricing_rules`, `suppliers`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/import-export/modules` | List module importable |
| GET | `/import-export/template/:module` | Download XLSX template |
| POST | `/import-export/preview/:module` | Preview import (validasi) |
| POST | `/import-export/confirm/:module` | Confirm import (async job) |
| GET | `/import-export/progress/:jobId` | Progress job import |
| POST | `/import-export/cancel/:jobId` | Batalkan job |
| GET | `/import-export/history/:module` | Riwayat import per module |
| GET | `/import-export/history/:module/:jobId` | Detail snapshot job |
| GET | `/import-export/history/:module/:jobId/rows` | Rows hasil import |
| GET | `/import-export/export/:module` | Export data (CSV/XLSX) |

#### System

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/health` | Health check | No |
| GET | `/ws` | WebSocket hub | Yes |
| GET | `/swagger/*any` | Swagger UI | No |

---

## Frontend

### Development

```bash
cd web
npm run dev       # Start dev server (port 5173)
npm run build     # Build for production
npm run test:run  # Unit test (Vitest)
npx playwright test  # Run E2E tests
```

### Module Structure

```
web/src/
├── app/               # App shell (main.svelte, router, providers, config permissions)
├── modules/           # Feature modules
│   ├── admin/         # Users, roles, audit logs
│   ├── auth/          # Login, session
│   ├── customer-groups/ # Customer groups management
│   ├── customers/     # Customer management
│   ├── dashboard/     # Charts, stats, live updates
│   ├── import-export/ # Import wizard, history
│   ├── inventory/     # Stock management
│   ├── pos/           # Point of Sale (split payment, parked sales)
│   ├── pricing/       # Pricing rules + approval
│   ├── product/       # Product catalog
│   ├── purchase-orders/ # PO + goods receiving
│   ├── reporting/     # Reports with chart config + export
│   ├── sales/         # Sales history
│   ├── settings/      # Settings
│   ├── shifts/        # Shift management
│   └── supplier/      # Supplier management
├── shared/            # API client (axios), websocket, services, stores, types, utils (Jakarta time, permissions)
│   └── ui/            # Shared UI components (Modal, DataTable, Pagination, dll)
├── app.css            # Global styles & Tailwind imports
└── main.js            # Entry point
```

### Jakarta Timezone

Backend menyimpan data dalam UTC, namun **semua query menggunakan timezone Asia/Jakarta**. Frontend menghitung tanggal Jakarta dalam UTC sebelum mengirim ke API:

- Jakarta midnight = UTC 07:00 (offset 7 jam)
- Util: `getTodayInJakarta()`, `getDateNDaysAgoInJakarta()` di `web/src/shared/utils/jakartaTime.ts`
- Backend mem-parse date dengan `time.ParseInLocation("2006-01-02", dateStr, jakartaLoc)`

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | (required) | 256-bit secret untuk JWT signing. Generate: `openssl rand -hex 32` |
| `JWT_SECRET_REFRESH` | `JWT_SECRET` | Secret terpisah untuk refresh token (direkomendasikan berbeda di produksi) |
| `DATABASE_URL` | (empty) | URL lengkap PostgreSQL; jika kosong dibangun dari `DB_*` |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5433` (dev) | PostgreSQL port |
| `DB_USER` | `pos` | Database username |
| `DB_PASSWORD` | `admin123` | Database password |
| `DB_NAME` | `retail_pos` | Database name |
| `ENV` | `development` | `development` (log text) / `production` (log JSON, mode release, sslmode require) |
| `LOG_LEVEL` | `debug`/`info` | Level log: debug, info, warn, error |
| `CORS_ORIGIN` | `http://localhost:5173` | Allowed CORS origin (tidak boleh `*` di production) |
| `PORT` | `9095` | Port HTTP server |
| `COOKIE_DOMAIN` | (empty) | Domain cookie refresh token |
| `COOKIE_SECURE` | `false` | Set `true` untuk HTTPS |
| `LOGIN_RATE_LIMIT_RPM` | `5` | Rate limit login (per menit) |
| `LOGIN_RATE_LIMIT_BURST` | `5` | Burst login |
| `RATE_LIMIT_RPS` | `50` | Rate limit API umum (per detik) |
| `RATE_LIMIT_BURST` | `100` | Burst API umum |
| `REFRESH_RATE_LIMIT_RPM` | `10` | Rate limit refresh (per menit) |
| `REFRESH_RATE_LIMIT_BURST` | `10` | Burst refresh |
| `STOCK_WARNING_THRESHOLD` | `10` | Stok di bawah ini = "perlu perhatian" |
| `STOCK_CRITICAL_THRESHOLD` | `5` | Stok di bawah ini = "low stock" |
| `STOCK_MINIMUM` | `10` | Stok default produk baru |

Copy `.env.example` ke `.env` untuk development lokal.

---

## Deployment

### Podman / Docker Compose (Recommended)

```bash
make build-all                       # Build image backend + frontend
./deploy/podman-deploy.sh start      # Start semua service
./deploy/podman-deploy.sh status     # Check status
./deploy/podman-deploy.sh logs       # View logs
./deploy/podman-deploy.sh migrate    # Jalankan migrasi
./deploy/podman-deploy.sh seed       # Seed data
./deploy/podman-deploy.sh stop       # Stop semua service
./deploy/podman-deploy.sh restart    # Restart
```

Atau gunakan Makefile: `make deploy`, `make stop`, `make restart`, `make status`, `make logs`, `make db-backup`, `make db-restore`, `make db-shell`.

### Systemd

```bash
sudo cp deploy/retail-pos.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now retail-pos
```

### Database Migrations

Migrations berjalan otomatis saat startup (tracking via tabel `schema_migrations`). Schema saat ini: **10 migrations (000–009)**. Lihat `database/migrations/` untuk detail. Untuk environment baru, jalankan migrasi secara berurutan dimulai dari `000_squash.sql`.

> **Penting:** Terapkan migrasi **sebelum** deploy binary server baru. Beberapa migrasi punya constraint urutan (mis. `006_consolidate_permissions.sql` dan `009_add_do_sequence.sql`) — lihat AGENTS.md.

Migrations terkini:
- `000_squash.sql` — Baseline schema + seed data awal (roles, permissions, users, payment methods, customer groups)
- `001_materialized_views.sql` — Materialized views untuk reporting (`refresh_sales_mv()`)
- `002_multi_warehouse.sql` — Multi-warehouse (unique constraint komposit `product_stock`)
- `003_shift_perf.sql` — Optimasi performa shift (composite index)
- `004_split_payment.sql` — Tabel `sale_payments` untuk split payment
- `005_reports_to.sql` — Hierarki manajer-bawahan di users
- `006_consolidate_permissions.sql` — Konsolidasi permission ke dot-notation (hapus `.read` dan `:read`)
- `007_purchase_orders.sql` — Tabel PO, goods receipts, permission purchase_order.*
- `008_add_cancel_permission.sql` — Permission `purchase_order.cancel`
- `009_add_do_sequence.sql` — Sequence `do_seq` untuk nomor DO

---

## Default Credentials

| Role | Username | Password | Deskripsi |
|------|----------|----------|-----------|
| Superadmin | `superadmin` | `admin123` | Semua permission |
| Admin | `admin` | `admin123` | User management, reports, PO, pricing (tanpa audit.view / role.update / user.delete) |
| Manager | `manager` | `admin123` | Inventory, sales, PO, pricing, shifts |
| Cashier | `cashier` | `admin123` | POS only (create/view sales, print, park, shift) |
| Staff | `staff` | `admin123` | Inventory + Dashboard |

Ganti password di production via `database/seeds/004_users.sql` atau UI change-password.

---

## Permission Matrix

Permission memakai **dot-notation** (`entity.action`), contoh: `user.view`, `product.create`, `sale.void`. Tabel ini adalah konfigurasi default dari seeds; dapat diubah via Role Management UI.

| Permission | Superadmin | Admin | Manager | Cashier | Staff |
|------------|:---:|:---:|:---:|:---:|:---:|
| `dashboard.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `product.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `product.update` | ✅ | ✅ | ✅ | – | ✅ |
| `product.create`, `product.delete` | ✅ | ✅ | – | – | – |
| `product.import`, `product.export` | ✅ | ✅ | – | – | – |
| `category.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `category.create` | ✅ | ✅ | ✅ | – | – |
| `category.update`, `category.delete` | ✅ | ✅ | – | – | – |
| `category.import`, `category.export` | ✅ | ✅ | – | – | – |
| `sale.view`, `sale.create`, `sale.print`, `sale.park` | ✅ | ✅ | ✅ | ✅ | – |
| `sale.void` | ✅ | ✅ | ✅ | – | – |
| `shift.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `shift.create` | ✅ | ✅ | ✅ | ✅ | – |
| `shift.review`, `shift.audit` | ✅ | ✅ | ✅ | – | – |
| `inventory.adjust` | ✅ | – | ✅ | – | ✅ |
| `report.view` | ✅ | ✅ | ✅ | – | – |
| `customer.view` | ✅ | ✅ | ✅ | ✅ | – |
| `customer.create`, `customer.update` | ✅ | ✅ | ✅ | – | – |
| `customer.delete` | ✅ | ✅ | – | – | – |
| `customer.import`, `customer.export` | ✅ | ✅ | – | – | – |
| `customer_group.view` | ✅ | ✅ | ✅ | ✅ | – |
| `customer_group.create/update/delete` | ✅ | ✅ | – | – | – |
| `store.view` | ✅ | ✅ | ✅ | ✅ | – |
| `store.create/update/delete` | ✅ | ✅ | – | – | – |
| `pricing.view` | ✅ | ✅ | ✅ | ✅ | – |
| `pricing.create`, `pricing.update` | ✅ | ✅ | ✅ | – | – |
| `pricing.delete` | ✅ | ✅ | – | – | – |
| `purchase_order.view/create/update/confirm/receive` | ✅ | ✅ | ✅ | – | – |
| `purchase_order.delete` | ✅ | – | – | – | – |
| `purchase_order.cancel` | ✅ | ✅ | ✅ | – | – |
| `user.view`, `user.create`, `user.update` | ✅ | ✅ | – | – | – |
| `user.delete` | ✅ | – | – | – | – |
| `role.view`, `role.create` | ✅ | ✅ | – | – | – |
| `role.update`, `role.delete` | ✅ | – | – | – | – |
| `audit.view` | ✅ | – | – | – | – |

---

## Testing

### Backend Tests

```bash
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos TEST_DB_PASSWORD=admin123 \
DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only \
go test -p 1 -count=1 ./...
```

### Coverage (exclude `cmd/` dan `tools/`)

```bash
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only \
go test -p 1 -count=1 -coverprofile=coverage.out $(go list ./... | grep -v -E '(retail-pos-system/cmd/|retail-pos-system/tools/)')
```

### Frontend Unit Tests (Vitest)

```bash
cd web && npm run test:run
```

### E2E Tests (Playwright)

```bash
cd web && npx playwright test --reporter=list
```

> E2E membutuhkan server backend + frontend yang berjalan (`./run-dev.sh` dan `npm run dev`).

### Test Database

Tests terhubung ke database `retail_pos_test` (konfigurasi via `TEST_DB_*` env vars). Framework test auto-apply migrasi pending. Jika schema test tidak sinkron: `dropdb retail_pos_test && createdb retail_pos_test`, lalu jalankan ulang tests.

---

## License

Proprietary - Developed for retail business use.
