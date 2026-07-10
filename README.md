# Retail POS System

Sistem Point of Sale (POS) modern untuk toko retail dengan manajemen inventory, penjualan, reporting, dan kontrol akses berbasis role.

## 🚀 Features

- **Point of Sale (POS)** — Transaksi penjualan dengan scanner, diskon, payment methods
- **Inventory Management** — Tracking stok, movement, low stock alerts, multi-category filter
- **Category Filter Modal** — Side-drawer multi-select with search, popular chips, and responsive grid
- **Import & Export Framework** — Schema-driven reusable import/export for Products, Categories, Brands, UOMs, Customers with XLSX templates, preview, validation, reference dropdowns, and import history
- **User Management** — RBAC (Role-Based Access Control) dengan permissions
- **Audit Logging** — Full audit trail untuk semua aksi (termasuk import)
- **Real-time Dashboard** — Statistik penjualan, revenue, analytics + live updates via WebSocket
- **WebSocket Support** — Notifikasi real-time

### Live Dashboard Behavior

- Dashboard stats auto-refresh via WebSocket `sale_created` events.
- Fallback polling hits `/api/dashboard/live` every 30 seconds.
- Four live cards: Today's Revenue, Transactions, Total Products, Low Stock Alerts.
- **Multi-store Ready** — Architecture mendukung multiple stores

## 🏗️ Architecture

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
- **Backend:** Go (Gin), PostgreSQL, JWT Auth, WebSocket (gorilla/websocket)
- **Frontend:** Svelte 5, Tailwind CSS 4, Vite 6, Playwright
- **Infrastructure:** Podman (rootless), Nginx, systemd

## 📦 Quick Start (5 menit)
### Frontend Technologies

- **Svelte 5** — Component-based UI with reactive state (`$state`, `$derived`, `$effect`)
- **Tailwind CSS 4** — Utility-first styling via Vite plugin
- **Vite 6** — Fast build tool and dev server (port 5173)
- **Playwright** — End-to-end browser testing
- **Lucide Svelte** — Icon library

### Frontend Development

```bash
# Start frontend dev server (port 5173)
cd web && npm run dev

# Build for production
cd web && npm run build

# Run E2E tests
cd web && npx playwright test
```

**Frontend architecture:**
```
web/
├── src/
│   ├── app.css               # Global styles & Tailwind imports
│   ├── app/
│   │   ├── layouts/          # Topbar, sidebar, breadcrumbs
│   │   └── pages/            # Route pages (Dashboard, POS, Inventory, etc.)
│   ├── lib/
│   │   └── components/ui/    # Reusable UI components (Badge, Modal, etc.)
│   ├── modules/              # Feature modules
│   │   └── import-export/    # Import wizard, history page, bulk actions
│   ├── routes/               # Route configuration
│   ├── shared/
│   │   ├── services/         # API services (import-export, auth, etc.)
│   │   ├── stores/           # Svelte stores (auth, toast)
│   │   ├── types/            # TypeScript interfaces
│   │   └── utils/            # Jakarta time, debounce, etc.
│   └── main.js               # Entry point
├── package.json
└── vite.config.js            # Vite + Svelte + Tailwind config
```

**Key UI Components:**
- `Modal.svelte` — Reusable centered dialog with fade/fly transitions
- `Badge.svelte` — Status badge (success, danger, warning, muted, primary variants)
- `Pagination.svelte` — Server-side pagination control
- `Skeleton.svelte` — Loading skeleton placeholder
- `ImportHistoryPage.svelte` — Import history list + detail views with accordion preview, row results grid
- `BulkActionDropdown.svelte` — Dropdown for import/export/template actions

### Prerequisites for Frontend

```bash
cd web
npm install
```

### Frontend Folder Structure

```
web/src/
├── app/                   # App shell
│   ├── layouts/           # Topbar, sidebar layouts
│   └── pages/             # Top-level route pages
├── lib/                   # Shared UI components (Svelte 5)
│   └── components/
│       └── ui/            # Badge, Modal, Pagination, Skeleton, etc.
├── modules/               # Feature modules
│   └── import-export/     # Import history page, import wizard
├── routes/                # Page routes & navigation
├── shared/                # Shared code
│   ├── services/          # API service functions
│   ├── stores/            # Auth, toast stores
│   ├── types/             # TypeScript interfaces
│   └── utils/             # Jakarta time, debounce, etc.
├── app.css                # Global styles & Tailwind imports
└── main.js                # Entry point
```

