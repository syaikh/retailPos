# RetailPOS

RetailPOS is a point-of-sale and inventory system with a Go backend and a Svelte frontend.

## Features

- User login with JWT authentication
- Product inventory management
- Sales creation via POS interface
- Admin and cashier roles
- WebSocket support for realtime updates
- Static SvelteKit frontend with build support

## Requirements

- Go 1.26+
- Node.js 18+ / npm
- PostgreSQL

## Setup

1. Clone the repository:

```bash
git clone <repo-url>
cd retailPos
```

2. Create a PostgreSQL database.

3. Create a `.env` file in the project root with the database and JWT settings:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_db_user
DB_PASSWORD=your_db_password
DB_NAME=your_db_name
JWT_SECRET=your_jwt_secret
PORT=8080
```

4. Run the database schema located in `migrations/0001_initial.sql`:

```bash
psql "host=$DB_HOST port=$DB_PORT user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME" -f migrations/0001_initial.sql
```

5. Seed default users:

```bash
go run ./cmd/seed
```

Default accounts:
- admin / admin123
- cashier / cashier123

## Run Backend

From the repository root:

```bash
go run main.go
```

Or build and run:

```bash
go build -o retailPos ./...
./retailPos
```

The API server will start on `http://localhost:8080` by default.

## Run Frontend

Install dependencies and start the Svelte frontend:

```bash
cd web
npm install
npm run dev -- --host
```

Open the web app in your browser at `http://localhost:4173` (or the port shown by Vite).

### Optional API override

If the backend runs on a different URL, set `VITE_API_URL` in `web/.env` or your shell:

```bash
VITE_API_URL=http://localhost:8080/api
```

## Production Build

```bash
cd web
npm run build
```

The built site is output to `web/build`.

## API Endpoints

- `POST /api/login` — login with JSON `{ "username": "...", "password": "..." }`
- `GET /api/products` — list products
- `GET /api/products?barcode=<sku>` — get a product by SKU
- `POST /api/products` — create a product (admin only)
- `POST /api/sales` — create a sale
- `GET /api/ws` — websocket endpoint

## Notes

- The backend reads database credentials from environment variables.
- The frontend uses `web/src/lib/api.js` and defaults to `http://localhost:8080/api`.
- Use the seed command after database setup to create initial users.
