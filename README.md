# RetailPOS

RetailPOS is a point-of-sale and inventory management system with a Go (Gin) backend and a SvelteKit frontend. It supports multi-store operations, role-based access control, real-time updates via WebSockets, and comprehensive inventory and sales tracking.

## Features

- **Authentication & Authorization**: JWT-based auth with refresh tokens, role-based permissions (admin/cashier), and per-store data isolation
- **Product Management**: CRUD operations, product groups/categories, barcode support, stock tracking
- **Point of Sale**: Cart interface, multiple payment methods, receipt generation (PDF), transaction history
- **Inventory**: Stock management, low-stock alerts, inventory export (Excel)
- **Analytics**: Dashboard statistics, sales charts, revenue reports
- **Real-time**: WebSocket notifications for stock updates and new sales
- **Multi-tenant**: Store-based data segregation (admin sees all, cashier restricted to assigned store)
- **Responsive UI**: Built with SvelteKit, TailwindCSS, Chart.js, and jsPDF

## Tech Stack

**Backend**
- Go 1.26+ with Gin web framework
- PostgreSQL (with migrations via SQL files)
- GORM ORM
- JWT authentication (access + refresh tokens)
- Gorilla WebSocket for real-time updates
- CORS-enabled API

**Frontend**
- SvelteKit 2 (Svelte 5)
- TypeScript
- Vite (build tool)
- TailwindCSS 4 for styling
- Axios for API calls
- Chart.js for analytics charts
- jsPDF + jsPDF-autotable for receipts
- XLSX for inventory export
- Lucide icons

## Requirements

- Go 1.26+
- Node.js 18+ / npm
- PostgreSQL 12+

## Installation

### 1. Clone and Setup

```bash
git clone <repo-url>
cd retailPos
```

### 2. Database Setup

Create a PostgreSQL database:

```sql
CREATE DATABASE retailpos;
```

### 3. Environment Configuration

Create a `.env` file in the project root:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=retailpos

# JWT secrets (generate strong random strings)
JWT_SECRET=your_access_jwt_secret_here
JWT_REFRESH_SECRET=your_refresh_jwt_secret_here

# Server port (optional, default: 8080)
PORT=8080
```

**Note**: Both `JWT_SECRET` and `JWT_REFRESH_SECRET` are required.

### 4. Run Database Migrations

```bash
# From project root
psql "host=$DB_HOST port=$DB_PORT user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME" -f migrations/0001_initial.sql

# Subsequent migrations will be applied automatically on startup
```

### 5. Seed Default Users

```bash
go run ./cmd/seed
```

Default accounts:
- **Admin**: `admin` / `admin123`
- **Cashier**: `cashier` / `cashier123`

## Running the Application

### Start Backend

```bash
# Development (with auto-reload if using air or similar)
go run main.go

# Or build and run
go build -o retailPos ./...
./retailPos
```

Backend runs on `http://localhost:8080` by default.

### Start Frontend

```bash
cd web
npm install
npm run dev -- --host
```

Frontend dev server runs on `http://localhost:4173` by default.

#### API URL Override (Optional)

If the backend runs on a different URL, set `VITE_API_URL`:

```bash
# In web/.env or shell
VITE_API_URL=http://localhost:8080/api
```

## Build for Production

```bash
cd web
npm run build
```

Built static files output to `web/build/`. The backend serves these files automatically when running the compiled binary.

## API Endpoints

### Authentication
- `POST /api/login` — Login (returns access + refresh tokens)
- `POST /api/logout` — Logout (revokes refresh token)
- `POST /api/refresh` — Refresh access token
- `GET /api/auth/validate` — Get current user + permissions

### Products
- `GET /api/products` — List products (with filters: search, pagination, store_id)
- `GET /api/products/:id` — Get single product
- `GET /api/products?barcode=<sku>` — Get product by SKU/barcode
- `POST /api/products` — Create product (admin only)
- `PUT /api/products/:id` — Update product (admin only)
- `DELETE /api/products/:id` — Soft-delete product (admin only)

### Product Groups
- `GET /api/product-groups` — List groups (with product counts)
- `POST /api/product-groups` — Create group (admin only)
- `PUT /api/product-groups/:id` — Update group (admin only)
- `DELETE /api/product-groups/:id` — Delete group (admin only)

### Sales
- `GET /api/sales` — Get sales history (filtered by user/store/date)
- `POST /api/sales` — Create new sale (cashier + admin)
- `GET /api/sales/chart` — Sales data for charts (public endpoint)

### Statistics
- `GET /api/stats` — Dashboard stats (total products, sales, revenue, low stock)

### Admin & Role Management (Admin only)
- `GET /api/permissions` — List all permissions
- `GET /api/roles` — List all roles
- `POST /api/roles` — Create role
- `PUT /api/roles/:id/permissions` — Update role permissions
- `DELETE /api/roles/:id` — Delete role
- `GET /api/users` — List users
- `PUT /api/users/:id/role` — Update user role
- `GET /api/inventory/export` — Export inventory to Excel