### Backend Development

```bash
# 1. Ensure postgres-dev is running (port 5433)
podman run -d --name postgres-dev -p 5433:5432 \
  -e POSTGRES_USER=pos -e POSTGRES_PASSWORD=admin123 -e POSTGRES_DB=retail_pos \
  postgres:15-alpine

# 2. Seed database with test data
./seed-dev.sh

# 3. Start backend (port 9095)
./run-dev.sh

# 4. In another terminal, start frontend (port 5173)
cd web && npm run dev

# 5. Open http://localhost:5173
```

### Backend Architecture

Modular monolith with domain modules under `internal/`. Each module owns its handlers, service, repository, and schema:

```
internal/
├── audit/             # Audit logging (domain events + listener)
├── brand/             # Brand CRUD + import adapter
├── category/          # Category CRUD + import adapter
├── customer/          # Customer CRUD + import adapter
├── inventory/         # Stock tracking, adjustments, low stock
├── middleware/        # Auth (JWT), CORS, rate limit, CSRF
├── platform/
│   └── importexport/  # Schema-driven import/export framework
│       ├── export/    # CSV/XLSX export engine
│       ├── handler/   # HTTP handlers (preview, confirm, history)
│       ├── history/   # Import history persistence (PG store)
│       ├── import/    # Import engine + validation pipeline
│       ├── progress/  # Job progress tracking (PG repo)
│       ├── schema/    # Module schema registry
│       ├── template/  # XLSX template generator
│       └── validation/# Row-level validation
├── product/           # Product CRUD + import adapter
├── report/            # Sales reports & charts
├── sale/              # POS transaction processing
├── shared/            # Shared types (importexport module schema)
├── uom/               # Unit of Measure CRUD + import adapter
├── user/              # User & auth management
└── config/            # App configuration
```

### Backend Folder Structure

```
go/
├── cmd/
│   ├── server/        # HTTP + WebSocket server entry point
│   └── seed/          # Database seeder entry point
├── internal/          # Modular monolith business logic
│   ├── brand/         #   ─ Brand module
│   ├── category/      #   ─ Category module
│   ├── customer/      #   ─ Customer module
│   ├── product/       #   ─ Product module
│   ├── uom/           #   ─ Unit of Measure module
│   ├── sale/          #   ─ POS / Sales module
│   ├── inventory/     #   ─ Stock module
│   ├── user/          #   ─ User & auth module
│   ├── audit/         #   ─ Audit trail module
│   ├── report/        #   ─ Reports module
│   ├── middleware/     #   ─ Shared middleware
│   ├── eventbus/      #   ─ In-process event bus
│   ├── config/        #   ─ App configuration
│   ├── shared/        #   ─ Shared types & interfaces
│   └── platform/
│       └── importexport/   # Import/Export framework
├── database/
│   ├── migrations/    # SQL migration files (up/down)
│   └── seeds/         # Seed data SQL files
└── deploy/            # Deployment configurations
```

### Backend Build & Install

```bash
# Build backend binary
make build-backend            # builds for linux/amd64
make build-backend-darwin     # builds for macOS (Apple Silicon)

# Install locally
make install-backend          # installs to $GOPATH/bin
```

### Deploy from Release

```bash
# Option 1: Use installer script (recommended)
curl -fsSL https://releases.example.com/install.sh | sudo bash

# Option 2: Download binary
VERSION=$(curl -s https://api.github.com/repos/owner/repo/releases/latest | grep '"tag_name"' | head -1 | cut -d'"' -f4)
wget https://releases.example.com/retail-pos-${VERSION}-linux-amd64.tar.gz
tar -xzf retail-pos-${VERSION}-linux-amd64.tar.gz
sudo mv retail-pos /usr/local/bin/
```

### Backend API Testing

