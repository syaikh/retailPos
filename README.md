# Retail POS System

Sistem Point of Sale (POS) modern untuk toko retail dengan manajemen inventory, penjualan, reporting, dan kontrol akses berbasis role.

## Features

- **Point of Sale (POS)** — Transaksi penjualan dengan scanner, diskon, payment methods
- **Inventory Management** — Tracking stok, movement, low stock alerts, multi-category filter
- **Import & Export Framework** — Schema-driven reusable import/export for Products, Categories, Brands, UOMs, Customers with XLSX templates, preview, validation, reference dropdowns, and import history
- **User Management** — RBAC (Role-Based Access Control) dengan permissions
- **Audit Logging** — Full audit trail untuk semua aksi (termasuk import)
- **Real-time Dashboard** — Statistik penjualan, revenue, analytics + live updates via WebSocket
- **WebSocket Support** — Notifikasi real-time
- **Swagger/OpenAPI** — API documentation via swaggo annotations
- **Structured Logging** — JSON (production) / text (development) via `log/slog`
- **EventBus Observability** — Atomic metrics for published/consumed/failed events
- **Dead-Letter Queue** — Failed events stored to PostgreSQL for retry

### Security Features

- JWT authentication with refresh token (HTTP-only cookie)
- CSRF protection on state-changing endpoints (validate, logout, change-password)
- Rate limiting with per-entry TTL (30min expiry, 5min cleanup)
- IP spoofing protection (uses `RemoteAddr` instead of `X-Forwarded-For`)
- Product search via tsvector (avoids ILIKE full table scan)
- Inventory adjustments use `SELECT ... FOR UPDATE` for concurrency safety

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Frontend (Nginx + Vite)              │
│  Port 5173 (HTTP) / 443 (HTTPS)                        │
│  Svelte 5 + Tailwind CSS 4 (dev: Vite HMR)             │
└────────────┬────────────────────────────────────────────┘
             │ /api/* → Backend
             │ /ws/*  → WebSocket
┌────────────┴────────────────────────────────────────────┐
│              Podman Pod (Shared Network)               │
├──────────────┬──────────────┬───────────────────────────┤
│ PostgreSQL   │ Go Backend   │ Nginx Frontend           │
│ Port 5432    │ Port 8080    │ Port 8081 → Host 5173    │
│ (Volume)     │ (Stateless)  │ (Static files)           │
└──────────────┴──────────────┴───────────────────────────┘
```

**Tech Stack:**
- **Backend:** Go (Gin), PostgreSQL, JWT Auth, WebSocket (gorilla/websocket), structured logging (slog)
- **Frontend:** Svelte 5, Tailwind CSS 4, Vite 6, Playwright
- **Infrastructure:** Podman (rootless), Nginx, systemd

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
  postgres:15-alpine
```

### Development

```bash
# 1. Seed database
./seed-dev.sh

# 2. Start backend (port 9095)
./run-dev.sh

# 3. Start frontend (port 5173)
cd web && npm run dev

# 4. Open http://localhost:5173
```

### Production

```bash
./deploy/podman-deploy.sh start
```

---

## Backend

### Module Structure

```
internal/
├── audit/             # Audit logging (domain events + listener)
├── brand/             # Brand CRUD + import adapter
├── category/          # Category CRUD + import adapter
├── customer/          # Customer CRUD + import adapter
├── eventbus/          # In-process event bus (retry, dead-letter, metrics)
├── inventory/         # Stock tracking, adjustments, low stock
├── middleware/        # Auth (JWT), CORS, rate limit, CSRF
├── platform/
│   └── importexport/  # Schema-driven import/export framework
├── product/           # Product CRUD (repository + query + bulk)
├── report/            # Sales reports & charts
├── sale/              # POS transaction processing + export
├── shared/            # Shared types, logger
├── uom/               # Unit of Measure CRUD + import adapter
├── user/              # User & auth management
└── config/            # App configuration
```

### Key Files

| File | Description |
|------|-------------|
| `cmd/server/main.go` | HTTP + WebSocket server entry point |
| `cmd/server/e2e_test.go` | End-to-end API tests |
| `internal/eventbus/bus.go` | Event bus with retry, dead-letter, observability |
| `internal/shared/logger.go` | Structured logging (slog) |
| `internal/sale/export.go` | CSV/XLSX export logic |
| `internal/product/query.go` | tsvector search + complex filters |
| `internal/product/bulk.go` | Bulk operations |
| `database/migrations/025_add_product_search_vector.sql` | Product full-text search |
| `database/migrations/026_add_dead_letter_events.sql` | Dead-letter events table |
| `docs/swagger.go` | OpenAPI annotations |

### Run Tests

```bash
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos TEST_DB_PASSWORD=admin123 \
DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only \
go test -p 1 -count=1 ./...
```

> `-p 1` forces sequential execution to avoid deadlocks from concurrent TRUNCATE/INSERT across packages.

### API Documentation

Swagger annotations are on key endpoints. To generate the spec:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/server/main.go -o docs/swagger
```

### API Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/login` | User login | No |
| POST | `/api/refresh` | Refresh token | No |
| POST | `/api/logout` | Logout | Yes |
| GET | `/api/stats` | Dashboard stats | Yes |
| GET | `/api/products` | List products (search + multi-category filter + tsvector) | Yes |
| POST | `/api/products` | Create product | Yes |
| PUT | `/api/products/:id` | Update product | Yes |
| DELETE | `/api/products/:id` | Delete product | Yes |
| POST | `/api/sales` | Create sale | Yes |
| GET | `/api/sales` | List sales | Yes |
| GET | `/api/sales/export` | Export sales (CSV/XLSX) | Yes |
| POST | `/api/inventory/adjust` | Manual stock adjustment | Yes |
| POST | `/api/change-password` | Change password | Yes |
| GET | `/api/import-export/modules` | List importable modules | Yes |
| GET | `/api/import-export/template/:module` | Download XLSX template | Yes |
| POST | `/api/import-export/preview/:module` | Preview import | Yes |
| POST | `/api/import-export/confirm/:module` | Confirm import | Yes |
| GET | `/api/import-export/export/:module` | Export data (CSV/XLSX) | Yes |
| GET | `/api/audit-logs` | List audit logs | Yes |
| GET | `/api/audit-logs/export` | Export audit logs | Yes |
| GET | `/health` | Health check | No |
| GET | `/ws` | WebSocket hub | Yes |

### Product Filtering

```
GET /api/products?search=mie&category=Makanan,Minuman&status=active&maxStock=10&sort=price_asc&limit=20&offset=0
```

Parameters: `search` (tsvector), `category` (comma-separated), `status`, `maxStock`, `sort` (`name_asc`, `name_desc`, `price_asc`, `price_desc`), `limit`, `offset`.

---

## Frontend

### Development

```bash
cd web
npm run dev       # Start dev server (port 5173)
npm run build     # Build for production
npx playwright test  # Run E2E tests
```

### Module Structure

```
web/src/
├── app/               # App shell (layout, topbar, sidebar)
├── lib/               # Shared UI components (Badge, Modal, Pagination, Skeleton)
├── modules/           # Feature modules
│   ├── admin/         # Roles management
│   ├── dashboard/     # Charts, stats, live updates
│   ├── import-export/ # Import wizard, history
│   ├── inventory/     # Stock management
│   ├── pos/           # Point of Sale
│   ├── product/       # Product catalog
│   ├── reporting/     # Reports with chart config + export utils
│   └── sale/          # Sales history
├── routes/            # Page routes
├── shared/            # Services, stores, types, utils
├── app.css            # Global styles & Tailwind imports
└── main.js            # Entry point
```

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | (required) | 256-bit secret for JWT signing. Generate: `openssl rand -hex 32` |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5433` (dev) | PostgreSQL port |
| `DB_USER` | `pos` | Database username |
| `DB_PASSWORD` | `admin123` | Database password |
| `DB_NAME` | `retail_pos` | Database name |
| `GIN_MODE` | `release` | Gin framework mode |
| `COOKIE_DOMAIN` | (empty) | Cookie domain for refresh token |
| `COOKIE_SECURE` | `false` | Set `true` for HTTPS |

Copy `.env.example` to `.env` for local development.

---

## Deployment

### Podman (Recommended)

```bash
./deploy/podman-deploy.sh start    # Start all services
./deploy/podman-deploy.sh status   # Check status
./deploy/podman-deploy.sh logs     # View logs
./deploy/podman-deploy.sh stop     # Stop all services
./deploy/podman-deploy.sh restart  # Restart
```

### Systemd

```bash
sudo cp deploy/retail-pos.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now retail-pos
```

### Database Migrations

Migrations run automatically on startup. Current schema: 026 migrations.

```bash
# Manual migration
PGPASSWORD=admin123 psql -h localhost -p 5433 -U pos -d retail_pos \
  -f database/migrations/025_add_product_search_vector.sql
```

---

## Default Credentials

| Role | Username | Password | Permissions |
|------|----------|----------|-------------|
| Superadmin | `superadmin` | `admin123` | All permissions |
| Admin | `admin` | `admin123` | User management, reports |
| Manager | `manager` | `admin123` | Inventory, sales view |
| Cashier | `cashier` | `admin123` | POS only |
| Staff | `staff` | `admin123` | Inventory + Dashboard |

Change these in production via `database/seeds/004_users.sql`.

---

## Permission Matrix

| Permission | Roles |
|------------|-------|
| `user:read`, `user:create`, `user:update`, `user:view` | admin, superadmin |
| `user:delete` | superadmin |
| `role:read`, `role:create` | admin, superadmin |
| `role:update`, `role:delete` | superadmin |
| `product:read` | all |
| `product:create`, `product:delete` | admin, superadmin |
| `product:update` | admin, manager, superadmin, staff |
| `category:read` | admin, manager, staff, superadmin |
| `category:create`, `category:update`, `category:delete` | admin, superadmin |
| `sale:read`, `sale:create` | admin, manager, cashier, superadmin |
| `sale:void` | admin, manager, superadmin |
| `inventory:read`, `inventory:adjust` | admin, manager, staff, superadmin |
| `inventory:export` | admin, superadmin |
| `report:read` | admin, manager, superadmin |
| `dashboard:read` | all |
| `pos:access` | cashier, manager, superadmin |
| `audit:read` | superadmin |

---

## Testing

### Backend Tests

```bash
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos TEST_DB_PASSWORD=admin123 \
DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only \
go test -p 1 -count=1 ./...
```

### E2E Tests (Playwright)

```bash
cd web && npx playwright test --reporter=list
```

---

## Deferred Items

See `docs/remediation-plan-2026-07-10.md` for full audit remediation status.

| Item | Status | Trigger |
|------|--------|---------|
| D1: Persistent Queue (pg_notify/Redis) | Deferred | Multi-instance scaling |
| D4: Backpressure Handling | Deferred | Event throughput bottleneck |

---

## License

Proprietary - Developed for retail business use.
