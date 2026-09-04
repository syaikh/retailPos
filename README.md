# Retail POS System

Retail POS System is a modern Point of Sale (POS) application for retail stores with inventory management, sales, purchase orders, pricing engine, reporting, shift management, and role-based access control.

> A note: "This is the single consolidated project reference. It contains both the Developer Guide and the End-User Manual."

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
- [Part E — User Guide](#part-e--user-guide)
  - [1. Getting Started](#1-getting-started)
  - [2. Dashboard](#2-dashboard)
  - [3. Point of Sale (POS)](#3-point-of-sale-pos)
  - [4. Transaction History](#4-transaction-history)
  - [5. Shifts](#5-shifts)
  - [6. Products & Inventory](#6-products--inventory)
  - [7. Categories, Brands & Units of Measure](#7-categories-brands--units-of-measure)
  - [8. Customers & Customer Groups](#8-customers--customer-groups)
  - [9. Suppliers](#9-suppliers)
  - [10. Storage Locations](#10-storage-locations)
  - [11. Pricing Rules](#11-pricing-rules)
  - [12. Purchase Orders](#12-purchase-orders)
  - [13. Stock Opname (Stock Count)](#13-stock-opname-stock-count)
  - [14. Konsinyasi Supplier](#14-konsinyasi-supplier)
  - [15. Reports](#15-reports)
  - [16. Store Management](#16-store-management)
  - [17. Administration](#17-administration)
  - [18. Import & Export](#18-import--export)
  - [Appendix A: Role / Permission Matrix](#appendix-a-role--permission-matrix)
  - [Appendix B: Status Reference](#appendix-b-status-reference)

---

## Part A — Project Overview & Features

## Features

- **Point of Sale (POS)** — Sales transactions with scanner, discounts, split payment (multi payment methods), and hold & recall (parked sales)
- **Purchase Order & Goods Receiving** — Purchasing workflow from suppliers: draft → confirmed → received, partial goods receiving, auto-generate PO/GR/DO numbers
- **Stock Opname** — Physical stock count sessions with 9-state workflow (draft → open → counting → verification → needs_recount → approved → posted → closed/cancelled), multi-scope sessions (store/warehouse/category/product), multi-counter assignment, blind count, recount workflow, adjustment ledger (IA- documents), and auto-adjustment of stock upon posting (FR-001 through FR-044)
- **Storage Locations** — Storage location master data (racks/warehouses) with warehouse/store scope, CRUD + bulk actions (phase 1 of per-rack stock tracking)
- **Store Management** — Store/outlet CRUD + store management UI page (list, active/inactive status)
- **Shift Management** — Cashier shift open/close, opening/closing balance, discrepancy review & audit
- **Pricing Engine** — Price rules (special price / promotion) by product, category, brand, customer group, and store; approval workflow (draft → pending → approved/rejected); real-time price resolver
- **Supplier Management** — Supplier CRUD, product-supplier links, preferred supplier, bulk actions, auto-generate codes (SUP-XXXXXX)
- **Konsinyasi Supplier (Consignment)** — Full consignment management: agreements, goods receiving, returns, settlements, payouts, consignment stock, POS checkout integration
- **Application Settings** — Global settings (store branding, jargon, logo) for superadmin only, receipt info per branch, per-user preferences (theme/light-dark, language)
- **Customer & Customer Groups** — Customer management, customer groups (Walk-in, Member, VIP), bulk actions
- **Multi-Warehouse & Multi-Store** — Inventory per warehouse/store with composite unique key, store management
- **Inventory Management** — Stock tracking, movement, low stock alerts, stock thresholds, multi-category filter
- **Import & Export Framework** — Schema-driven reusable import/export for Products, Categories, Customer Groups, Brands, UOMs, Customers, Pricing Rules, Suppliers, Stores with XLSX templates, preview, validation, reference dropdowns, import history (async job), and cancel
- **User Management** — RBAC (Role-Based Access Control) with dot-notation permissions, manager-subordinate hierarchy (org chart), soft delete
- **Audit Logging** — Full audit trail for all actions (including login/logout, import, change-password)
- **Real-time Dashboard** — Sales statistics, revenue, analytics + live updates via WebSocket, daily/weekly/monthly charts, period comparison, pricing breakdown
- **WebSocket Support** — Real-time notifications (dashboard live, PO updates)
- **Swagger/OpenAPI** — API documentation via swaggo annotations
- **Structured Logging** — JSON (production) / text (development) via `log/slog`
- **EventBus Observability** — Atomic metrics for published/consumed/failed events
- **Dead-Letter Queue** — Failed events stored to PostgreSQL for retry
- **Materialized Views** — Pre-aggregated daily/hourly sales data for fast reporting queries; refresh coordinated by `report.RefreshCoordinator` (debounced, default 30s after `sale.created`, coalescing) plus hourly ticker in `cmd/server/main.go`

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
│  Svelte 5 + Tailwind CSS 4 + Vite 6 (HMR)               │
└──────────────┬───────────────────────────────────────────┘
               │ /api/* → Backend, /ws/* → WebSocket
┌──────────────┴───────────────────────────────────────────┐
│  Go Backend (Gin)        http://localhost:9095           │
│  PostgreSQL              localhost:5433 (postgres-dev)   │
│  `./run-dev.sh` rebuild + restart automatically (press r)     │
└──────────────────────────────────────────────────────────┘
```

### Production (Podman / Docker Compose)

```
┌──────────────────────────────────────────────────────────┐
│  Nginx Frontend            Port 80 / 443                │
│  Go Backend                Port 8080 (internal)         │
│  PostgreSQL 18             Volume retail-pos-postgres-data│
│  Network: retail-pos-network                              │
│  `./deploy/podman-deploy.sh start`                       │
└──────────────────────────────────────────────────────────┘
```

**Tech Stack:**
- **Backend:** Go (Gin), PostgreSQL 18 (pgx), JWT Auth, WebSocket (gorilla/websocket), structured logging (slog)
- **Frontend:** Svelte 5, Tailwind CSS 4, Vite 6, Chart.js, jsPDF, Playwright, Vitest
- **Infrastructure:** Podman (rootless), Docker Compose, Nginx, systemd

---

## Part C — Quick Start

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
./deploy/podman-deploy.sh migrate # Run migrations (optional, automatic at startup)
./deploy/podman-deploy.sh seed    # Seed initial data (optional)
```

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
| `database/migrations/000_squash.sql` | Baseline schema (role, user, product, sale, inventory, etc.) |
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
swag init -g cmd/server/main.go -o docs/swagger
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
| POST | `/sales/parked` | Park (hold) transaction | `sale.park` |
| GET | `/sales/parked` | Parked sales list | `sale.park` |
| GET | `/sales/parked/:id` | Parked sale detail | `sale.park` |
| POST | `/sales/parked/:id/recall` | Recall parked sale | `sale.park` |
| POST | `/sales/parked/:id/complete` | Complete parked sale (checkout) | `sale.park` |
| DELETE | `/sales/parked/:id` | Cancel parked sale | `sale.park` |
| GET | `/payment-methods` | Payment method list | Public |
| GET | `/payment-methods/:code` | Payment method detail | Yes |

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
| POST | `/pricing-rules/:id/submit` | Submit for approval | `pricing.update` |
| POST | `/pricing-rules/:id/approve` | Approve rule | `pricing.update` |
| POST | `/pricing-rules/:id/reject` | Reject rule | `pricing.update` |
| POST | `/pricing/resolve` | Resolve final price | `pricing.view` |
| GET | `/products/search` | Search products (for pricing) | `pricing.view` |
| GET | `/suppliers` | Supplier list | `pricing.view` |
| GET | `/suppliers/:id` | Supplier detail | `pricing.view` |
| POST | `/suppliers` | Create supplier | `pricing.create` |
| PUT | `/suppliers/:id` | Update supplier | `pricing.update` |
| DELETE | `/suppliers/:id` | Delete supplier | `pricing.delete` |
| PUT | `/suppliers/bulk` | Bulk update | `pricing.update` |
| DELETE | `/suppliers/bulk` | Bulk delete | `pricing.delete` |
| GET | `/suppliers/:id/products` | Products from supplier | `pricing.view` |
| POST | `/suppliers/:id/products` | Link product to supplier | `pricing.update` |
| DELETE | `/suppliers/:id/products/:productId` | Unlink product | `pricing.update` |
| PUT | `/suppliers/:id/products/:productId` | Update relation (unit_cost) | `pricing.update` |
| POST | `/suppliers/:id/products/:productId/preferred` | Set preferred supplier | `pricing.update` |
| GET | `/products/:id/suppliers` | Suppliers for product | `pricing.view` |

##### Consignment (Konsinyasi Supplier)

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/consignment/suppliers` | Consignment supplier list | `consignment.view` |
| GET | `/consignment/arrangements` | Consignment arrangement list | `consignment.view` |
| POST | `/consignment/arrangements` | Create arrangement | `consignment.create` |
| GET | `/consignment/arrangements/:id` | Arrangement detail | `consignment.view` |
| PUT | `/consignment/arrangements/:id/terms` | Update terms/conditions | `consignment.update` |
| GET | `/consignment/receipts` | Goods receipt list | `consignment.view` |
| POST | `/consignment/receipts` | Create goods receipt | `consignment.create` |
| GET | `/consignment/receipts/:id` | Receipt detail | `consignment.view` |
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
| GET | `/consignment/payment-methods` | Consignment payment methods | `consignment.settle` |
| POST | `/consignment/settlements/:id/payouts` | Create payment to supplier | `consignment.pay` |

##### Shifts

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| POST | `/shifts/open` | Open shift | `shift.create` |
| POST | `/shifts/:id/close` | Close shift | `shift.create` |
| POST | `/shifts/close-all` | Close all active shifts | `shift.create` |
| POST | `/shifts/:id/review` | Review shift discrepancy | `shift.review` |
| POST | `/shifts/:id/audit` | Physical cash audit | `shift.audit` |
| GET | `/shifts/active` | Current active shift | Yes |
| GET | `/shifts` | Shift list | `shift.view` |
| GET | `/shifts/export` | Export shifts | `shift.view` |
| GET | `/shifts/:id` | Shift detail | `shift.view` |

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
| GET | `/audit-logs/export` | Export audit logs | `audit.view` |
| GET | `/audit-logs/entity-types` | Entity type list | `audit.view` |

##### Import & Export

Supported modules: `products`, `categories`, `brands`, `uoms`, `customers`, `pricing_rules`, `suppliers`, `stores`.

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
| GET | `/settings` | All settings | `app_settings.view` |
| PUT | `/settings` | Update settings | `app_settings.update` |
| POST | `/settings/logo` | Upload logo | `app_settings.update` |
| DELETE | `/settings/logo` | Delete logo | `app_settings.update` |

##### System

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/health` | Health check | No |
| GET | `/ws` | WebSocket hub | Yes |
| GET | `/swagger/*any` | Swagger UI | No |

### Frontend

#### Development

```bash
cd web
npm run dev       # Start dev server (port 5173)
npm run build     # Build for production
npm run test:run  # Unit test (Vitest)
npx playwright test  # Run E2E tests
```

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
| `ENV` | `development` | `development` (text log) / `production` (JSON log, release mode, sslmode require) |
| `LOG_LEVEL` | `debug`/`info` | Log level: debug, info, warn, error |
| `CORS_ORIGIN` | `http://localhost:5173` | Allowed CORS origin (must not be `*` in production) |
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
| `STOCK_WARNING_THRESHOLD` | `10` | Stock below this = "needs attention" |
| `STOCK_CRITICAL_THRESHOLD` | `5` | Stock below this = "low stock" |
| `CART_HOLD_TTL_HOURS` | `24` | How many hours a cart session is held before considered expired |
| `REPORT_REFRESH_DEBOUNCE` | `30` | Seconds debounce for materialized view refresh after `sale.created` |

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

Or use the Makefile: `make deploy`, `make stop`, `make restart`, `make status`, `make logs`, `make db-backup`, `make db-restore`, `make db-shell`.

#### Systemd

```bash
sudo cp deploy/retail-pos.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now retail-pos
```

#### Database Migrations

Migrations are SQL files in `database/migrations/` (currently **`000_squash.sql`, `001`–`007`, `031`–`032`**). Migrations do **not** run automatically on server start — run them explicitly via `./deploy/podman-deploy.sh migrate` (or the test harness, which applies pending migrations to the test DB). Tracking of which migrations have been applied is in the `schema_migrations` table.

**Fresh database (new spin-up):** `migrate` bootstraps `pgcrypto`, `invoice_seq`, and the `schema_migrations` table first, then applies each file sequentially from `000_squash.sql` with `ON_ERROR_STOP=1`. Result: complete schema + reference data (roles, 85 permissions, grants, 5 default users, payment methods, customer groups). Business data (stores, products, customers, sales) must be seeded via `./seed-dev.sh` or `./deploy/podman-deploy.sh seed`.

> **Important:** Apply migrations **before** deploying a new server binary. Migrations are idempotent (`IF NOT EXISTS` / `ON CONFLICT DO NOTHING`) and must be run sequentially from `000_squash.sql`. Current migrations: `001`–`007`, `031`–`039` — see AGENTS.md for deployment ordering details.

Current migrations:
- `000_squash.sql` — Baseline schema + initial seed data (roles, 85 permissions, grants, 5 default users, payment methods, customer groups, tombstone sequences)
- `001_consignment.sql` — Consignment supplier: `consignment_*` tables, sequence, `consignment.*` permissions
- `002_settlement_items_product_id.sql` — `consignment_settlement_items.product_id` column (FK to products, NULL-able)
- `003_settlement_updated_at.sql` — `consignment_settlements.updated_at` column
- `004_supplier_code_sequence.sql` — `supplier_seq` sequence for auto-generating supplier codes (`SUP-%06d`)
- `005_app_settings.sql` — `app_settings` table (global key-value: branding, receipt text), seed defaults, `app_settings.view`/`app_settings.update` permissions
- `006_user_preferences.sql` — `users.language` and `users.theme` columns for per-user preferences
- `007_sale_lookup.sql` — `sale.lookup` permission + grant to `cashier` (grant to `manager` later revoked by `031`)
- `031_revoke_sale_lookup_manager.sql` — Revoke `sale.lookup` grant from `manager` role
- `032_sale_detail_and_receipt_print.sql` — `sale.detail` and `receipt.print` permissions; grant to `cashier`, `manager`, `admin`, `superadmin`
- `033_*` — Audit log store FK + immutability trigger + cash_change table
- `034_audit_immutable_bypass.sql` — GUC-aware bypass for audit immutability trigger
- `035_audit_correlation_id.sql` — `audit_logs.correlation_id` column
- `036_audit_export_permission.sql` — `audit.export` permission; grant to `superadmin`, `admin`
- `037_audit_immutable_fk_bypass.sql` — Allow FK-cascade updates through append-only trigger
- `038_grant_audit_view_to_admin.sql` — Grant `audit.view` to `admin` (fix: admin had export but not view)
- `039_business_permission_audit.sql` — Business-perspective audit: +12 manager, +4 cashier, +1 staff permissions

### Default Credentials

| Role | Username | Password | Description |
|------|----------|----------|-------------|
| Superadmin | `superadmin` | `admin123` | All permissions (84 including consignment.*, app_settings.*, audit.*) |
| Admin | `admin` | `admin123` | Operational management: user CRUD, product/category/customer/pricing full CRUD, PO, stock opname, audit view+export (without user.delete, role.update/delete, app_settings.update, purchase_order.delete) |
| Manager | `manager` | `admin123` | Store operator: product/category/customer full CRUD, pricing, PO, stock opname, consignment view/create/update/settle, shifts |
| Cashier | `cashier` | `admin123` | POS: create/view sales, park, shift, stock count, dashboard, category/pricing/customer_group view, Find Transaction lookup |
| Staff | `staff` | `admin123` | View-only: product + stock opname counting + category view |

Change password in production via the UI change-password. (Default user password seeds previously lived in `database/seeds/`, which was retired; the default `admin123` users are created in `database/migrations/000_squash.sql`.)

### Permission Matrix

Permissions use **dot-notation** (`entity.action`), e.g.: `user.view`, `product.create`, `stock_opname.post`. This table is the default configuration from seeds; it can be changed via the Role Management UI. Total 85 permissions (including `consignment.*`, `app_settings.*`, `sale.lookup`, `sale.detail`, `receipt.print`, `audit.export`).

| Permission | Superadmin | Admin | Manager | Cashier | Staff |
|------------|:---:|:---:|:---:|:---:|:---:|
| `dashboard.view` | ✅ | ✅ | ✅ | ✅ | – |
| `product.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `product.create` | ✅ | ✅ | ✅ | – | – |
| `product.update` | ✅ | ✅ | ✅ | – | – |
| `product.delete` | ✅ | ✅ | – | – | – |
| `product.import`, `product.export` | ✅ | ✅ | – | – | – |
| `product.history.view` | ✅ | ✅ | – | – | – |
| `product.cost.view` | ✅ | ✅ | ✅ | – | – |
| `category.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `category.create` | ✅ | ✅ | ✅ | – | – |
| `category.update`, `category.delete` | ✅ | ✅ | ✅ | – | – |
| `category.import`, `category.export` | ✅ | ✅ | – | – | – |
| `sale.view` | ✅ | ✅ | ✅ | ✅ | – |
| `sale.create`, `sale.park` | ✅ | ✅ | – | ✅ | – |
| `sale.lookup` | – | – | – | ✅ | – |
| `sale.detail`, `receipt.print` | ✅ | ✅ | ✅ | ✅ | – |
| `shift.view`, `shift.create` | ✅ | ✅ | ✅ | ✅ | – |
| `shift.review`, `shift.audit` | ✅ | ✅ | ✅ | – | – |
| `inventory.adjust` | ✅ | ✅ | ✅ | – | – |
| `report.view` | ✅ | ✅ | ✅ | – | – |
| `customer.view` | ✅ | ✅ | ✅ | ✅ | – |
| `customer.create`, `customer.update` | ✅ | ✅ | ✅ | – | – |
| `customer.delete` | ✅ | ✅ | ✅ | – | – |
| `customer.import`, `customer.export` | ✅ | ✅ | ✅ | – | – |
| `customer_group.view` | ✅ | ✅ | ✅ | ✅ | – |
| `customer_group.create/update/delete` | ✅ | ✅ | ✅ | – | – |
| `store.view` | ✅ | ✅ | – | – | – |
| `store.create/update/delete` | ✅ | ✅ | – | – | – |
| `storage_location.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `storage_location.create/update/delete` | ✅ | ✅ | – | – | – |
| `stock_opname.view` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `stock_opname.count`, `stock_opname.submit` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `stock_opname.create`, `stock_opname.assign` | ✅ | ✅ | ✅ | – | – |
| `stock_opname.verify`, `stock_opname.post`, `stock_opname.close`, `stock_opname.report` | ✅ | ✅ | ✅ | – | – |
| `stock_opname.cancel`, `stock_opname.export`, `stock_opname.recount` | ✅ | ✅ | ✅ | – | – |
| `pricing.view` | ✅ | ✅ | ✅ | ✅ | – |
| `pricing.create`, `pricing.update` | ✅ | ✅ | ✅ | – | – |
| `pricing.delete` | ✅ | ✅ | ✅ | – | – |
| `purchase_order.view/create/update/confirm/receive` | ✅ | ✅ | ✅ | – | – |
| `purchase_order.delete` | ✅ | – | – | – | – |
| `purchase_order.cancel` | ✅ | ✅ | ✅ | – | – |
| `consignment.view` | ✅ | ✅ | ✅ | – | – |
| `consignment.create`, `consignment.update` | ✅ | ✅ | ✅ | – | – |
| `consignment.settle` | ✅ | ✅ | ✅ | – | – |
| `consignment.pay` | ✅ | ✅ | – | – | – |
| `app_settings.view` | ✅ | ✅ | – | – | – |
| `app_settings.update` | ✅ | – | – | – | – |
| `user.view`, `user.create`, `user.update` | ✅ | ✅ | – | – | – |
| `user.delete` | ✅ | – | – | – | – |
| `role.view`, `role.create` | ✅ | ✅ | – | – | – |
| `role.update`, `role.delete` | ✅ | – | – | – | – |
| `audit.view`, `audit.export` | ✅ | ✅ | – | – | – |

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
./print-agent.sh -t file -p 9123 --token s3cret --allowed-origins http://localhost:5173
```

Flags: `-t/--transport`, `-p/--port`, `-o/--output-dir`, `--tcp-addr`,
`--serial-device`, `--token`, `--allowed-origins`, `-b/--build`, `-h/--help`.

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
| `PRINT_TOKEN` | — | optional bearer token (localhost CORS is the main control) |
| `ALLOWED_ORIGINS` | `*` | comma-separated allowed CORS origins |

#### Security

Listens on all interfaces by default for `PORT`; restrict/forward as needed for
your deployment. For localhost use, the browser-origin CORS check is the primary
control; `PRINT_TOKEN` is optional hardening (it would ship in the frontend
bundle, so treat it as obfuscation, not real auth).

#### Testing

```bash
cd tools/print-agent
go test ./...
```

---

## Part E — User Guide

### Table of Contents

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
    - [Roles & Permissions](#roles--permissions-1)
    - [Audit Logs](#audit-logs)
18. [Import & Export](#18-import--export)
19. [Appendix A: Role / Permission Matrix](#appendix-a-role--permission-matrix)
20. [Appendix B: Status Reference](#appendix-b-status-reference)

---

### 1. Getting Started

#### Roles and Permissions

The system has five built-in roles. Your role determines which menus you see and which actions you can take.

| Role | Typical user | What they do |
|------|--------------|--------------|
| **Superadmin** | System owner | Everything, including user deletion, role management, and audit logs |
| **Admin** | Store administrator | Everything except deleting users/roles and audit logs (can create and edit users and roles) |
| **Manager** | Store/ops manager | Dashboard, transactions, reports, shifts, purchase orders, stock opname; manages products, categories, customers, pricing rules, suppliers; adjusts inventory; no POS register |
| **Cashier** | Front-line seller | POS, own transactions, own shifts, customer lookup, stock counting |
| **Staff** | Warehouse/counter staff | Products (view), stock opname counting |

A complete permission-to-role matrix is in [Appendix A](#appendix-a-role--permission-matrix).

#### Logging In

1. Open the Retail POS application in your browser.
2. On the login page, enter your **Username** and **Password**.
3. Click **Login** (or press Enter).

After a successful login you are taken to the screen appropriate for your role:

- **Cashier** → the **Shifts** page (you must open a shift before using the POS).
- **Staff** → the **Products** page.
- **Superadmin / Admin / Manager** → the **Dashboard**.

> Your session is active for the current browser tab only. If you close the browser, you will need to log in again.

#### The Main Screen (Navigation)

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

#### Notifications

The bell shows live notifications as events happen, including:
- **Low stock alerts** — products below the critical threshold.
- **New sales** — when a transaction is completed.
- **Purchase order received** — when goods are received.
- **Stock opname events** — created / submitted / approved / needs recount / cancelled (requires `stock_opname.view`).

Clicking a notification jumps to the relevant page (e.g. the stock opname session, the transaction, or the product list filtered to low stock).

#### Logging Out

1. Click **Logout** at the bottom of the sidebar.

> **Cashier note:** you cannot log out while a shift is open. Close your shift first (the Logout button shows the tooltip *"Close shift first"*).

#### User Preferences

Each user can personalise their experience via the profile menu (click your username at the bottom of the sidebar):

- **Theme** — Switch between **Light** and **Dark** mode. The setting is saved per user and applied on every login.
- **Language** — Choose the display language. The setting is saved per user and applied on every login.

> These preferences are stored server-side and follow you across devices.

#### Application Settings (Superadmin Only)

Superadmins can configure global branding under **Administration → Settings**:

- **Store Name** — displayed in the sidebar, login page, and receipts.
- **Store Jargon** — a subtitle/tagline (e.g. "Management System").
- **Logo** — uploaded image shown on receipts and the login page.
- **Receipt Header** — custom text printed at the top of receipts.
- **Receipt Footer** — custom text printed at the bottom of receipts (default: "Terima kasih atas kunjungan Anda!").

Admins can view these settings but only superadmins can edit them.

---

### 2. Dashboard

The Dashboard gives you a live summary of the day:

- **Today's Revenue** — total revenue so far today (updates live as sales are completed).
- **Transactions** — how many sales were completed today.
- **Total Products** — the number of units in the catalog.
- **Low Stock Alerts** — products that need attention ("Action required" or "All stock healthy").

Below the cards are **Quick Access** tiles that jump to Point of Sale, Inventory, Reports, and Administration.

---

### 3. Point of Sale (POS)

The POS is the cash register screen. It has two areas: a **product search panel** on the left and a **cart** on the right. On mobile the cart becomes a bottom sheet that you can show or hide.

#### Before You Start: Open a Shift

If you are a **cashier**, you must open a shift first:

1. When you reach the POS without an open shift, you'll see *"Anda harus membuka shift terlebih dahulu"* (You must open a shift first) and be redirected to **Shifts**.
2. Click **Open Shift** and enter your **opening balance** — the amount of cash in the drawer at the start of the shift.
3. Confirm. You are taken to the POS.

The opening balance is used later to reconcile the drawer when you close the shift.

#### Adding Items to the Cart

1. Press **F2** (or click) to focus the search box.
2. Type the product **name, SKU, or barcode**. Results update as you type.
3. **Enter** adds the first matching product to the cart. Alternatively, use **Arrow Up / Arrow Down** to highlight a product and press **Enter**, or click a row then the **Add** button (double-click a row also adds it).

The table shows **Product name / Stock / Price / Add**. The stock shown is the available quantity *after* subtracting what is already in your cart. Products with no stock have a disabled Add button. A colored stock badge tells you at a glance:

- Red `0` — out of stock
- Red — at or below the critical threshold
- Amber — at or below the warning threshold
- Green — healthy

When a product has an active pricing rule, the cart shows the discounted price in green with the original price struck through, and the name of the applied rule. Items whose price was frozen for an in-progress transaction show *"harga dibekukan"* (price frozen).

#### Editing the Cart

- Use the **+ / −** buttons, or type a quantity in the box (limited to available stock).
- Click the **X** to remove an item.
- Press **ALT+Delete** to clear the entire cart.
- Press **F6** to hold (park) the sale for later.

The cart footer shows the **Subtotal (DPP)**, **PPN 11%** (when applicable), and **Total** above the pay button. Discounts are applied automatically by Pricing Rules — there is no manual discount entry at the register.

#### Holding and Recalling Sales

You can park a sale and resume it later — stock is **not** reduced while held.

- **Hold:** press **F6** (or click Hold). The cart is saved and the toast *"Sale held"* appears.
- **Recall:** press **F5** (or click Recall) to open the **Held Sales** list. Each entry shows `Cart #id`, the total, and the item count. Click **Recall** on one to restore it to the cart (*"Sale resumed"*).

#### Checkout & Payment

1. Press **F4** or click the green **Bayar [F4]** (Pay) button. The **Pembayaran** (Payment) modal opens with a default **CASH** row of Rp 0.
2. Click the payment-method buttons to add an **Alokasi Pembayaran** (Payment Allocation) row for each method. The available methods are **Cash (CASH)**, **Card (CARD)**, **E-Wallet (E_WALLET)**, **Transfer (TRANSFER)**, and **QRIS**.
   - For non-cash methods a **No. Referensi** (Reference Number) field is pre-filled; you can edit it.
   - For cash, use the quick buttons **5rb / 10rb / 20rb / 50rb / 100rb** to add denominations, or press **F7 (Tepat)** (Exact) to set cash to exactly the total.
   - Use **Reset** to zero a row and **Hapus semua** (Clear All) to remove all allocations.
3. Split payment is supported — add multiple allocations as long as they sum to the total.
4. Press **Enter** or click **Selesai** (Done) to complete the sale. This button is only enabled when the allocations equal the total.
5. Press **Esc** (or click **Batal** (Cancel)) to cancel the checkout and return to the cart.

On success you'll see *"Sale completed"*, the cart clears, and a receipt is printed automatically according to the selected **print mode** (see below).

#### Customer Selection

By default the sale is to **Walk-in / General**. To attach a customer:

1. Click the customer row in the checkout modal.
2. In the **Pilih Customer** (Select Customer) dialog, search by **name or phone**, or choose **Walk-in / Umum** (Walk-in / General).
3. Selecting a customer applies that customer's group pricing to the items in the cart.

#### Reprinting a Receipt

After a sale, the cart footer shows **Print · {invoice number}**. Click it to reprint the last sale's receipt.

#### Receipt Printing Modes

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

#### Setting up a register

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

#### Keyboard Shortcuts

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

### 4. Transaction History

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
- Totals: **Hemat** (Savings), **Subtotal (DPP)**, **PPN 11%**, and **TOTAL**.
- Actions: **Print Receipt** (thermal receipt) and **Download Invoice** (a PDF invoice).

**Exporting**
Use the Export dropdown to download **CSV** or **Excel** of the filtered results (`transactions-YYYY-MM-DD`).

> Note: there is currently no void/refund feature in the system. A transaction's status is `completed` and purchases cannot be returned through the app.

---

### 5. Shifts

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

### 6. Products & Inventory

The **Products** page (Inventory → Products, or Master Data → Products) is your product catalog and your stock-level screen.

#### Browsing Products

- **Search** — by name, SKU, or barcode.
- **Kategori** (Category) — filter by one or more categories.
- **Status** — All / Active / Inactive / Archived.
- **Low Stock** toggle — show only products at or below the critical threshold.
- **Supplier** — filter to a specific supplier's products (arrived via the Suppliers page).
- Sortable columns and pagination (20 per page).

Active filter chips appear below the toolbar; use the **X** on a chip or **Clear all** to reset.

#### Adding / Editing a Product

Click **Add Product** (superadmin/admin only) and fill in:

- **Name** (required), **SKU** (required), **Barcode** (optional)
- **Category** (required) — type to search existing categories
- **Brand**, **Unit**, **Tax Class** (e.g. PPN 11%)
- **Price (IDR)** (required), **Cost (IDR)**, **Stock** (required)
- **Description** (optional)
- **Status** — Draft / Active / Inactive / Discontinued (Archived is available to admin/superadmin)

On **edit**, a **Pricing Rules** panel lists the rules currently attached to the product (inactive rules are dimmed).

**Deleting a product** is permanent and removes it from the catalog — only admin/superadmin can do it.

#### Adjusting Stock

To change the on-hand quantity of a product:

1. On the product row, open the **Adjust Stock** action.
2. Enter a **Quantity Change** — positive adds stock, negative reduces it.
3. Enter **Notes** — a reason is required (e.g. "damaged", "return", "found on shelf").
4. Click **Adjust Stock**.

This records an inventory adjustment; the note is kept as the reason.

> Stock is also changed automatically when a sale completes (reduced), when a purchase order is received (increased), and when a stock opname is posted.

#### Rack Stock (Stok Rak)

Opening a product's **detail drawer** shows a **Stok Rak (Lokasi)** (Rack Stock (Location)) panel listing how much of the product sits in each storage location (rack/shelf). Rack rows are a *sub-account* of the global stock — set/transfer operations never change the global stock number.

- **Tambah Stok / Set** (Add Stock / Set) — records the exact quantity of the product in a chosen location (upsert; overwrites the current rack figure).
- **Transfer** — moves a quantity from one location to another (requires the source to have enough stock).

Rack stock is reconciled automatically when a **stock opname scoped to a storage location** is posted: the rack row is corrected to the physical count, and the global stock is recomputed from that count (see §13), so a rack count reconciles the sub-account with the global number.

#### Low Stock Alerts

Thresholds are configured system-wide (defaults: warning 10, critical 5). Products at or below the critical level are highlighted in red and trigger a dashboard alert and a notification.

#### Bulk Actions

Tick the checkboxes on rows to select products, then use the bulk bar to:
- **Change Status** — set selected products to Active / Inactive / Archived.
- **Export / Import** — see [Import & Export](#17-import--export).

---

### 7. Categories, Brands & Units of Measure

These master-data pages live under Master Data (Categories is also under Administration).

**Categories** (`/categories`)
- Create/edit/delete categories; the list shows each category's product count.
- Products reference categories by name.

**Brands** (`/brands`)
- Create/edit/delete brands (name, etc.), with export/import and an import history.

**Units of Measure** (`/units-of-measure`)
- Create/edit units (code, name, description, active). Units are used on product records (e.g. pcs, box, kg).

---

### 8. Customers & Customer Groups

#### Customers (`/customers`)

**Search & filter:** by name, phone, or email; status (All/Active/Inactive); and customer group.

**Creating a customer**
Click **Add Customer** and fill in:
- **Name** (required), **Phone** (required, 7–20 digits/format), **Email** (required)
- Optional: **Address**, **Customer Group**, **Note**

**Editing** — change details and toggle the **Active** checkbox.

**Deactivating** — use the trash icon; the customer is hidden from active listings but their history is preserved. (Reactivation is done via the edit form's Active checkbox.)

**Bulk actions** — change status (Active/Inactive) or delete selected customers (history preserved).

There is no credit/balance feature; customers are used mainly to attach group pricing and to record who bought what.

#### Customer Groups (`/customer-groups`)

Groups allow you to apply group-specific pricing at the POS.

**Create:** **Tambah Group** (Add Group) → **Group Name** (required), **Description**, and an avatar **color**.
**Edit / Delete / Duplicate:** via the row's kebab menu (Duplicate pre-fills the name as `{name} (Salinan)` (Copy)).
**View members:** kebab menu → **Lihat Anggota** (View Members) jumps to the Customers page filtered to that group; a **Kembali** (Back) banner returns to the group list.

Clicking a row opens a drawer with group details and an **activity history** (created/updated/deleted by whom and when).

At the POS, when a customer in a group is attached to a cart, the group's pricing rules apply automatically.

---

### 9. Suppliers

The **Suppliers** page manages the vendors you purchase from.

**Create:** **Add Supplier** → **Supplier Name** (required) and **Supplier Code** (required), plus optional contact person, phone, email, address, and notes.

**Edit:** change details and toggle **Active**.

**View:** the detail drawer shows supplier info and their products. A **products** link filters the Products page to this supplier (with a **Kembali ke Suppliers** (Back to Suppliers) banner).

**Consignment filter:** click **Konsinyasi** in the toolbar to show only suppliers flagged as consignment suppliers. This filter stacks with the Active/Inactive status filter. The URL `/suppliers?is_consignment=true` deep-links to this filtered view. When navigating from the Consignment module's **View Suppliers** link, a back arrow appears above the toolbar to return to the arrangements list.

Suppliers are used by Purchase Orders — when creating a PO you pick a supplier and choose only products linked to that supplier.

---

### 10. Storage Locations

**Storage Locations** (`/storage-locations`, Indonesian UI) are master data for where products are physically kept (racks/shelves), scoped to a **warehouse** or a **store**.

- **Search** by code or name; filter **Semua / Aktif / Nonaktif** (All / Active / Inactive).
- **Tambah Lokasi** (Add Location) → **Kode** (Code, required, e.g. `RAK-A-01`) and **Nama** (Name, required, e.g. `Rak A - Baris 1`), a scope (**Gudang**/Warehouse or **Toko**/Store), and optional **Catatan** (Notes).
- **Edit** and **Delete** via the row's action menu; bulk **Aktifkan / Nonaktifkan / Hapus** (Activate / Deactivate / Delete).

This is master data only for the location itself. Rack-level stock tracking is live — see **Rack Stock (Stok Rak)** in §6 — and rack-aware stock counts are available via the **Storage Location (Rack)** scope in §13.

---

### 11. Pricing Rules

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

#### Creating a Pricing Rule

Click **Tambah Rule** (Add Rule) and complete the five-step form:

1. **Informasi Rule** (Rule Information) — Name (required), Price Type, Method, and Value.
2. **Kondisi** (Conditions) — minimum/maximum quantity (empty = unlimited), customer group (All Groups), outlet (All Outlets).
3. **Target** — choose products, categories, and/or brands (at least one target is required; leave unused fields empty).
4. **Jadwal** (Schedule) — All Days / Weekdays / Weekend, active hours (Dari Jam–Sampai Jam) (From Hour – To Hour), and validity dates (empty = always).
5. **Ringkasan Rule** (Rule Summary) — a live 12-row preview of the rule.

You can tick **"Boleh digabung (stacking)"** (Allow stacking) to allow the rule to combine with other rules.

On save, the system checks for **conflicts** with existing rules. If a conflict is found, a **Konflik Ditemukan** (Conflict Found) warning lists the conflicting rules and lets you choose **Tetap Simpan** (Save Anyway) or go back.

#### Approval Workflow

Rules move through an approval workflow:

```
Draft → Pending → Approved
               ↘ Rejected
```

- **Ajukan** (Submit) — moves a draft to pending.
- **Approve** — approves a pending rule (making it active).
- **Reject** — rejects a pending rule (back to draft).

You can also **Edit**, **Duplikasi** (Duplicate), **Hapus** (Delete), and **Aktifkan/Nonaktifkan** (Enable/Disable) rules, and use the bulk bar.

**Filters:** search, All/Aktif/Nonaktif (All/Active/Inactive), approval status (Semua Approval) (All Approval), rule type, and method.

#### Simulating a Price

The **Simulasi** (Simulation) tool answers "what will this cost?":

1. Click **Simulasi** (Simulate).
2. Select a **product** (type at least 2 characters), **Jumlah** (Quantity), **Customer Group**, and **Toko** (Store).
3. Click **Hitung** (Calculate).
4. The result shows the original price, the final price, and the rule applied (discounted, markup, or normal).

The Product edit form also shows the rules attached to each product.

---

### 12. Purchase Orders

The **Purchase Orders** page manages orders to suppliers. Statuses: **Draft → Confirmed → Partial Received → Fully Received**, or **Cancelled** (the backend also supports `waiting_approval`/`rejected` for the approval workflow).

#### Creating a Purchase Order

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

#### Confirming a Purchase Order

While a PO is in **Draft** you can **Edit** it, **Confirm** it, or **Cancel** it. Confirming locks the order and makes it ready for receiving.

#### Receiving Goods

When goods arrive (PO status **Confirmed** or **Partial Received**), click **Receive**:

1. The **Receive Goods** modal lists each item with **Ordered**, **Remaining** (= ordered − received), and fields for **Qty Good** and **Qty Damaged**.
2. Enter how many units arrived in good condition and how many were damaged. The two are constrained so the total never exceeds the remaining quantity.
3. Add optional **Notes**, then **Create Goods Receipt**.

On success:
- A **Goods Receipt** is recorded with a **DO number** (Delivery Order) that is generated automatically — you'll see the toast *"Goods receipt created (DO-…)"*. The DO number also appears on the PO detail drawer.
- Good stock is added to inventory automatically. Damaged stock is not.
- The PO status is recalculated: **Partial Received** (some items still outstanding) or **Fully Received** (everything received).

You can receive goods in multiple batches until the PO is fully received. The PO detail drawer lists all DO numbers.

#### Cancelling a Purchase Order

Use **Cancel PO** on a **Draft** or **Confirmed** PO (with confirmation). Once a PO is fully received it can no longer be cancelled.

---

### 13. Stock Opname (Stock Count)

Stock Opname is the physical stock count workflow. It produces an official record of actual vs. system stock, and — after approval — automatically adjusts inventory.

#### The 9-State Workflow

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

#### Creating a Session

1. On the Stock Opname page click **New Stock Opname**.
2. Optional **Title**.
3. Optional **Blind count** checkbox — *hide system quantities from counters* (counters only see physical numbers, so they are not biased).
4. Add one or more **Scopes** — pick a scope type (e.g. store, warehouse, category, product, etc.) and the specific value. A "manual" row covers **all active products**.
   - The session covers the union of the selected scopes. Sessions may run in parallel as long as they never count the same SKU.
   - **Storage Location (Rack)** is a scope type that counts the products sitting in one rack. It must be the *only* scope of the session. Expected quantities come from the rack's `product_stock` row (products with no rack row are expected at 0). When the session is **posted**, the rack row is corrected to the physical count, and the global stock is recomputed as *the old global minus the old rack figure (never below 0), plus the new rack count* — so a rack count reconciles the sub-account with the global number even when sales have caused the two to drift apart.
5. Optional **Notes**, then create.

#### Assigning Counters

While the session is Draft/Open/Counting/Needs Recount, an assigner can **Assign Counter** — add counter users to the session. Only assigned counters can enter counts.

> By role: **Manager/admin/superadmin** create, assign, verify, post, and close sessions (they cannot enter counts). **Cashiers and staff** hold the `stock_opname.count`/`stock_opname.submit` permissions and are the usual counters — a manager assigns them to a session before counting begins.

#### Opening & Counting

- **Open Session** (from Draft) — requires a comment explaining *why this session is being opened*.
- **Start Counting** — begins the counting phase. A counter enters the **physical** quantity for each product using the **Count** button on each item row.
- Blind sessions hide system quantities during entry.

#### Verification

- **Submit for Verification** (from Counting) — sends the results to a verifier.
- **Verify / Reject** (from Verification):
  - *"Verifying approves the count without changing inventory. Posting is a separate step."*
  - Rejecting returns the session to counting.
- **Request Recount** — returns the session to counting for corrections.
- **Resume Counting** — counters continue after a recount request.

#### Posting Adjustments

After a session is **Approved**, an authorized user **Posts the Adjustment**:
- *"Posting applies the verified differences to inventory and creates an adjustment document (IA-…)."*
- The toast shows *"Adjustment {number} posted — inventory adjusted"*.
- Stock is updated for every item with a difference, and the **Adjustments Report** records each IA- document.

Posting is deliberately separate from verification (separation of duties) — the person who verifies should not be the only one who posts.

#### Closing & Cancelling

- **Close Session** (from Posted) — *"This finalises the record."*
- **Cancel** (from Draft/Open/Counting/Needs Recount) — *"This cannot be undone."*

#### Adjustments Report

**Stock Opname → Adjustments** (`/stock-opnames/adjustments`) lists all adjustment documents (IA-…):
- Search by adjustment/session number, filter **Posted/Reversed**.
- Columns: Adjustment number, Session, Status, Total Diff, Total Value, Created By, Created At.
- Rows link back to the source session.

> Sessions can be exported to CSV from the list (per-row **Export CSV**) and from the detail page.

---

### 14. Konsinyasi Supplier

The **Konsinyasi Supplier** (Consignment Supplier) module (`/consignment`) manages consignment stock — goods owned by a supplier that sit on your shelves. You sell them at terms you agree on, and the supplier is paid only after the goods are sold and a settlement is processed.

#### Core Concept — How Consignment Works

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

#### Prerequisites

Before using the consignment module, you need:

1. **A supplier marked as konsinyasi.** Go to **Suppliers** -> Add or Edit a supplier -> toggle **Supplier Konsinyasi** on. Only suppliers with this flag appear in the consignment module.
2. **Products linked to that supplier.** The supplier must have products assigned to it (done via the Supplier detail or Product edit page).
3. **Permission.** Your role needs `consignment.view` at minimum. Creating, updating, settling, and paying each require separate permissions — see [Appendix A](#appendix-a-role--permission-matrix).

#### Navigating the Module

Open **Konsinyasi Supplier** from the sidebar. You'll see the **Arrangements List** — a table of all consignment arrangements across your accessible stores.

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
1. Click **Arrangement Baru** (New Arrangement) (top-right).
2. In the modal, select the **Supplier** from the dropdown (only consignment suppliers appear).
3. The **Store** defaults to your current store (superadmin/admin can change it).
4. Click **Create**. The arrangement appears in the list with status **Aktif**.

> If no consignment suppliers appear in the dropdown, go to **Suppliers** first and toggle the **Supplier Konsinyasi** (Consignment Supplier) flag on the supplier you want to use.

**Opening an arrangement:**
Click the **Buka** (Open) button on any row. This opens the arrangement detail view with six tabs. A back arrow (< Kembali (Back)) at the top-left returns you to the list.

The arrangement header shows the supplier name, status badge (**Aktif** / **Berakhir**), and the last visit date.

> **First-time setup:** If the arrangement has no pricing terms yet, the Terms tab opens automatically with a warning banner: "Set pricing terms before receiving goods." Once terms are added, subsequent opens land on the Receiving tab as usual.

**View Suppliers:** the arrangement list header has a **Lihat Pemasok** (View Suppliers) button that navigates to the Suppliers page filtered to show only consignment suppliers. A back button on the Suppliers page returns you to the arrangements list.

---

#### Tab 1: Penerimaan (Receiving Goods)

This tab records goods delivered by the supplier. It is the default tab when you open an existing arrangement that already has pricing terms set.

**The receipt history** shows all past receipts with columns: Receipt Number (CR-xxxxxx), Date, Items count, and Total Value.

**Recording a new receipt:**

1. Click **Catat Penerimaan** (Record Receipt) (top-right).
2. The receipt form opens with one empty product line. For each line:
   - **Produk** (Product, required) — search and select the product.
   - **Dibawa** (Brought) — the quantity the supplier delivered (default: 1).
   - **Ditolak** (Rejected) — units you refuse (damaged, wrong item, etc.). Default: 0.
   - **Accepted** = Dibawa minus Ditolak (shown automatically below the fields).
3. If the product has a term, the agreed price per unit is displayed (e.g. "Terms: Rp 50,000 per unit"). If there is **no term** for the product, a yellow warning "Belum ada terms" (No terms yet) appears — go to the Terms tab first to add one.
4. Click **+ Tambah Baris** (Add Row) to add more product lines.
5. Optionally add **Catatan** (Notes) at the bottom.
6. Click **Simpan** (Save) to save. A toast confirms the receipt number and the receipt appears in the history.

> **Stock impact:** accepted quantities are added to the supplier's consignment stock immediately.

---

#### Tab 2: Terms (Price & Store Share)

Terms define the pricing agreement for each product on consignment. **You must set terms before receiving goods** for those products — otherwise the receipt form will warn you.

**The terms list** shows: Product (name + SKU), Price (per unit), and Store Share.

**Adding a term:**

1. Click **Tambah Term** (Add Term) (top-right).
2. Fill in the form:
   - **Produk** (Product, required) — search and select the product.
   - **Harga (Rp)** (Price, required) — the agreed retail price per unit.
   - **Jenis Share** (Share Type, required) — choose one:
     - **Persentase (%)** (Percentage) — the store keeps a percentage of each sale (e.g. 20%). Must be between 0 and 100 (exclusive).
     - **Nominal Tetap (Rp)** (Fixed Amount) — the store keeps a fixed Rp amount per unit sold. Must be greater than 0.
   - **Nilai Share** (Share Value) — the share value (percentage or Rp amount, depending on the type above).
3. Click **Simpan** (Save).

> **Example:** Price = Rp 50,000, Share Type = Persentase, Share Value = 20. When one unit is sold, the store keeps Rp 10,000 and the supplier is owed Rp 40,000.

> **Note:** When you save terms, the entire list is replaced. The system preserves all existing terms automatically when you add a new one — only the newly added term needs to be filled in.

---

#### Tab 3: Retur Tertunda (Pending Return)

A **Retur Tertunda** (Pending Return) records items pulled off the display **before** they are physically handed back to the supplier. This removes them from available stock while keeping them as supplier ownership until the formal return.

**The list shows:** Product (name + SKU), Quantity, Reason, Status (**Terbuka** = Open / **Diproses** = Returned/Fulfilled), and Date.

**Recording a pending return:**

1. Click **Catat Retur Tertunda** (Record Pending Return) (top-right).
2. Fill in the form:
   - **Produk dari Stok** (Product from Stock, required) — the dropdown shows only products with available consignment stock, along with their available quantity (e.g. "Kopi ABC (SKU-001) — Stok tersedia 50" (Available stock 50)).
   - **Jumlah** (Quantity, required) — how many units to pull from display. Cannot exceed the available stock (a "Maks: XX" (Max: XX) hint appears below the field).
   - **Alasan** (Reason, required) — pick one: **Rusak** (Damaged), **Kadaluarsa** (Expired), **Retur Pelanggan** (Customer Return), or **Lainnya** (Other).
   - **Catatan** (Notes, optional) — free-text notes.
3. Click **Simpan** (Save).

> **Stock impact:** the product's available stock decreases by the pending return quantity. The pending return quantity is tracked separately until a formal return is created.

**Cancelling a pending return:** A pending return in **Terbuka** (Open) status can be cancelled via the API, which restores the available stock.

---

#### Tab 4: Retur (Formal Return)

A formal return records the **physical hand-back** of goods to the supplier. It generates an RT-xxxxxx document and removes the items from supplier ownership entirely.

**The list shows:** Return Number (RT-xxxxxx), Date, Item count, and Total quantity returned.

If there are open pending returns, a yellow notice appears at the top: *"X retur tertunda terbuka"* (X pending returns open) — reminding you to link them.

**Recording a formal return:**

1. Click **Catat Retur** (Record Return) (top-right).
2. The form opens with one empty product line. For each line:
   - **Produk** (Product, required) — search and select the product.
   - **Jumlah** (Quantity) — the quantity being returned.
   - **Alasan** (Reason) — pick one: **Rusak** (Damaged), **Kadaluarsa** (Expired), **Retur Pelanggan** (Customer Return), or **Lainnya** (Other).
   - **Link ke Retur Tertunda** (Link to Pending Return, optional) — if this return corresponds to an existing pending return, select it from the dropdown. This closes the pending return and reduces the pending_return_qty. You can leave it as "Tidak ada link" (No link) if no pending return applies.
   - **Catatan** (Notes, optional) — notes for this line.
3. Click **+ Tambah Baris** (Add Row) to return multiple products at once.
4. Optionally add **Catatan Keseluruhan** (Overall Notes) at the bottom.
5. Click **Simpan** (Save). A toast confirms the return number (RT-xxxxxx).

> **Stock impact:** the product's total consignment stock decreases by the returned quantity. If a pending return was linked, the pending_return_qty is also reduced.

---

#### Tab 5: Settlement & Payout

The settlement tab handles the financial side — calculating what you owe the supplier for sold goods and recording payments.

This tab has two sections:

##### Unsettled Sales Preview (top card)

This shows all completed POS sales of consignment items that have **not yet been settled** — i.e. items the supplier is still owed money for.

**The preview table shows per product:** Product name, Quantity sold, Unit Price, Subtotal, and Store Share amount.

**The footer row shows three totals:**
- **Total Penjualan** (Total Sales) — total sale value of unsettled items.
- **Hak Toko** (Store Share) — your store's total share.
- **Terhutang ke Supplier** (Owed to Supplier) — the amount you owe the supplier (= Total Penjualan minus Hak Toko).

**Creating a settlement:**

1. Review the unsettled items in the preview.
2. Click **Buat Settlement** (Create Settlement) (top-right). The button is disabled when there are no unsettled items.
3. A confirmation modal shows the number of items and the total payable amount.
4. Click **Buat Settlement** (Create Settlement) to confirm. A toast confirms the settlement number (CS-xxxxxx).
5. The unsettled preview clears (all items are now part of the settlement) and the settlement appears in the history below.

> Settlement covers **all** unsettled sales — you cannot settle only some items.

##### Settlement History (bottom card)

This lists all past settlements with columns: Settlement Number (CS-xxxxxx), Date, Total amount, and Status (**Menunggu Pembayaran** = Pending Payment / **Dibayar** = Paid).

**Recording a payout (paying the supplier):**

1. On a settlement with status **Menunggu Pembayaran** (Pending Payment), click **Bayar** (Pay).
2. The payout modal shows the outstanding amount at the top.
3. Fill in:
   - **Metode Pembayaran** (Payment Method, required) — select from the available payment methods (Cash, Card, E-Wallet, Transfer, QRIS, etc.).
   - **Jumlah** (Amount, required) — defaults to the full outstanding amount. You can enter a lower amount for partial payment; the settlement remains pending until fully paid.
   - **No. Referensi** (Reference Number, optional) — a reference number (e.g. transfer receipt number).
   - **Catatan** (Notes, optional) — notes about the payment.
4. Click **Bayar** (Pay). A toast confirms the payout number (CP-xxxxxx).

> The settlement status changes to **Dibayar** (Paid) only when the total paid equals the total payable. Until then, the **Bayar** (Pay) button remains available for additional payments.

---

#### Tab 6: Stok (Consignment Stock)

This is a read-only view of the consignment stock for this supplier.

**The stock table shows per product:**

| Column | Meaning |
|--------|---------|
| Product | Product name and SKU |
| Stok Tersedia (Available Stock) | Available quantity (can be sold) |
| Retur Tertunda (Pending Return) | Quantity pending return (pulled from display, not yet handed back) |

> **How stock changes:** Receipts increase available stock. POS sales decrease both total and available stock. Pending returns decrease available stock and increase pending_return_qty. Formal returns decrease total stock and decrease pending_return_qty.

---

#### Quick Reference: Document Numbers

| Document | Format | Created When |
|----------|--------|-------------|
| Receipt (Penerimaan) | CR-xxxxxx | You record goods received from the supplier |
| Return (Retur) | RT-xxxxxx | You record goods physically handed back to the supplier |
| Settlement | CS-xxxxxx | You create a settlement for unsold items |
| Payout (Pembayaran) | CP-xxxxxx | You record a payment to the supplier |

#### Quick Reference: Stock Math

| Event | Total Stock | Available Stock | Pending Return |
|-------|:-----------:|:---------------:|:--------------:|
| Receipt (goods received) | increases | increases | — |
| POS Sale | decreases | decreases | — |
| Pending Return created | — | decreases | increases |
| Pending Return cancelled | — | increases | decreases |
| Formal Return | decreases | — | decreases |

#### Complete Walkthrough — End-to-End Example

Here is a full example of a consignment flow for a supplier "Toko Kopi Maju":

**Step 1 — Setup**
1. Go to **Suppliers**. Create or edit "Toko Kopi Maju" and toggle **Supplier Konsinyasi** (Consignment Supplier) on.
2. Link the products this supplier will provide (e.g. "Kopi Robusta 250g", "Teh Hijau 100g").
3. Go to **Konsinyasi Supplier** (Consignment Supplier). Click **Arrangement Baru** (New Arrangement), select "Toko Kopi Maju", and create.

**Step 2 — Set Terms**
1. Open the arrangement. Go to the **Terms** tab.
2. Add a term for "Kopi Robusta 250g": Price = Rp 45,000, Share = Persentase (Percentage) 25%.
3. Add a term for "Teh Hijau 100g": Price = Rp 25,000, Share = Nominal Tetap (Fixed Amount) Rp 5,000.

**Step 3 — Receive Goods**
1. Switch to the **Penerimaan** (Receiving) tab. Click **Catat Penerimaan** (Record Receipt).
2. Line 1: Kopi Robusta 250g, Dibawa (Brought) = 100, Ditolak (Rejected) = 2. Accepted = 98.
3. Line 2: Teh Hijau 100g, Dibawa (Brought) = 200, Ditolak (Rejected) = 0. Accepted = 200.
4. Click **Simpan** (Save). Receipt CR-000001 is created. Stock increases.

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
1. Go to the **Settlement** tab. The preview shows 5 units of Kopi Robusta sold.
   - Total Penjualan (Total Sales): Rp 225,000 (5 x Rp 45,000)
   - Hak Toko (Store Share): Rp 56,250 (5 x Rp 11,250)
   - Terhutang ke Supplier (Owed to Supplier): Rp 168,750
2. Click **Buat Settlement** (Create Settlement) and confirm. Settlement CS-000001 is created.
3. Finance pays the supplier via bank transfer. Click **Bayar** (Pay) on CS-000001.
4. Select Transfer, enter the full amount Rp 168,750, add the transfer reference number. Click **Bayar** (Pay).
5. Settlement status changes to **Dibayar** (Paid). Done.

---

### 15. Reports

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

### 16. Store Management

The **Stores** page (`/stores`, Indonesian UI) manages store branches.

- **Tambah Toko** (Add Store) → **Nama Toko** (Store Name, required, e.g. "Cabang Bandung"), optional **Alamat** (Address) and **Telepon** (Phone).
- **Edit** — change details and toggle **Aktif** (Active).
- **Delete** — the confirmation suggests deactivating instead of deleting.

Active stores are used elsewhere in the system (e.g. as a scope for storage locations and stock opname, and as the outlet filter for pricing rules).

---

### 17. Administration

The Administration group is shown only to **admin** and **superadmin** (Audit Logs is superadmin-only).

#### Users

Manage login accounts:
- **Add User** — username (alphanumeric), email, **password** (min 6 characters), **role** (superadmin/admin/cashier/manager/staff), **active** status, and an optional **reports-to** manager (or *None (top-level)*).
- **Edit** — change details, role, active status, or set a **new password** (leave blank to keep the current one).
- Deactivate or delete users. The superadmin account cannot be deleted and deleting users is superadmin-only.

> There is no self-service "change password" screen. Passwords are set/reset by an administrator through User Management.

#### Roles & Permissions

Custom roles let you grant exactly the right permissions:
- **Create Role** — Step 1: name + description. Step 2: tick permission checkboxes grouped by area (User & Role, Product, Category, Sales, Inventory, Customer, Report, Dashboard, POS, System), with group toggles, a permission counter, and search.
- **Edit / Duplicate** (`(copy)` suffix) / **Delete** via the row menu. System roles cannot be deleted, and deleting roles requires superadmin (admin can create, edit, and duplicate roles but not delete them).
- Role permission changes take effect for members on their next request.

#### Audit Logs

A read-only log of important actions (who did what and when), with filters for action, resource, and date range, plus export. Superadmin only.

---

### 18. Import & Export

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

### Appendix A: Role / Permission Matrix

Legend: ✓ full access · ◐ partial/limited · — no access

| Capability | Superadmin | Admin | Manager | Cashier | Staff |
|------------|:---:|:---:|:---:|:---:|:---:|
| Dashboard | ✓ | ✓ | ✓ | ✓ | — |
| Point of Sale (create sale) | ✓ | ✓ | — | ✓ | — |
| View transactions | ✓ | ✓ | ✓ | ✓ (own) | — |
| Reports | ✓ | ✓ | ✓ | — | — |
| Shifts — open/close own | ✓ | ✓ | ✓ | ✓ | — |
| Shifts — view/review all | ✓ | ✓ | ✓ | — | — |
| Products — view | ✓ | ✓ | ✓ | ✓ | ✓ |
| Products — create/edit | ✓ | ✓ | ✓ | — | — |
| Products — delete | ✓ | ✓ | — | — | — |
| Inventory adjustment | ✓ | ✓ | ✓ | — | — |
| Categories — view | ✓ | ✓ | ✓ | ✓ | ✓ |
| Categories — create | ✓ | ✓ | ✓ | — | — |
| Categories — edit/delete | ✓ | ✓ | ✓ | — | — |
| Customers — view | ✓ | ✓ | ✓ | ✓ | — |
| Customers — create/update | ✓ | ✓ | ✓ | — | — |
| Customers — delete/export/import | ✓ | ✓ | ✓ | — | — |
| Customer groups — view | ✓ | ✓ | ✓ | ✓ | — |
| Customer groups — manage | ✓ | ✓ | ✓ | — | — |
| Suppliers (use module) | ✓ | ✓ | ✓ | — | — |
| Storage locations — manage | ✓ | ✓ | — | — | — |
| Pricing rules — create/manage | ✓ | ✓ | ✓ | ✓ (view) | — |
| Purchase orders — create/confirm/receive | ✓ | ✓ | ✓ | — | — |
| Stock opname — create/assign/verify/post/close | ✓ | ✓ | ✓ | — | — |
| Stock opname — count/submit | ✓ | ✓ | ✓ | ✓ | ✓ |
| Stock opname — export/report | ✓ | ✓ | ✓ | — | — |
| Konsinyasi — view | ✓ | ✓ | ✓ | — | — |
| Konsinyasi — create/update terms | ✓ | ✓ | ✓ | — | — |
| Konsinyasi — settle | ✓ | ✓ | ✓ | — | — |
| Konsinyasi — pay supplier | ✓ | ✓ | — | — | — |
| Stores — manage | ✓ | ✓ | — | — | — |
| Users — create/edit | ✓ | ✓ | — | — | — |
| Users — delete | ✓ | — | — | — | — |
| Roles — create | ✓ | ✓ | — | — | — |
| Roles — update/delete | ✓ | — | — | — | — |
| Audit logs — view | ✓ | ✓ | — | — | — |
| Audit logs — export | ✓ | ✓ | — | — | — |
| Application settings — view | ✓ | ✓ | — | — | — |
| Application settings — update | ✓ | — | — | — | — |
| Import/Export (product, category, customer) | ✓ | ✓ | ✓ (customer) | — | — |

> Permission codes are checked in real time. Even within a role, custom roles can be granted any subset of permissions (see [Roles & Permissions](#roles--permissions-1)). Exact permission codes per action: `dashboard.view`, `sale.create/view/lookup/detail/park`, `product.view/create/update/delete/export/import/history.view/cost.view`, `category.view/create/update/delete/export/import`, `customer.view/create/update/delete/export/import`, `customer_group.view/create/update/delete`, `pricing.view/create/update/delete`, `purchase_order.view/create/update/confirm/receive/cancel/delete`, `shift.view/create/review/audit`, `report.view`, `inventory.adjust`, `stock_opname.view/create/assign/count/submit/verify/post/close/recount/cancel/export/report`, `storage_location.view/create/update/delete`, `consignment.view/create/update/settle/pay`, `app_settings.view/update`, `store.view/create/update/delete`, `user.view/create/update/delete`, `role.view/create/update/delete`, `audit.view/export`. The Suppliers module has no dedicated permission code — its page is gated by `pricing.view`, so superadmin, admin, and manager can use it.

---

### Appendix B: Status Reference

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

---

## License

Proprietary - Developed for retail business use.