```bash
# Health check (no auth)
curl http://localhost:9095/health

# Login
curl -X POST http://localhost:9095/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"superadmin","password":"admin123"}'

# Get products (with token)
TOKEN="<access_token_from_login>"
curl http://localhost:9095/api/products \
  -H "Authorization: Bearer $TOKEN"
```

### Run Backend Tests

Tests require PostgreSQL connection and `JWT_SECRET`. Use env vars to point to dev DB:

```bash
TEST_DB_PORT=5433 DB_PORT=5433 TEST_DB_USER=pos TEST_DB_PASSWORD=admin123 \
DB_USER=pos DB_PASSWORD=admin123 JWT_SECRET=test-secret-for-testing-only \
go test -p 1 -count=1 ./...
```

> **Note:** `-p 1` forces sequential test execution to avoid deadlocks from concurrent `TRUNCATE` and `INSERT` across packages that connect to the same database.

## 🔧 Detailed Installation

### Using Podman (Recommended Production)

The `deploy/podman-deploy.sh` script automates everything:

```bash
# Start services
./deploy/podman-deploy.sh start

# Check status
./deploy/podman-deploy.sh status

# View logs
./deploy/podman-deploy.sh logs          # all services
./deploy/podman-deploy.sh logs backend  # specific service

# Stop services
./deploy/podman-deploy.sh stop

# Restart
./deploy/podman-deploy.sh restart
```

**What it does:**
1. Creates a Podman pod with shared network
2. Starts PostgreSQL container with persistent volume
3. Runs database migrations & seeds (roles, permissions, users)
4. Starts Go backend container (listens on port 8080 internally)
5. Starts Nginx frontend container (maps host port 5173 → container 8081)
6. Waits for backend to be healthy
7. Shows connection details

### Systemd Service (Auto-start on Boot)

For production servers, configure systemd:

```bash
# Option 1: Use generated service
podman generate systemd --name retail-pos-pod --new --files
sudo cp retail-pos-pod*.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now retail-pos-pod.service

# Option 2: Use pre-made service
sudo cp deploy/retail-pos.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now retail-pos
```

Check status:
```bash
sudo systemctl status retail-pos
sudo journalctl -u retail-pos -f
```

### Docker Compose Alternative

If you prefer Docker Compose:

```bash
# Build images first
podman build -t retail-pos-backend -f deploy/backend/Dockerfile .
podman build -t retail-pos-frontend -f deploy/frontend/Dockerfile .

# Start with compose
podman compose -f deploy/docker-compose.yml up -d

# Check status
podman compose -f deploy/docker-compose.yml ps

# View logs
podman compose -f deploy/docker-compose.yml logs -f

# Stop
podman compose -f deploy/docker-compose.yml down
```

## ⚙️ Configuration

### Environment Variables

#### Development Environment (`.env` file)

For local development, copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

| Variable | Default | Description |
|----------|---------|-------------|
| `FRONTEND_PORT` | `5173` | Frontend dev server port (Vite) |
| `BACKEND_PORT` | `9095` | Backend dev server port |
| `DATABASE_PORT` | `5433` | PostgreSQL dev container port (postgres-dev) |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_USER` | `pos` | Database username |
| `DB_PASSWORD` | `admin123` | Database password |
| `DB_NAME` | `retail_pos` | Database name |

**Usage:**
```bash
# Start with custom ports
FRONTEND_PORT=3001 BACKEND_PORT=3002 npm run dev

# Or use the .env file (scripts auto-load it)
./run-dev.sh
./seed-dev.sh
```

#### Production Environment (`.env` file)

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host (inside pod) |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `pos` | Database username |
| `DB_PASSWORD` | (auto-generated) | Database password |
| `DB_NAME` | `retail_pos` | Database name |
| `GIN_MODE` | `release` | Gin framework mode |

### Custom Database Credentials

```bash
# Create .env file
cp deploy/.env.example .env
# Edit .env with your values

# Or set inline
DB_USER=myuser DB_PASSWORD=mypass ./deploy/podman-deploy.sh start
```

### Changing Ports

**Development:**
```bash
# Edit .env file
FRONTEND_PORT=3001
BACKEND_PORT=3002
DATABASE_PORT=5434
```

**Production:**
```bash
# Edit deploy/.env or set via environment
HOST_FRONTEND_PORT=8080   # External port (host)