### WebSocket
- `GET /api/ws` — WebSocket endpoint for real-time updates

    Query parameters:
    - `store_id` — Store identifier (required)
    - `token` — JWT access token (optional; if omitted, connection is unauthenticated)

    Events you can listen to:
    - `stock_update` — Product stock changed
    - `new_sale` — New sale created
    - `error` — Connection/authentication errors

    Events you can send:
    - `ping` — Keep-alive heartbeat

## Project Structure

```
retailPos/
├── main.go                    # Application entry point
├── go.mod / go.sum            # Go dependencies
├── retailPos                  # Compiled backend binary (generated)
├── seeder                     # Database seeder binary (generated)
├── migrations/                # SQL migration files
│   ├── 0001_initial.sql
│   ├── 0002_add_barcode.sql
│   ├── 0002_add_product_name_to_sale_items.sql
│   ├── 0002_auth_tables.sql
│   ├── 0003_roles_table.sql
│   ├── 0004_seed_roles_permissions.sql
│   ├── 0005_migrate_user_roles.sql
│   ├── 0006_cleanup_payment_methods.sql
│   ├── 0007_seed_inventory_export_permission.sql
│   ├── 0008_add_store_id_to_users.sql
│   ├── 0009_add_store_id_to_products.sql
│   └── 0010_add_store_id_to_sales.sql
├── cmd/                       # CLI commands
│   ├── migrate/              # Migration runner
│   ├── seed/                 # Database seeder
│   └── seeder/               # Alternative seeder binary
├── internal/                  # Internal packages
│   ├── auth/                 # Authentication (JWT, middleware)
│   ├── handler/              # HTTP handlers + tests
│   ├── model/                # Data models
│   ├── repo/                 # Database repositories
│   ├── service/              # Business logic
│   └── ws/                   # WebSocket hub
├── web/                       # SvelteKit frontend
│   ├── src/
│   │   ├── routes/           # Page routes (auth + app)
│   │   ├── lib/
│   │   │   ├── api/          # API client & modules
│   │   │   ├── composables/  # useCart, useCheckout, etc.
│   │   │   ├── domain/       # Domain entities/services
│   │   │   ├── infrastructure/ # WebSocket client
│   │   │   └── stores/       # Svelte stores (auth, cart, UI)
│   │   ├── app.html          # HTML entry
│   │   └── app.css           # Global styles (Tailwind)
│   ├── package.json
│   ├── vite.config.js
│   └── svelte.config.js
├── .env                       # Environment variables (create from .env.example)
├── .env.example               # Example environment config
├── backend.log                # Backend logs
└── README.md
```

## Database Schema Overview

- **users** — User accounts (with role, store assignment, credentials)
- **roles** — Admin/cashier roles + permissions mapping
- **products** — Inventory items (SKU, barcode, price, stock, store_id)
- **product_groups** — Categories for organizing products
- **sales** — Sales transactions (cashier, payment method, store)
- **sale_items** — Line items within each sale
- **refresh_tokens** — Active refresh tokens for JWT rotation

## Multi-Store Access Control

- **Admin** users have no store restriction — they can access all data across all stores.
- **Cashier** users are assigned a `store_id` and can only view/create sales for their own store. Product visibility is also store-filtered.

## Running Tests

```bash
# Backend tests (from project root)
go test ./...

# Or specific package
go test ./internal/handler
```

Note: Tests require a running PostgreSQL database with the schema applied.

## Development Notes

- Backend uses Gin with grouped routes (`/api` prefix). Protected routes use `auth.AuthMiddleware` and `auth.RoleMiddleware`.
- Frontend API client (`web/src/lib/api/client.ts`) includes automatic JWT refresh logic on 401 responses.
- WebSocket messages are JSON with `type` and `payload` fields. The backend broadcasts on `stock_update` and `new_sale` events.
- Static assets (SvelteKit build) are served by the backend from `web/build/` at runtime.
- CORS is configured to allow all origins in development; restrict for production.
- `PORT` environment variable controls the backend bind port (default: 8080).

## Default Credentials

| Role   | Username | Password   |
|--------|----------|------------|
| Admin  | admin    | admin123   |
| Cashier| cashier  | cashier123 |

**Change these after first login!**

## Security Considerations

- Store `JWT_SECRET` and `JWT_REFRESH_SECRET` in environment variables; never commit them.
- Enable HTTPS in production (use a reverse proxy like Nginx).
- Set `AllowOrigins` in CORS to your frontend domain (not `*`) for production.
- The refresh token is stored in `sessionStorage` and sent via `X-Refresh-Token` header.
- Consider setting shorter JWT expiry times and implementing token blacklisting if needed.

## License

MIT
