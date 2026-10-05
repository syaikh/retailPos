# Retail POS System

Retail POS System is a modern Point of Sale (POS) application for retail stores with inventory management, sales, purchase orders, pricing engine, reporting, shift management, and role-based access control.

> This document is the developer reference: architecture, setup, API, configuration,
> deployment, testing, and the print agent. The **end-user manual** for shop staff is a
> separate document — see [User Manual](docs/guides/user-manual.md).

## Table of Contents

- [Part A — Project Overview & Features](#part-a--project-overview--features)
  - [Features](#features)
  - [Security Features](#security-features)
- [Part B — Architecture](#part-b--architecture)
  - [Development](#development)
  - [Production (Podman / Docker Compose)](#production-podman--docker-compose)
  - [Tech Stack](#tech-stack)
- [Part C — Quick Start](#part-c--quick-start)
  - [Prerequisites](#prerequisites)
  - [Development](#development-1)
  - [Production](#production)
- [Part D — Developer Guide](#part-d--developer-guide)
  - [Backend](#backend)
  - [API Reference](#api-reference)
  - [Frontend](#frontend)
  - [Configuration](#configuration)
  - [Deployment](#deployment)
  - [Default Credentials](#default-credentials)
  - [Permission Matrix](#permission-matrix)
  - [Testing](#testing)
  - [Print Agent (Go)](#print-agent-go)
- [User Manual](docs/guides/user-manual.md) — end-user manual for shop staff, published separately

---

## Part A — Project Overview & Features

## Features

- **Point of Sale (POS)** — Sales transactions with scanner, discounts, split payment (multi payment methods), and hold & recall (parked sales)
- **Purchase Order & Goods Receiving** — Purchasing workflow from suppliers: draft → confirmed → received, partial goods receiving, auto-generate PO/GR/DO numbers
- **Stock Opname** — Physical stock count sessions with 9-state workflow (draft → open → counting → verification → needs_recount → approved → posted → closed/cancelled), multi-scope sessions (store/warehouse/category/product), multi-counter assignment, blind count, recount workflow, adjustment ledger (IA- documents), and auto-adjustment of stock upon posting (FR-001 through FR-044)
- **Storage Locations** — Storage location master data (racks/warehouses) with warehouse/store scope, CRUD + bulk actions (phase 1 of per-rack stock tracking)
- **Store Management** — Store/outlet CRUD + store management UI page (list, active/inactive status)
- **Shift Management** — Cashier shift open/close, opening/closing balance, discrepancy review & audit
- **Pricing Engine** — Price rules (special price / promotion) by product, category, brand, customer group, and store; approval workflow (`pending` → `approved`/`rejected`, with a self-approval guard on the author); real-time price resolver
- **Supplier Management** — Supplier CRUD, product-supplier links, preferred supplier, bulk actions, auto-generate codes (SUP-XXXXXX)
- **Konsinyasi Supplier (Consignment)** — Full consignment management: agreements, goods receiving, returns, settlements, payouts, consignment stock, POS checkout integration
- **Application Settings** — Global settings (store branding, jargon, logo) for superadmin and manager to view (superadmin edits), receipt info per branch, per-user preferences (theme/light-dark, language)
- **Customer & Customer Groups** — Customer management, customer groups (Walk-in, Member, VIP), bulk actions
- **Multi-Warehouse & Multi-Store** — Inventory per warehouse/store with composite unique key, store management
- **Inventory Management** — Stock tracking, movement, low stock alerts, stock thresholds, multi-category filter
- **Import & Export Framework** — Schema-driven reusable import/export for Products, Categories, Brands, UOMs, Customers, Pricing Rules, Suppliers with XLSX templates, preview, validation, reference dropdowns, import history (async job), and cancel. Customer Groups and Stores are registered as schemas (template download works) but their operations currently fail with 403 — see [Import & Export](docs/guides/user-manual.md#18-import--export)
- **User Management** — RBAC (Role-Based Access Control) with dot-notation permissions, manager-subordinate hierarchy (org chart), soft delete
- **Audit Logging** — Full audit trail for all actions (including login/logout, import, change-password)
- **Real-time Dashboard** — Sales statistics, revenue, analytics + live updates via WebSocket, daily/weekly/monthly charts, period comparison, pricing breakdown
- **WebSocket Support** — Real-time notifications (dashboard live, PO updates)
- **Swagger/OpenAPI** — API documentation via swaggo annotations
- **Structured Logging** — JSON (production) / text (development) via `log/slog`
- **EventBus Observability** — Atomic metrics for published/consumed/failed events
- **Dead-Letter Queue** — Failed events stored to PostgreSQL for retry
- **Materialized Views** — Pre-aggregated daily/hourly sales data for fast reporting queries; refresh coordinated by `report.RefreshCoordinator` — once at startup, then at each Jakarta hour boundary, with exponential-backoff retries after a failure. A `sale.created` event only invalidates the dashboard cache; it never triggers a refresh

### Security Features

- JWT authentication with refresh token (HTTP-only cookie, separate refresh secret)
- CSRF protection on state-changing endpoints (validate, logout, change-password)
- Rate limiting with per-entry TTL (separate for login, refresh, and general API)
- IP spoofing protection (uses `RemoteAddr` not `X-Forwarded-For`)
- Product search via tsvector (avoids ILIKE full table scan)
- Inventory adjustments use `SELECT ... FOR UPDATE` for concurrency safety
- Security headers middleware (CSP, X-Frame-Options, etc.), body limit, gzip

---

## Part B — Architecture

### Development

```
┌──────────────────────────────────────────────────────────┐
│  Frontend (Vite dev server)   http://localhost:5173      │
│  Svelte 5 + Tailwind CSS 4 + Vite 6 (HMR)                │
└──────────────┬───────────────────────────────────────────┘
               │ /api/* → Backend, /ws/* → WebSocket
┌──────────────┴───────────────────────────────────────────┐
│  Go Backend (Gin)        http://localhost:9095           │
│  PostgreSQL              localhost:5433 (postgres-dev)   │
│  `./run-dev.sh` rebuild + restart automatically (press r)│
└──────────────────────────────────────────────────────────┘
```

### Production (Podman / Docker Compose)

```
┌──────────────────────────────────────────────────────────┐
│  Nginx Frontend            Port 8000 → 8081              │
│  Go Backend                Port 8080 (internal)          │
│  PostgreSQL 18            Volume retail-pos-postgres-data│
│  Pod retail-pos-pod        One shared network namespace  │
│  `./deploy/podman-deploy.sh start`                       │
└──────────────────────────────────────────────────────────┘
```

### Tech Stack

- **Backend:** Go (Gin), PostgreSQL 18 (pgx), JWT Auth, WebSocket (gorilla/websocket), structured logging (slog)
- **Frontend:** Svelte 5, Tailwind CSS 4, Vite 6, Chart.js, jsPDF, Playwright, Vitest
- **Infrastructure:** Podman (rootless), Docker Compose, Nginx, systemd

---

## Part C — Quick Start

### Prerequisites

```bash
# Backend
go version  # 1.26+ (see go.mod)

# Frontend
cd web && npm install

# Database (dev)
podman run -d --name postgres-dev -p 5433:5432 \
  -e POSTGRES_USER=pos -e POSTGRES_PASSWORD=admin123 -e POSTGRES_DB=retail_pos \
  postgres:18-alpine
```

### Development

```bash
# 1. Copy configuration
cp .env.example .env

# 2. Seed database (dummy data)
./seed-dev.sh            # large data (products, transactions)
./seed-daily-dev.sh      # daily transactions only

# 3. Start backend (port 9095, auto-reload via r/q keys)
./run-dev.sh

# 4. Start frontend (port 5173)
cd web && npm run dev

# 5. Open http://localhost:5173
```

### Production

```bash
make build-all                 # Build backend + frontend images
./deploy/podman-deploy.sh start   # Start all services
./deploy/podman-deploy.sh migrate # Run migrations (required before first start; not automatic)
./deploy/podman-deploy.sh seed    # Seed initial data (optional)
```

Run `(cd web && npm ci)` once after cloning — `web/dist` is gitignored and the
deployment script rebuilds it before packaging the frontend image. See
[Deployment](#deployment) for the details.

---
## Part D — Developer Guide

### Backend

#### Module Structure

```
internal/
├── appsettings/       # Application settings (branding, receipt, per-user preferences)
├── audit/             # Audit logging (domain events + listener)
├── brand/             # Brand CRUD + import adapter
├── category/          # Category CRUD + import adapter
├── config/            # App configuration (env, timezone)
├── consignment/       # Consignment supplier (arrangements, receipts, returns, settlements, payouts)
├── customergroup/     # Customer group CRUD + bulk actions
├── customer/          # Customer CRUD + bulk actions + import adapter
├── eventbus/          # In-process event bus (retry, dead-letter, metrics)
├── events/            # Event type definitions (domain event structs)
├── inventory/         # Stock tracking, adjustments, low stock, per-location stock
├── middleware/        # Auth (JWT), CORS, rate limit, CSRF, security headers
├── ownership/         # Ownership helper utilities
├── permissions/       # Permission code definitions and RBAC checking
├── platform/
│   └── importexport/  # Schema-driven import/export framework
├── pricing/           # Pricing rules engine + resolver + approval workflow
├── product/           # Product CRUD (repository + query + bulk)
├── purchase/          # Purchase orders + goods receipts
├── report/            # Dashboard stats, charts, comparisons
├── sale/              # POS transaction, split payment, parked sales, export
├── shared/            # Shared types, logger, response helpers
├── shift/             # Cashier shift management
├── stockopname/       # Stock opname sessions (count, verify, post adjustment)
├── storagelocation/   # Storage locations (racks/shelves) CRUD
├── store/             # Store CRUD
├── supplier/          # Supplier CRUD + product-supplier links
├── uom/               # Unit of Measure CRUD + import adapter
├── user/              # User & role management + auth (login/refresh)
├── wiring/            # Dependency injection / wiring
└── pkg/
    └── websocket/     # WebSocket hub
```

#### Key Files

| File | Description |
|------|-------------|
| `cmd/server/main.go` | HTTP + WebSocket server entry point (routing, middleware, graceful shutdown) |
| `cmd/server/e2e_test.go` | End-to-end API tests |
| `internal/wiring/wiring.go` | Dependency wiring |
| `internal/eventbus/bus.go` | Event bus with retry, dead-letter, observability |
| `internal/pricing/resolver.go` | Final price resolver (rule → effective price) |
| `internal/purchase/service.go` | Purchase order & goods receipt logic |
| `internal/shift/service.go` | Shift lifecycle (open/close/review/audit) |
| `internal/stockopname/service.go` | Stock opname workflow (9-state lifecycle, count/verify/post) |
| `internal/storagelocation/service.go` | Storage location CRUD |
| `internal/consignment/service.go` | Consignment supplier (arrangements, receipts, returns, settlements) |
| `internal/appsettings/handler.go` | Application settings (branding, receipt info, per-user preferences) |
| `internal/sale/service.go` | POS transaction, parked sales, split payment |
| `internal/shared/logger.go` | Structured logging (slog) |
| `database/migrations/000_baseline.sql` | Version 1 baseline: complete schema + reference data (role, user, product, sale, inventory, etc.) |
| `docs/swagger.go` | OpenAPI annotations |

#### Run Tests

```bash
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos TEST_DB_PASSWORD=admin123 \
DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only \
go test -p 1 -count=1 ./...
```

> `-p 1` forces sequential execution to avoid deadlocks from concurrent TRUNCATE/INSERT across packages. Tests connect to the `retail_pos_test` database.

#### API Documentation

Swagger annotations are on key endpoints. To generate the spec:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g docs/swagger.go -o docs
```

Spec is accessible at `/swagger/*any` while the server is running.

### API Reference

#### API Endpoints

Base path: `/api`. All endpoints require JWT (via `Authorization: Bearer` or cookie) unless stated as "Public".

##### Auth

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/login` | Login (set cookie refresh_token) | No |
| POST | `/refresh` | Refresh access token | No |
| POST | `/validate` | Validate session + permission list | Yes |
| POST | `/logout` | Logout (revoke refresh token) | Yes |
| POST | `/change-password` | Change own password | Yes |

##### Dashboard & Reports

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/dashboard/stats` | Dashboard statistics (today's revenue, etc.) | `dashboard.view` |
| GET | `/dashboard/live` | Live statistics (WebSocket) | `dashboard.view` |
| GET | `/dashboard/chart` | Sales chart data | `report.view` |
| GET | `/dashboard/chart/weekly` | Weekly report | `report.view` |
| GET | `/dashboard/chart/monthly` | Monthly report | `report.view` |
| GET | `/dashboard/comparison` | Period comparison | `report.view` |
| POST | `/dashboard/export` | Export dashboard (CSV/XLSX) | `report.view` |
| GET | `/dashboard/years` | Available years | `report.view` |
| GET | `/dashboard/pricing-breakdown` | Pricing breakdown | `report.view` |

##### Products, Categories, Brands, UOM

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/products` | Product list (search + multi-category filter) | Public |
| GET | `/products/next-sku` | Next SKU generator | Public |
| GET | `/products/options` | Distinct filter options (categories, brands, UOM, stores) | Public |
| GET | `/products/:id` | Product detail | Yes |
| POST | `/products` | Create product | `product.create` |
| PUT | `/products/:id` | Update product | `product.update` |
| DELETE | `/products/:id` | Delete product | `product.delete` |
| POST | `/products/bulk/status` | Bulk update status | `product.update` |
| GET | `/categories` | Category list | Public |
| GET | `/categories/manage` | Category list (management, paginated) | `category.view` |
| POST | `/categories` | Create category | `category.create` |
| PUT | `/categories/:id` | Update category | `category.update` |
| DELETE | `/categories/:id` | Delete category | `category.delete` |
| GET | `/brands` | Brand list | Public |
| POST | `/brands` | Create brand | `product.create` |
| PUT | `/brands/:id` | Update brand | `product.update` |
| DELETE | `/brands/:id` | Delete brand | `product.delete` |
| GET | `/units-of-measure` | UOM list | Public |
| POST | `/units-of-measure` | Create UOM | `product.create` |
| PUT | `/units-of-measure/:id` | Update UOM | `product.update` |
| DELETE | `/units-of-measure/:id` | Delete UOM | `product.delete` |
| GET | `/tax-classes` | List tax classes | Public |
| GET | `/stock-thresholds` | Stock thresholds | Public |

##### Sales

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| POST | `/sales` | Create transaction (split payment, parked recall) | `sale.create` |
| GET | `/sales` | Sales history | `sale.view` |
| GET | `/sales/:id` | Sales detail | `sale.view` |
| GET | `/sales/export` | Export sales (CSV/XLSX) | `report.view` |
| GET | `/sales/lookup` | Find Transaction search (barcode/receipt no.) | `sale.lookup` |
| GET | `/sales/lookup/:id` | Find Transaction result detail | `sale.detail` |
| POST | `/sales/parked` | Park (hold) transaction | `sale.park` |
| GET | `/sales/parked` | Parked sales list | `sale.park` |
| GET | `/sales/parked/:id` | Parked sale detail | `sale.park` |
| POST | `/sales/parked/:id/recall` | Recall parked sale | `sale.park` |
| POST | `/sales/parked/:id/complete` | Complete parked sale (checkout) | `sale.park` |
| DELETE | `/sales/parked/:id` | Cancel parked sale | `sale.park` |
| GET | `/payment-methods` | Payment method list | Public |
| GET | `/payment-methods/:code` | Payment method detail | Yes |

###### POS Cart

The POS screen drives a server-side cart session; `/pos/cart` is the live path, while
`/sales/parked` remains for parked transactions created outside a cart. Every cart route
requires `sale.create`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/pos/cart` | Create or resume the caller's open cart |
| GET | `/pos/cart` | Current open cart |
| GET | `/pos/cart/:id` | Cart detail |
| GET | `/pos/cart/held` | Held carts (hold TTL from `CART_HOLD_TTL_HOURS`) |
| POST | `/pos/cart/items` | Add line item |
| PATCH | `/pos/cart/items/:itemId` | Update line quantity |
| DELETE | `/pos/cart/items/:itemId` | Remove line item |
| PATCH | `/pos/cart/:id/customer` | Attach customer |
| POST | `/pos/cart/:id/hold` | Hold cart |
| POST | `/pos/cart/:id/resume` | Resume held cart |
| POST | `/pos/cart/:id/cancel` | Cancel cart |
| POST | `/pos/cart/:id/checkout` | Checkout cart into a sale |

##### Inventory

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| POST | `/inventory/adjust` | Manual stock adjustment | `inventory.adjust` |
| GET | `/inventory/locations` | View stock per location (rack) | `product.view` |
| POST | `/inventory/locations` | Set stock at location (rack) | `inventory.adjust` |
| POST | `/inventory/locations/transfer` | Transfer stock between locations | `inventory.adjust` |

##### Stock Opname

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| POST | `/stock-opnames` | Create Stock Opname session (multi-scope snapshot) | `stock_opname.create` |
| GET | `/stock-opnames` | Session list (filter status/scope, pagination) | `stock_opname.view` |
| GET | `/stock-opnames/:id` | Session detail | `stock_opname.view` |
| GET | `/stock-opnames/assignable-users` | List assignable users | `stock_opname.assign` |
| POST | `/stock-opnames/:id/open` | Open session (Draft → Open) | `stock_opname.create` |
| POST | `/stock-opnames/:id/cancel` | Cancel session (draft/open/counting/needs_recount) | `stock_opname.cancel` |
| POST | `/stock-opnames/:id/assignments` | Assign counter/supervisor | `stock_opname.assign` |
| GET | `/stock-opnames/:id/assignments` | Session assignment list | `stock_opname.view` |
| PUT | `/stock-opnames/:id/assignments/:assignmentId` | Reassign counter | `stock_opname.assign` |
| PUT | `/stock-opnames/items/:itemId/count` | Save counting results (autosave) | `stock_opname.count` |
| GET | `/stock-opnames/items/:itemId/counts` | Counting history per item | `stock_opname.view` |
| POST | `/stock-opnames/:id/start` | Start counting (Draft/Open → Counting) | `stock_opname.count` |
| POST | `/stock-opnames/:id/submit` | Submit counting results (Counting → Verification) | `stock_opname.submit` |
| POST | `/stock-opnames/:id/verify` | Verify (persist differences, does not change stock yet; Verification → Approved) | `stock_opname.verify` |
| POST | `/stock-opnames/:id/reject` | Reject session (Verification → Needs Recount) | `stock_opname.verify` |
| POST | `/stock-opnames/:id/recount` | Request recount (Verification → Needs Recount) | `stock_opname.recount` |
| POST | `/stock-opnames/:id/resume` | Resume counting (Needs Recount → Counting) | `stock_opname.count` |
| POST | `/stock-opnames/:id/post-adjustment` | Post adjustment to stock + create IA- document (Approved → Posted) | `stock_opname.post` |
| POST | `/stock-opnames/:id/close` | Close session (Posted → Closed) | `stock_opname.close` |
| GET | `/stock-opnames/:id/summary` | Session progress summary | `stock_opname.view` |
| GET | `/stock-opnames/:id/difference` | Stock difference report | `stock_opname.view` |
| GET | `/stock-opnames/:id/export` | Export report (CSV/Excel/PDF) | `stock_opname.export` |
| GET | `/stock-opnames/adjustments` | Adjustments report (IA- documents) | `stock_opname.report` |
| GET | `/stock-opnames/adjustments/:id` | Adjustment document detail | `stock_opname.report` |

##### Storage Locations

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/storage-locations` | Storage location list (search, filter is_active) | `storage_location.view` |
| GET | `/storage-locations/:id` | Location detail | `storage_location.view` |
| POST | `/storage-locations` | Create location (scope warehouse/store) | `storage_location.create` |
| PUT | `/storage-locations/:id` | Update location | `storage_location.update` |
| DELETE | `/storage-locations/:id` | Delete location | `storage_location.delete` |
| PUT | `/storage-locations/bulk` | Bulk update | `storage_location.update` |
| DELETE | `/storage-locations/bulk` | Bulk delete | `storage_location.delete` |

##### Customers & Customer Groups

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/customers` | Customer list | `customer.view` |
| GET | `/customers/:id` | Customer detail | `customer.view` |
| POST | `/customers` | Create customer | `customer.create` |
| PUT | `/customers/:id` | Update customer | `customer.update` |
| DELETE | `/customers/:id` | Delete customer | `customer.delete` |
| POST | `/customers/bulk/status` | Bulk update status | `customer.update` |
| POST | `/customers/bulk/delete` | Bulk delete | `customer.delete` |
| GET | `/customer-groups` | Customer group list | `customer_group.view` |
| GET | `/customer-groups/:id` | Group detail | `customer_group.view` |
| POST | `/customer-groups` | Create group | `customer_group.create` |
| PUT | `/customer-groups/:id` | Update group | `customer_group.update` |
| DELETE | `/customer-groups/:id` | Delete group | `customer_group.delete` |
| PUT | `/customer-groups/bulk` | Bulk update | `customer_group.update` |
| DELETE | `/customer-groups/bulk` | Bulk delete | `customer_group.delete` |

##### Stores

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/stores` | Store list | `store.view` |
| GET | `/stores/active` | Active store list | `store.view` |
| GET | `/stores/:id` | Store detail | `store.view` |
| GET | `/stores/:id/readiness` | Onboarding readiness (blockers + ready flag) | `store.view` |
| GET | `/warehouses` | Warehouse list (store-scoped) | Authenticated |
| POST | `/stores` | Create store | `store.create` |
| PUT | `/stores/:id` | Update store | `store.update` |
| DELETE | `/stores/:id` | Delete store | `store.delete` |

##### Purchase Orders & Goods Receiving

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| POST | `/purchase-orders` | Create PO draft | `purchase_order.create` |
| GET | `/purchase-orders` | PO list (filter status/supplier) | `purchase_order.view` |
| GET | `/purchase-orders/:id` | PO detail | `purchase_order.view` |
| PUT | `/purchase-orders/:id` | Update PO draft | `purchase_order.update` |
| DELETE | `/purchase-orders/:id` | Delete PO draft | `purchase_order.delete` |
| POST | `/purchase-orders/:id/confirm` | Confirm PO | `purchase_order.confirm` |
| POST | `/purchase-orders/:id/cancel` | Cancel PO | `purchase_order.cancel` |
| GET | `/purchase-orders/:id/receipts` | PO goods receipts list | `purchase_order.view` |
| POST | `/goods-receipts` | Receive goods (auto-generate GR & DO number) | `purchase_order.receive` |

##### Pricing Engine & Suppliers

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/pricing-rules` | Pricing rules list | `pricing.view` |
| GET | `/pricing-rules/:id` | Rule detail | `pricing.view` |
| POST | `/pricing-rules` | Create rule | `pricing.create` |
| PUT | `/pricing-rules/:id` | Update rule | `pricing.update` |
| DELETE | `/pricing-rules/:id` | Delete rule | `pricing.delete` |
| POST | `/pricing-rules/check-conflicts` | Check rule conflicts | `pricing.view` |
| POST | `/pricing-rules/:id/approve` | Approve rule | `pricing.approve` |
| POST | `/pricing-rules/:id/reject` | Reject rule | `pricing.approve` |
| POST | `/pricing/resolve` | Resolve final price | `pricing.view` |
| GET | `/products/search` | Search products (for pricing) | `pricing.view` |

Suppliers have their own permission codes — they are **not** gated by `pricing.*`:

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/suppliers` | Supplier list | `supplier.view` |
| GET | `/suppliers/:id` | Supplier detail | `supplier.view` |
| GET | `/suppliers/:id/usage` | Supplier usage summary (receipts, payables, terms) | `supplier.view` |
| POST | `/suppliers` | Create supplier | `supplier.create` |
| PUT | `/suppliers/:id` | Update supplier (optimistic concurrency via `version`) | `supplier.update` |
| DELETE | `/suppliers/:id` | Delete supplier (soft delete) | `supplier.delete` |
| PUT | `/suppliers/bulk` | Bulk update | `supplier.update` |
| DELETE | `/suppliers/bulk` | Bulk delete | `supplier.delete` |
| GET | `/suppliers/:id/products` | Products from supplier | `supplier.view` |
| POST | `/suppliers/:id/products` | Link product to supplier | `supplier.update` |
| PUT | `/suppliers/:id/products/:productId` | Update relation (unit_cost, store scope) | `supplier.update` |
| DELETE | `/suppliers/:id/products/:productId` | Unlink product | `supplier.update` |
| POST | `/suppliers/:id/products/:productId/preferred` | Set preferred supplier (unique per product + store) | `supplier.update` |
| GET | `/products/:id/suppliers` | Suppliers for product | `supplier.view` |

##### Consignment (Konsinyasi Supplier)

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/consignment/suppliers` | Consignment supplier list | `consignment.view` |
| GET | `/consignment/arrangements` | Consignment arrangement list | `consignment.view` |
| POST | `/consignment/arrangements` | Create arrangement | `consignment.create` |
| GET | `/consignment/arrangements/:id` | Arrangement detail | `consignment.view` |
| GET | `/consignment/arrangements/:id/available-products` | Products eligible for a new term | `consignment.view` |
| POST | `/consignment/arrangements/:id/end` | End arrangement | `consignment.update` |
| POST | `/consignment/arrangements/:id/terms` | Add product term | `consignment.update` |
| PUT | `/consignment/arrangements/:id/terms` | Update terms/conditions | `consignment.update` |
| DELETE | `/consignment/arrangements/:id/terms/:productId` | Remove product term | `consignment.update` |
| GET | `/consignment/receipts` | Goods receipt list | `consignment.view` |
| POST | `/consignment/receipts` | Create goods receipt | `consignment.create` |
| GET | `/consignment/receipts/:id` | Receipt detail | `consignment.view` |
| PUT | `/consignment/receipts/:id` | Update goods receipt | `consignment.update` |
| GET | `/consignment/stock` | Consignment stock | `consignment.view` |
| GET | `/consignment/pending-returns` | Pending returns | `consignment.view` |
| POST | `/consignment/pending-returns` | Create pending return | `consignment.update` |
| GET | `/consignment/returns` | Return list | `consignment.view` |
| POST | `/consignment/returns` | Create formal return | `consignment.create` |
| GET | `/consignment/returns/:id` | Return detail | `consignment.view` |
| GET | `/consignment/settlements/preview` | Settlement preview | `consignment.settle` |
| GET | `/consignment/settlements` | Settlement list | `consignment.view` |
| POST | `/consignment/settlements` | Create settlement | `consignment.settle` |
| GET | `/consignment/settlements/:id` | Settlement detail | `consignment.view` |
| GET | `/consignment/payment-methods` | Consignment payment methods | `consignment.pay` or `consignment.settle` |
| POST | `/consignment/settlements/:id/payouts` | Create payment to supplier | `consignment.pay` |

##### Shifts

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| POST | `/shifts/open` | Open shift | `shift.create` |
| POST | `/shifts/:id/close` | Close shift | `shift.create` |
| POST | `/shifts/:id/review` | Review shift discrepancy | `shift.review` |
| POST | `/shifts/:id/audit` | Physical cash audit | `shift.audit` |
| GET | `/shifts/active` | Current active shift | Yes |
| GET | `/shifts` | Shift list | `shift.view` |
| GET | `/shifts/export` | Export shifts | `shift.view` |
| GET | `/shifts/:id` | Shift detail | `shift.view` |
| GET | `/shifts/:id/report` | Shift report (sales, payments, cash variance) | `shift.view` |
| GET | `/shifts/:id/cash-movements` | Cash movement list | `shift.view` |
| POST | `/shifts/:id/cash-movements` | Record cash in/out movement | `shift.cash_movement` |

##### User & Role Management

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/admin/users` | User list | `user.view` |
| POST | `/admin/users` | Create user | `user.create` |
| PUT | `/admin/users/:id` | Update user | `user.update` |
| DELETE | `/admin/users/:id` | Delete user (soft delete) | `user.delete` |
| GET | `/admin/users/:id/subordinates` | User's subordinates | `user.view` |
| GET | `/admin/users/:id/manager` | User's manager | `user.view` |
| GET | `/admin/users/org-chart` | Org chart | `user.view` |
| GET | `/admin/roles` | Role list | `role.view` |
| POST | `/admin/roles` | Create role | `role.create` |
| PUT | `/admin/roles/:id` | Update role | `role.update` |
| PUT | `/admin/roles/:id/permissions` | Update role permissions | `role.update` |
| DELETE | `/admin/roles/:id` | Delete role | `role.delete` |
| GET | `/admin/permissions` | List all permissions | `role.view` |
| PUT | `/users/me/preferences` | Update user preferences (theme, language) | Yes |

##### Audit Logs

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/audit-logs` | Audit log list (filter date, action, entity) | `audit.view` |
| GET | `/audit-logs/:id` | Audit log detail | `audit.view` |
| GET | `/audit-logs/export` | Export audit logs | `audit.export` |
| GET | `/audit-logs/entity-types` | Entity type list | `audit.view` |

##### Import & Export

Registered schemas: `products`, `categories`, `customer_groups`, `brands`, `uoms`, `customers`, `pricing_rules`, `suppliers`, `stores`.

Import, export, and history are gated by a per-module permission map
(`modulePerms` in `internal/platform/importexport/handler/handler.go`). `customer_groups` and
`stores` have no entry, so those operations currently answer **403 "permission not defined for
module"** even though `/import-export/modules` advertises them and their template download
works.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/import-export/modules` | List importable modules |
| GET | `/import-export/template/:module` | Download XLSX template |
| POST | `/import-export/preview/:module` | Import preview (validation) |
| POST | `/import-export/confirm/:module` | Confirm import (async job) |
| GET | `/import-export/progress/:jobId` | Import job progress |
| POST | `/import-export/cancel/:jobId` | Cancel job |
| GET | `/import-export/history/:module` | Import history per module |
| GET | `/import-export/history/:module/:jobId` | Job snapshot detail |
| GET | `/import-export/history/:module/:jobId/rows` | Import result rows |
| GET | `/import-export/export/:module` | Export data (CSV/XLSX) |

##### Application Settings

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/settings/public` | Public branding (store name, jargon) | No |
| GET | `/settings/logo` | Store logo | No |
| GET | `/settings` | All settings (branding, receipt text, branch contact) | Any authenticated |
| PUT | `/settings` | Update settings | `app_settings.update` |
| POST | `/settings/logo` | Upload logo | `app_settings.update` |
| DELETE | `/settings/logo` | Delete logo | `app_settings.update` |

Reading `GET /settings` is deliberately not gated by `app_settings.view` — receipts render it on
every screen, including the cashier POS. The Settings *screen* is still gated by
`app_settings.view` in the frontend route map; only the read is relaxed.

##### System

Unauthenticated endpoints below are registered at the **router root**, not under the `/api` base path.

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/health` | Health check | No |
| GET | `/metrics` | EventBus counters (JSON), incl. `ReportRefreshCoord` metrics | No |
| GET | `/ws` | WebSocket hub | Yes |
| GET | `/swagger/*any` | Swagger UI | No |

### Frontend

#### Development

```bash
cd web
npm run dev       # Start dev server (port 5173)
npm run build     # Build for production
npm run test:run  # Unit test (Vitest)
npx playwright test  # E2E — run from the repository root, not web/
```

> `web/package.json` has no `test:e2e` script. Playwright's config and the `test:e2e*` scripts live in the repository root `package.json`.

#### Module Structure

```
web/src/
├── app/               # App shell (main.svelte, router, providers, config permissions)
├── modules/           # Feature modules
│   ├── admin/         # Users, roles, audit logs
│   ├── auth/          # Login, session
│   ├── consignment/   # Consignment supplier (arrangements, receipts, returns, settlements)
│   ├── customer-groups/ # Customer groups management
│   ├── customers/     # Customer management
│   ├── dashboard/     # Charts, stats, live updates
│   ├── import-export/ # Import wizard, history
│   ├── inventory/     # Stock management, per-location stock
│   ├── pos/           # Point of Sale (split payment, parked sales)
│   ├── pricing/       # Pricing rules + approval
│   ├── product/       # Product catalog
│   ├── purchase-orders/ # PO + goods receiving
│   ├── reporting/     # Reports with chart config + export
│   ├── sales/         # Sales history
│   ├── settings/      # Application settings (branding, per-user preferences)
│   ├── shifts/        # Shift management
│   ├── stock-opname/  # Stock opname (list, detail, counting, adjustments report)
│   ├── storage-location/ # Storage locations management
│   ├── stores/        # Store management
│   └── supplier/      # Supplier management
├── shared/            # API client (axios), websocket, services, stores, types, utils (Jakarta time, permissions)
│   └── ui/            # Shared UI components (Modal, DataTable, Pagination, etc.)
├── app.css            # Global styles & Tailwind imports
└── main.js            # Entry point
```

#### Jakarta Timezone

Backend stores data in UTC, but **all queries use the Asia/Jakarta timezone**. The frontend calculates Jakarta dates in UTC before sending to the API:

- Jakarta midnight = UTC 07:00 (7-hour offset)
- Utilities: `getTodayInJakarta()`, `getDateNDaysAgoInJakarta()` in `web/src/shared/utils/jakartaTime.ts`
- Backend parses dates with `time.ParseInLocation("2006-01-02", dateStr, jakartaLoc)`

### Configuration

#### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | (required) | 256-bit secret for JWT signing. Generate: `openssl rand -hex 32` |
| `JWT_SECRET_REFRESH` | `JWT_SECRET` | Separate secret for refresh tokens (recommended to differ in production) |
| `DATABASE_URL` | (empty) | Full PostgreSQL URL; if empty, built from `DB_*` |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5433` (dev) | PostgreSQL port |
| `DB_USER` | `pos` | Database username |
| `DB_PASSWORD` | `admin123` | Database password |
| `DB_NAME` | `retail_pos` | Database name |
| `DB_SSLMODE` | `require` (prod) / `disable` (dev) | libpq sslmode. Defaults to `require` when `ENV=production`; the stock `postgres:18-alpine` image ships `ssl=off`, so production must either mount a certificate or set this explicitly |
| `ENV` | `development` | `development` (text log) / `production` (JSON log, release mode, sslmode require) |
| `LOG_LEVEL` | `debug`/`info` | Log level: debug, info, warn, error |
| `CORS_ORIGIN` | `http://localhost:5173` (dev) | Allowed CORS origin (must not be `*` in production). Production runs on `http://localhost:8000` |
| `PORT` | `9095` | HTTP server port |
| `FRONTEND_PORT` | `5173` | Frontend dev server port (Vite) |
| `BACKEND_PORT` | `9095` | Backend dev server port (Go) |
| `DATABASE_PORT` | `5433` | Development database port (postgres-dev container) |
| `COOKIE_DOMAIN` | (empty) | Refresh token cookie domain |
| `COOKIE_SECURE` | `false` | Set to `true` for HTTPS |
| `LOGIN_RATE_LIMIT_RPM` | `5` | Login rate limit (per minute) |
| `LOGIN_RATE_LIMIT_BURST` | `5` | Login burst |
| `RATE_LIMIT_RPS` | `50` | General API rate limit (per second) |
| `RATE_LIMIT_BURST` | `100` | General API burst |
| `REFRESH_RATE_LIMIT_RPM` | `10` | Refresh rate limit (per minute) |
| `REFRESH_RATE_LIMIT_BURST` | `10` | Refresh burst |
| `WS_RATE_LIMIT_RPM` | `20` | WebSocket upgrade rate limit (per minute) |
| `WS_RATE_LIMIT_BURST` | `5` | WebSocket burst |
| `STOCK_WARNING_THRESHOLD` | `10` | Stock below this = "needs attention" |
| `STOCK_CRITICAL_THRESHOLD` | `5` | Stock below this = "low stock" |
| `CART_HOLD_TTL_HOURS` | `24` | How many hours a cart session is held before considered expired |
| `REPORT_REFRESH_DEBOUNCE` | `30` | Base retry delay in seconds for a **failed** materialized-view refresh; retries use exponential backoff. Not a `sale.created` debounce — refreshes happen at startup and on each Jakarta hour boundary |
| `VITE_PRINT_MODE` | (empty) | Receipt print mode (`preview` renders in-browser). Frontend build-time only |
| `VITE_PRINT_AGENT_URL` | `http://localhost:9123` | Print agent base URL. Frontend build-time only; leave unset for a multi-register shop so each till finds the agent on its own PC. See `docs/guides/print-agent-production.md` |

The defaults above are the **code fallbacks** used when a variable is unset. `.env.example` ships different values for the login and refresh limiters (`60/60` and `120/120`), so `cp .env.example .env` gives a much looser limiter than the table suggests — tune the values in `.env`, not just the code.

Copy `.env.example` to `.env` for local development.

### Deployment

#### Podman / Docker Compose (Recommended)

```bash
make build-all                       # Build backend + frontend images
./deploy/podman-deploy.sh start      # Start all services
./deploy/podman-deploy.sh status     # Check status
./deploy/podman-deploy.sh logs       # View logs
./deploy/podman-deploy.sh migrate    # Run migrations
./deploy/podman-deploy.sh seed       # Seed data
./deploy/podman-deploy.sh stop       # Stop all services
./deploy/podman-deploy.sh restart    # Restart
```

`web/dist` is gitignored and `deploy/frontend/Dockerfile` only `COPY`s it, so the
bundle has to be built on the host: run `(cd web && npm ci)` once after cloning.
From then on `podman-deploy.sh build` (and `make build-frontend`, which delegates
to it) rebuilds `web/dist` whenever anything under `web/src` is newer than the
last build. Do **not** call `podman build -f deploy/frontend/Dockerfile`
directly — it packages whatever `dist` happens to be on disk and reports success,
which silently ships the previous commit's JavaScript. `SKIP_FRONTEND_BUILD=1`
bypasses the check when a bundle was produced elsewhere.

Or use the Makefile: `make deploy`, `make stop`, `make restart`, `make status`, `make logs`, `make db-backup`, `make db-restore`, `make db-shell`, `make db-fresh` (reset the **dev** database to first-install state — destroys all data).

#### Auto-start on boot (systemd + Quadlet)

`podman generate systemd` was removed in Podman 5.0. Use the Quadlet units in
`deploy/quadlet/`:

```bash
./deploy/podman-deploy.sh build        # Quadlet starts containers, it does not build them

# The secret file is created with sudo, so it is root-owned; a rootless user
# service cannot read a root-owned 0600 file. Hand it to $USER, keep mode 600.
sudo chown "$USER" /etc/retail-pos/backend.env
sudo chmod 600 /etc/retail-pos/backend.env

loginctl enable-linger "$USER"         # else the stack waits for your next login

mkdir -p ~/.config/containers/systemd
cp deploy/quadlet/* ~/.config/containers/systemd/
systemctl --user daemon-reload

# Enable all three, postgres included — `start` alone does not survive a reboot.
systemctl --user enable --now retail-pos-postgres.service \
  retail-pos-backend.service retail-pos-frontend.service
```

Rootful hosts: copy to `/etc/containers/systemd/` and drop `--user`. See
`deploy/PRODUCTION-DEPLOYMENT.md`.

#### Database Migrations

Migrations are SQL files in `database/migrations/` (currently **`000_baseline.sql`** plus `054`–`060`). Migrations do **not** run automatically on server start — run them explicitly via `./deploy/podman-deploy.sh migrate` (or the test harness, which applies pending migrations to the test DB). Tracking of which migrations have been applied is in the `schema_migrations` table.

**Fresh database (new spin-up):** `migrate` bootstraps `pgcrypto`, `invoice_seq`, and the `schema_migrations` table first, then applies every migration in lexical order with `ON_ERROR_STOP=1`. Result: complete schema + reference data (roles, 91 permissions, 272 grants, 6 default users, payment methods, customer groups) plus the placeholder **Default Store**. Business data (products, customers, sales) must be seeded via `./seed-dev.sh` or `./deploy/podman-deploy.sh seed`.

> **Important:** Apply migrations **before** deploying a new server binary. Every migration is permanently re-runnable (`IF NOT EXISTS` / `DO` guards / `ON CONFLICT DO NOTHING`) because all three runners replay the whole directory on every run rather than trusting the ledger.

Current migrations:

| File | Purpose |
|------|---------|
| `000_baseline.sql` | Version 1 baseline: complete schema (60 tables, 3 materialised views, functions, indexes, constraints) plus reference data (6 roles, 91 permissions, 272 grants, the 6 system users, payment methods, customer groups, default store, walk-in customer, `app_settings`) |
| `054_pricing_rule_created_by.sql` | Adds nullable `pricing_rules.created_by` so a rule records its author and self-approval can be blocked |
| `055_pricing_rule_new_rows_start_pending.sql` | Realigns the `status`/`is_active` column defaults with the approval workflow — new rows start `pending`/`inactive` |
| `056_revoke_supervisor_pricing_mutation.sql` | Removes `pricing.create`/`update`/`delete` from supervisor on databases that already granted them (the baseline only INSERTs grants, so a removal needs an explicit `DELETE`) |
| `057_pricing_retire_draft_status.sql` | Retires the `draft` pricing status; remaps legacy rows to `pending` and hardens `chk_pricing_status` |
| `058_supplier_terms_store_scope.sql` | Moves the store boundary off `suppliers` onto `product_suppliers` (`UNIQUE NULLS NOT DISTINCT`), so negotiated terms are per store |
| `059_store_fk_integrity.sql` | Closes the last four `store_id` FK gaps (customers, users, goods_receipts, purchase_orders). Validates existing rows, so it hard-fails on an orphan — run `scripts/audit-store-fk-orphans.sh` first |
| `060_supplier_governance.sql` | Adds supplier provenance (`created_by`, `updated_by`) and optimistic concurrency (`version`), and narrows the global supplier-code uniqueness to live rows |

`000_baseline.sql` also clears the ledger rows of the 32 migrations it replaced, so after a fresh baseline install `schema_migrations` holds one row per migration file. Those 32 files (`000_squash.sql` + `001`–`007` + `031`–`053`) are preserved, unmodified, in `database/migrations/archive/pre-squash-migrations.tar.gz` — never execute an archived file against a live database. New migrations must start at `061_*.sql`.

### Default Credentials

| Role | Username | Password | Description |
|------|----------|----------|-------------|
| Superadmin | `superadmin` | `admin123` | All 90 granted permissions (everything except `sale.lookup`), including consignment.*, supplier.*, app_settings.*, audit.* |
| Manager | `manager` | `admin123` | Operational management: user CRUD (no delete), product/category/customer/pricing full CRUD, PO, stock opname, consignment view/create/update/settle/pay, store management, audit view+export (without user.delete, role.update/delete, app_settings.update, purchase_order.delete) |
| Supervisor | `supervisor` | `admin123` | Store operator: product/category/customer full CRUD, pricing, PO, stock opname, consignment view/create/update/settle, shifts, POS sales (sale.create) |
| Cashier | `cashier` | `admin123` | POS: create/view sales, park, shift, stock count, dashboard, category/pricing/customer_group view, Find Transaction lookup |
| Inventory Staff | `inventory_staff` | `admin123` | Stock ops: inventory.adjust, stock opname full lifecycle (create/assign/count/verify/post/close/export/report), storage location manage |
| Finance | `finance` | `admin123` | Records supplier payments and views financial reports (consignment view/pay, reporting) |

All six accounts are flagged by `000_baseline.sql` and must rotate `admin123` on first login — the backend answers HTTP 428 for every protected call until the password is changed, and the SPA blocks behind a *Change Your Password* dialog. Afterwards, change your own password anytime at `/account/password` (lock icon in the sidebar). (Default user password seeds previously lived in `database/seeds/`, which was retired; the default `admin123` users are created in `database/migrations/000_baseline.sql`.)

### Permission Matrix

Permissions use **dot-notation** (`entity.action`), e.g.: `user.view`, `product.create`, `stock_opname.post`. This table is the default configuration seeded by `000_baseline.sql`; it can be changed via the Role Management UI. Total **91 permissions**, 272 role grants (including `consignment.*`, `app_settings.*`, `supplier.*`, `sale.lookup`, `sale.detail`, `receipt.print`, `audit.export`, `shift.cash_movement`). `sale.lookup` is deliberately cashier-only — it is the one code superadmin does not hold.

| Permission | Superadmin | Manager | Supervisor | Cashier | Inventory Staff | Finance |
|------------|:---:|:---:|:---:|:---:|:---:|:---:|
| `dashboard.view` | ✅ | ✅ | ✅ | ✅ | – | ✅ |
| `product.view` | ✅ | ✅ | ✅ | ✅ | – | – |
| `product.create` | ✅ | ✅ | ✅ | – | – | – |
| `product.update` | ✅ | ✅ | ✅ | – | – | – |
| `product.delete` | ✅ | ✅ | – | – | – | – |
| `product.import`, `product.export` | ✅ | ✅ | – | – | – | – |
| `product.history.view` | ✅ | ✅ | – | – | – | – |
| `product.cost.view` | ✅ | ✅ | ✅ | – | – | – |
| `category.view` | ✅ | ✅ | ✅ | ✅ | – | – |
| `category.create` | ✅ | ✅ | ✅ | – | – | – |
| `category.update`, `category.delete` | ✅ | ✅ | ✅ | – | – | – |
| `category.import`, `category.export` | ✅ | ✅ | – | – | – | – |
| `sale.view` | ✅ | ✅ | ✅ | ✅ | – | ✅ |
| `sale.create`, `sale.park` | ✅ | ✅ | ✅ | ✅ | – | – |
| `sale.lookup` | – | – | – | ✅ | – | – |
| `sale.detail`, `receipt.print` | ✅ | ✅ | ✅ | ✅ | – | – |
| `shift.view`, `shift.create`, `shift.cash_movement` | ✅ | ✅ | ✅ | ✅ | – | – |
| `shift.review`, `shift.audit` | ✅ | ✅ | ✅ | – | – | – |
| `inventory.adjust` | ✅ | ✅ | ✅ | – | ✅ | – |
| `report.view` | ✅ | ✅ | ✅ | – | – | ✅ |
| `customer.view` | ✅ | ✅ | ✅ | ✅ | – | – |
| `customer.create`, `customer.update` | ✅ | ✅ | ✅ | – | – | – |
| `customer.delete`, `customer.import`, `customer.export` | ✅ | ✅ | ✅ | – | – | – |
| `customer_group.view` | ✅ | ✅ | ✅ | ✅ | – | – |
| `customer_group.create/update/delete` | ✅ | ✅ | ✅ | – | – | – |
| `store.view` | ✅ | ✅ | – | – | – | – |
| `store.create` | ✅ | – | – | – | – | – |
| `store.update`, `store.delete` | ✅ | ✅ | – | – | – | – |
| `storage_location.view` | ✅ | ✅ | ✅ | ✅ | ✅ | – |
| `storage_location.create/update/delete` | ✅ | ✅ | – | – | ✅ | – |
| `stock_opname.view` | ✅ | ✅ | ✅ | ✅ | ✅ | – |
| `stock_opname.count`, `stock_opname.submit` | ✅ | ✅ | ✅ | ✅ | ✅ | – |
| `stock_opname.create`, `stock_opname.assign` | ✅ | ✅ | ✅ | – | ✅ | – |
| `stock_opname.verify`, `stock_opname.post`, `stock_opname.close`, `stock_opname.report` | ✅ | ✅ | ✅ | – | ✅ | – |
| `stock_opname.cancel`, `stock_opname.export`, `stock_opname.recount` | ✅ | ✅ | ✅ | – | ✅ | – |
| `pricing.view` | ✅ | ✅ | ✅ | ✅ | – | – |
| `pricing.create`, `pricing.update` | ✅ | ✅ | – | – | – | – |
| `pricing.approve` | ✅ | ✅ | – | – | – | – |
| `pricing.delete` | ✅ | ✅ | – | – | – | – |
| `purchase_order.view/create/update/confirm/receive` | ✅ | ✅ | ✅ | – | – | – |
| `purchase_order.delete` | ✅ | – | – | – | – | – |
| `purchase_order.cancel` | ✅ | ✅ | ✅ | – | – | – |
| `consignment.view` | ✅ | ✅ | ✅ | – | – | ✅ |
| `consignment.create`, `consignment.update` | ✅ | ✅ | ✅ | – | – | – |
| `consignment.settle` | ✅ | ✅ | ✅ | – | – | – |
| `consignment.pay` | ✅ | ✅ | – | – | – | ✅ |
| `supplier.view` | ✅ | ✅ | ✅ | – | – | – |
| `supplier.create`, `supplier.update`, `supplier.delete` | ✅ | ✅ | – | – | – | – |
| `app_settings.view` | ✅ | ✅ | – | – | – | – |
| `app_settings.update` | ✅ | – | – | – | – | – |
| `user.view`, `user.create`, `user.update` | ✅ | ✅ | – | – | – | – |
| `user.delete` | ✅ | – | – | – | – | – |
| `role.view`, `role.create` | ✅ | ✅ | – | – | – | – |
| `role.update`, `role.delete` | ✅ | – | – | – | – | – |
| `audit.view` | ✅ | ✅ | – | – | – | ✅ |
| `audit.export` | ✅ | ✅ | – | – | – | – |

### Testing

#### Backend Tests

```bash
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos TEST_DB_PASSWORD=admin123 \
DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only \
go test -p 1 -count=1 ./...
```

#### Coverage (excluding `cmd/` and `tools/`)

```bash
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only \
go test -p 1 -count=1 -coverprofile=coverage.out $(go list ./... | grep -v -E '(retail-pos-system/cmd/|retail-pos-system/tools/)')
```

#### Frontend Unit Tests (Vitest)

```bash
cd web && npm run test:run
```

#### E2E Tests (Playwright)

```bash
npx playwright test --reporter=list
```

> Run from the repository root (where `playwright.config.js` is located). E2E requires the backend + frontend servers to be running (`./run-dev.sh` and `npm run dev`).

#### Test Database

Tests connect to the `retail_pos_test` database (configured via `TEST_DB_*` env vars). The test framework auto-applies pending migrations. If the test schema is out of sync: `dropdb retail_pos_test && createdb retail_pos_test`, then re-run the tests.

### Print Agent (Go)

Local print agent for the POS. Receives receipt payloads from the browser and
dispatches them to a printer as **ESC/POS** bytes. Dependency-free, single binary.

This replaces the previous Node agent (`tools/print-agent` historically shipped a
Node version). The frontend (`web/src/shared/services/print-service.ts`) talks to
it over `POST /print`.

#### Build & run

```bash
cd tools/print-agent
go build -o print-agent ./cmd/print-agent
PORT=9123 PRINT_TRANSPORT=file ./print-agent
```

A flag-driven launcher is also provided (`print-agent.sh`); it builds the binary
on first run (or with `-b`) and translates flags to env vars:

```bash
./print-agent.sh -t file -p 9123 -o /tmp/receipt-out      # file transport
./print-agent.sh -t tcp --tcp-addr 192.168.1.50:9100      # network printer
./print-agent.sh -t serial --serial-device /dev/ttyUSB0   # USB-serial printer
./print-agent.sh -t file -p 9123 --allowed-origins http://localhost:5173
```

Flags: `-t/--transport`, `-p/--port`, `-o/--output-dir`, `--tcp-addr`,
`--serial-device`, `--allowed-origins`, `-b/--build`, `-h/--help`.

> `--token` was **removed** and is now rejected with an explicit error. The only client is a
> browser, so a token shipped to the browser is not a secret. See
> `docs/guides/print-agent-production.md`.

In `file` mode (default, no hardware needed) receipts are written as ESC/POS
`.bin` files to `PRINT_OUTPUT_DIR` (default OS temp dir), e.g.
`/tmp/receipt-print-<jobid>.bin`. Inspect them to validate the renderer.

#### Endpoints

| Method | Path | Purpose |
| ------ | ---- | ------- |
| GET | `/health` | Agent + printer health |
| GET | `/printer` | Configured printer + connection status |
| POST | `/print` | Enqueue a print job (idempotent by `job_id`) |
| GET | `/print/jobs/{id}` | Job status |
| POST | `/print/jobs/{id}/retry` | Retry a failed job |

`POST /print` body (matches `print-service.ts`):

```json
{
  "invoice": "INV-1",
  "data": { "invoice_number": "INV-1", "items": [ ... ], "total_amount": 10000, ... },
  "branding": { "storeName": "My Store", "storeAddress": "...", "storePhone": "...", "receiptHeader": "...", "receiptFooter": "..." }
}
```

The agent returns `202 Accepted` with `{ "job_id", "status": "queued" }`. On
duplicate `job_id` it returns the existing job (no reprint).

#### Transports

| `PRINT_TRANSPORT` | Config | Use |
| ----------------- | ------ | --- |
| `file` (default) | `PRINT_OUTPUT_DIR` | Dev/CI — ESC/POS `.bin` output |
| `tcp` | `PRINT_TCP_ADDR=host:port` | Network thermal printer (port 9100) |
| `serial` | `PRINT_SERIAL_DEVICE=/dev/ttyUSB0` | USB-serial thermal printer |

> Real USB (vendor/product discovery via libusb) is a future enhancement; most
> 58mm "USB" printers expose a serial device node and work with `serial`.

#### Configuration

| Env | Default | Description |
| --- | --- | --- |
| `PORT` | `9123` | Listen port |
| `PRINT_TRANSPORT` | `file` | `file` \| `tcp` \| `serial` |
| `PRINT_OUTPUT_DIR` | OS temp | File transport output dir |
| `PRINT_TCP_ADDR` | — | `host:port` for tcp |
| `PRINT_SERIAL_DEVICE` | — | serial device path |
| `ALLOWED_ORIGINS` | `*` | comma-separated allowed CORS origins; required when `ENV=production` |

#### Security

Listens on all interfaces by default for `PORT`; restrict/forward as needed for
your deployment.

Access is controlled by the `ALLOWED_ORIGINS` check plus keeping the agent on a
trusted network. There is no bearer token: `PRINT_TOKEN` was supported and
removed, because a secret shipped to a browser is not a secret, it had no other
client, and enabling it made every print return 401. The browser-origin check
cannot be forged cross-origin, which is why it is the control rather than a
token. See `docs/guides/print-agent-production.md`.

#### Testing

```bash
cd tools/print-agent
go test ./...
```

---

## User Manual

The end-user manual is published separately so shop staff can read or print it
without the developer material: **[docs/guides/user-manual.md](docs/guides/user-manual.md)**.

It covers getting started, the dashboard, POS and shifts, products and
inventory, customers, suppliers, pricing rules, purchase orders, stock opname,
konsinyasi, reports, store management, administration, and import/export.

---

## License

Proprietary - Developed for retail business use.