# Or set environment variable before running:
HOST_FRONTEND_PORT=80 ./deploy/podman-deploy.sh start
# Note: port 80 requires rootful podman or capability
```

## 🔐 Security

### Production Hardening Checklist

- [ ] **Change default passwords** - Edit seed files or set `DB_PASSWORD` env var
- [ ] **Use SSL/TLS** - Add Let's Encrypt certificates to nginx
- [ ] **Firewall rules** - Only open ports 80, 443 (and 5173 if needed)
- [ ] **Non-root containers** - Already implemented (nginx UID 1000, retailpos UID 1000)
- [ ] **Secrets management** - Use Docker/Podman secrets or mounted files
- [ ] **Regular updates** - Update base images monthly

### Enable HTTPS with Let's Encrypt

```bash
# 1. Stop nginx temporarily
./deploy/podman-deploy.sh stop

# 2. Obtain certificate (standalone mode)
sudo certbot certonly --standalone -d yourdomain.com

# 3. Copy certs to nginx config directory
sudo mkdir -p /etc/retail-pos/ssl
sudo cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem /etc/retail-pos/ssl/
sudo cp /etc/letsencrypt/live/yourdomain.com/privkey.pem /etc/retail-pos/ssl/
sudo chmod 600 /etc/retail-pos/ssl/privkey.pem

# 4. Update nginx config (deploy/nginx/nginx.conf) - uncomment SSL section
# Change:
#   listen 8081 ssl;
#   ssl_certificate /etc/retail-pos/ssl/fullchain.pem;
#   ssl_certificate_key /etc/retail-pos/ssl/privkey.pem;

# 5. Rebuild frontend image
make build-frontend

# 6. Start services
./deploy/podman-deploy.sh start

# 7. Setup auto-renewal
sudo crontab -e
# Add: 0 12 * * * /usr/bin/certbot renew --quiet --post-hook "podman restart retail-pos-frontend"
```

## 🗄️ Database Management

### Backup

```bash
# Manual backup
podman exec postgres pg_dump -U pos retail_pos > backup_$(date +%Y%m%d).sql

# Automated backup script (cron)
0 2 * * * podman exec postgres pg_dump -U pos retail_pos > /backup/pos_$(date +\%Y\%m\%d).sql
```

### Restore

```bash
# Stop backend (to prevent writes)
podman stop backend

# Restore
podman exec -i postgres psql -U pos -d retail_pos < backup_20260429.sql

# Restart backend
podman start backend
```

### Volume Management

Data stored in Podman volume `retail-pos-postgres-data`:

```bash
# Backup volume
podman volume export retail-pos-postgres-data > postgres-volume.tar

# Restore volume
podman volume import retail-pos-postgres-data postgres-volume.tar
```

## 🧪 Testing

### Unit Tests (Backend)

```bash
cd cmd/server
go test ./... -v
```

### E2E Tests (Playwright)

E2E test suite covers inventory filtering, category multi-select, POS flow, dashboard, reports, admin, and login:

```bash
# Ensure both servers are running
./deploy/podman-deploy.sh start

# Run tests (headless)
npx playwright test --reporter=list

# Run with UI (headed mode + slowMo)
npx playwright test --headed --slowMo 1000

# Run specific spec
npx playwright test tests/e2e/inventory.spec.ts

# View HTML report
npx playwright show-report
```

Test configuration: `tests/e2e/fixtures.ts`, run script: `run-e2e.sh`

### API Testing

```bash
# Health check (no auth)
curl http://localhost:5173/api/stats
# Returns: {"error":"authorization token required"}

# Login
curl -X POST http://localhost:5173/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"superadmin","password":"admin123"}'

# Get stats (with token)
TOKEN="<access_token_from_login>"
curl http://localhost:5173/api/stats \
  -H "Authorization: Bearer $TOKEN"
```

## 📊 Monitoring

### Service Status

```bash
# Using script
./deploy/podman-deploy.sh status

# Raw podman
podman pod ps
podman ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
```

### Frontend Dev Health Check

```bash
# Frontend Vite dev server (hot reload)
curl -I http://localhost:5173

# Backend API (requires auth for /api/*)
curl -I http://localhost:5173/api/health

# Backend direct (dev port)
curl -I http://localhost:9095/health
```

### Logs

```bash
# All logs
./deploy/podman-deploy.sh logs

# Specific service
./deploy/podman-deploy.sh logs backend
./deploy/podman-deploy.sh logs postgres
./deploy/podman-deploy.sh logs frontend

# Systemd journal (if using systemd service)
sudo journalctl -u retail-pos -f
```

### Health Endpoints

```bash
# Backend API (requires auth for /api/*)
curl -I http://localhost:5173/api/health

# Nginx status
curl http://localhost:5173/

# Database
podman exec postgres pg_isready -U pos
```

## 🔄 Updates & Maintenance

### Update Application Code

```bash
# 1. Pull latest code
git pull

# 2. Rebuild images
make build-all

# 3. Restart services
./deploy/podman-deploy.sh restart
```

### Update Frontend Dependencies

```bash
cd web
npm update
npm run build
make build-frontend
./deploy/podman-deploy.sh restart
```

### Update Base Images (Security)

```bash
# Update nginx image
podman pull nginx:alpine
make build-frontend
./deploy/podman-deploy.sh restart

# Update postgres image
podman pull postgres:15-alpine
# Recreate pod (will keep volume)
./deploy/podman-deploy.sh stop
./deploy/podman-deploy.sh start
```

### Clean Up

```bash
# Stop services
./deploy/podman-deploy.sh stop

# Remove everything (WARNING: deletes data!)
make clean

# Remove only images
make clean-images

# Remove orphaned resources
podman system prune -a
```

## 📁 Project Structure

```
retail-pos-system/
├── cmd/
│   ├── server/          # Go backend main
│   ├── seed/            # Database seeder
│   └── dummy/           # Test utilities
├── web/                 # Frontend (Svelte 5 + Tailwind CSS 4)
│   ├── src/
│   │   ├── app.css      # Global styles & Tailwind imports
│   │   ├── app/         # App shell (layout, topbar, sidebar)
│   │   ├── lib/         # Shared UI components (Badge, Modal, etc.)
│   │   ├── modules/     # Feature modules (import-export, etc.)
│   │   ├── routes/      # Svelte route pages
│   │   ├── shared/      # Shared services, types, utils
│   │   └── main.js      # App entry
│   └── vite.config.js
├── internal/            # Backend modular monolith
│   ├── brand/           # Brand module
│   ├── category/        # Category module
│   ├── customer/        # Customer module
│   ├── product/         # Product module
│   ├── uom/             # Unit of Measure module
│   ├── sale/            # Sales / POS
│   ├── inventory/       # Stock tracking
│   ├── user/            # User & auth
│   ├── audit/           # Audit trail
│   ├── report/          # Reports
│   ├── middleware/       # Auth, CSRF, rate limit
│   ├── eventbus/        # In-process event bus
│   ├── config/          # Config
│   ├── shared/          # Shared types
│   └── platform/
│       └── importexport/ # Import/Export framework
├── database/
│   ├── migrations/      # SQL migrations (001-023)
│   └── seeds/           # Seed data (users, roles, products)
├── tests/
│   └── e2e/             # Playwright E2E tests
├── docs/
│   ├── examples/        # Example import XLSX files
│   └── archived-plans/  # Archived planning docs
├── deploy/
│   ├── backend/         # Backend Dockerfile
│   ├── frontend/        # Frontend Dockerfile + nginx.conf
│   ├── podman-deploy.sh # Deployment script
│   ├── docker-compose.yml
│   ├── retail-pos.service # systemd unit
│   └── PRODUCTION-DEPLOYMENT.md
├── Makefile
├── .env.example
├── AGENTS.md           # AI agent development guide
└── CONTRIBUTING.md
```

## 🔑 Default Credentials

| Role | Username | Password | Permissions |
|------|----------|----------|-------------|
| Superadmin | `superadmin` | `admin123` | All permissions |
| Admin | `admin` | `admin123` | User management, reports |
| Manager | `manager` | `admin123` | Inventory, sales view |
| Cashier | `cashier` | `admin123` | POS only |
| Staff | `staff` | `admin123` | Inventory + Dashboard |

⚠️ **Change these in production!** Edit `database/seeds/004_users.sql` before first deployment.

## 🔐 Permission Matrix

| Permission | Name | Description | Roles |
|------------|------|-------------|-------|
| **User Management** |||
| `user:read` | Baca user | Lihat daftar user (paginated) | admin, superadmin |
| `user:view` | Lihat detail user | Lihat detail satu user | admin, superadmin |
| `user:create` | Tambah user | Create new user account | admin, superadmin |
| `user:update` | Edit user | Modify user data/role | admin, superadmin |
| `user:delete` | Hapus user | Delete user account | superadmin |
| **Role Management** |||
| `role:read` | Baca role | View list of roles | admin, superadmin |
| `role:create` | Tambah role | Create new role | admin, superadmin |
| `role:update` | Edit role | Modify role & permissions | superadmin |
| `role:delete` | Hapus role | Delete role | superadmin |
| **Product Management** |||
| `product:read` | Baca produk | View product catalog | admin, manager, staff, cashier, superadmin |
| `product:create` | Tambah produk | Add new product | admin, superadmin |
| `product:update` | Edit produk | Edit product details | admin, manager, superadmin, staff |
| `product:delete` | Hapus produk | Delete product (soft delete) | admin, superadmin |
| **Category Management** |||
| `category:read` | Baca kategori | View categories list | admin, manager, staff, superadmin |
| `category:create` | Tambah kategori | Add new category | admin, superadmin |
| `category:update` | Edit kategori | Edit category details | admin, superadmin |
| `category:delete` | Hapus kategori | Delete category | admin, superadmin |
| **Sales** |||
| `sale:read` | Baca penjualan | View sales history | admin, manager, cashier, superadmin |
| `sale:create` | Buat penjualan | Process POS transaction | admin, manager, cashier, superadmin |
| `sale:void` | Void penjualan | Void/refund transactions | admin, manager, superadmin |
| **Inventory** |||
| `inventory:read` | Baca inventory | View inventory list | admin, manager, staff, superadmin |
| `inventory:adjust` | Adjust inventory | Manual stock adjustment | admin, manager, staff, superadmin |
| `inventory:export` | Export inventory | Export inventory data | admin, superadmin |
| **Reports & Dashboard** |||
| `report:read` | Lihat laporan | Access reports & charts | admin, manager, superadmin |
| `dashboard:read` | Lihat dashboard | View main dashboard | admin, manager, cashier, staff, superadmin |
| **POS** |||
| `pos:access` | Akses POS | Access POS page | cashier, manager, superadmin |
| **System (Superadmin Only)** |||
| `audit:read` | Lihat audit log | View system audit logs | superadmin |

### Role Summary

| Role | Permissions |
|------|-------------|
| **Superadmin** | Semua permission (termasuk audit:read, role:update, role:delete, user:delete) |
| **Admin** | Semua permission kecuali audit:read, role:update, role:delete, user:delete. Bisa manage user (create/read/update) tapi tidak bisa hapus user atau modifikasi role |
| **Manager** | product:read, product:update, sale:read, sale:void, report:read, dashboard:read, inventory:read, inventory:adjust, category:read |
| **Cashier** | product:read, sale:create, sale:read, pos:access, dashboard:read (Dashboard + POS) |
| **Staff** | product:read, inventory:read, inventory:adjust, category:read, dashboard:read (Dashboard + Inventory) |

## 🆘 Troubleshooting

### Frontend build "503 Service Temporarily Unavailable" or blank page

```bash
# Check nginx logs
./deploy/podman-deploy.sh logs frontend

# Dev server: check Vite compilation errors
cd web && npm run build

# Likely causes:
# 1. Backend not running → check backend logs
# 2. Rate limit exceeded → wait 1 minute or adjust nginx.conf
# 3. Port conflict → check if port 5173 already in use
# 4. Build error → cd web && npm run build to see compiler output
```

### Backend "health check failed"

```bash
# Check backend logs
podman logs backend

# Common issues:
# - Database not ready → wait longer or check postgres logs
# - DB connection string wrong → verify env vars
# - Migration failed → check migration SQL syntax
```

### Database "relation does not exist"

```bash
# Migrations didn't run
./deploy/podman-deploy.sh stop
podman volume rm retail-pos-postgres-data  # WARNING: deletes data!
./deploy/podman-deploy.sh start
```

### Port 5173 already in use

```bash
# Find what's using the port
sudo ss -tulpn | grep 5173

# Kill it or change HOST_FRONTEND_PORT
HOST_FRONTEND_PORT=8080 ./deploy/podman-deploy.sh start
```

### E2E tests fail with "Connection refused"

```bash
# Ensure services are running
./deploy/podman-deploy.sh status

# Check if frontend is accessible
curl http://localhost:5173/

# If not, restart
./deploy/podman-deploy.sh restart
```

### "Address already in use" (nginx bind error)

This happens when multiple nginx containers try to bind same port. Clean up:

```bash
podman pod rm -f retail-pos-pod
podman stop nginx 2>/dev/null || true
./deploy/podman-deploy.sh start
```

## 📚 API Documentation

API endpoints are documented in code handlers. Key endpoints:

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/login` | User login | No |
| POST | `/api/refresh` | Refresh token | No |
| POST | `/api/logout` | Logout | Yes |
| GET | `/api/stats` | Dashboard stats | Yes |
| GET | `/api/products` | List products (supports search + multi-category filter) | Yes |
| POST | `/api/products` | Create product | Yes |
| PUT | `/api/products/:id` | Update product | Yes |
| DELETE | `/api/products/:id` | Delete product | Yes |
| GET | `/api/categories` | List product categories | Yes |
| POST | `/api/sales` | Create sale | Yes |
| GET | `/api/inventory/export` | Export inventory | Yes |
| GET | `/api/admin/users` | List users (admin) | Yes |
| POST | `/api/admin/users` | Create user (admin) | Yes |
| GET | `/api/import-export/modules` | List importable modules | Yes |
| GET | `/api/import-export/template/:module` | Download import template (XLSX) | Yes |
| POST | `/api/import-export/preview/:module` | Preview & validate import file | Yes |
| POST | `/api/import-export/confirm/:module` | Confirm & start import | Yes |
| GET | `/api/import-export/progress/:jobId` | Get import job progress | Yes |
| POST | `/api/import-export/cancel/:jobId` | Cancel import job | Yes |
| GET | `/api/import-export/history/:module` | List import history | Yes |
| GET | `/api/import-export/history/:module/:jobId` | Import job detail + snapshot | Yes |
| GET | `/api/import-export/history/:module/:jobId/rows` | Import row results | Yes |
| GET | `/api/import-export/export/:module` | Export data (CSV/XLSX) | Yes |
| GET | `/ws` | WebSocket hub | Yes (token query) |

### Product Filtering

The `/api/products` endpoint supports:

- `search` — substring match on product name/SKU
- `category` — one or more category names, comma-separated (multi-select OR filter)
- `limit` / `offset` — pagination
- `maxStock` — low stock filter

Example: `GET /api/products?search=mie&category=Makanan,Minuman&limit=20`

## 🚀 Production Deployment Checklist

- [ ] Build images on production server (or push to registry)
- [ ] Configure firewall (open 80/443, optionally 5173)
- [ ] Set strong `DB_PASSWORD` in `deploy/.env` or script
- [ ] Change default user passwords in `database/seeds/004_users.sql`
- [ ] Configure SSL certificates (Let's Encrypt)
- [ ] Enable systemd service for auto-start
- [ ] Set up log rotation (`logrotate` for nginx + journald)
- [ ] Configure automated backups (cron + `pg_dump`)
- [ ] Set up monitoring (Prometheus metrics from `/metrics` if enabled)
- [ ] Test failover (restart services, verify auto-recovery)

## 🆘 Support

- **Logs:** `./deploy/podman-deploy.sh logs`
- **Systemd:** `sudo journalctl -u retail-pos -f`
- **Podman:** `podman pod ps` & `podman ps -a`

For bugs/features, open an issue on GitHub (if available).

## 📄 License

Proprietary - Developed for retail business use.

---

**Ready to deploy?** Run `./deploy/podman-deploy.sh start` and open http://localhost:5173 🎉
