# Retail POS System

Sistem Point of Sale (POS) modern untuk toko retail dengan manajemen inventory, penjualan, reporting, dan kontrol akses berbasis role.

## 🚀 Features

- **Point of Sale (POS)** - Transaksi penjualan dengan scanner, diskon, payment methods
- **Inventory Management** - Tracking stok, movement, low stock alerts
- **User Management** - RBAC (Role-Based Access Control) dengan permissions
- **Audit Logging** - Full audit trail untuk semua aksi
- **Real-time Dashboard** - Statistik penjualan, revenue, analytics
- **WebSocket Support** - Notifikasi real-time
- **Multi-store Ready** - Architecture mendukung multiple stores

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Frontend (Nginx)                     │
│  Port 5173 (HTTP) / 443 (HTTPS)                        │
│  Serves static files + Reverse Proxy                   │
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
- **Frontend:** HTML/CSS/JavaScript (Vanilla - no framework needed)
- **Infrastructure:** Podman (rootless), Nginx, systemd

## 📦 Quick Start (5 menit)

### Prerequisites

```bash
# Fedora/RHEL/CentOS
sudo dnf install podman git make

# Ubuntu/Debian
sudo apt install podman git make
```

### 1. Clone Repository

```bash
git clone <repository-url>
cd retail-pos-system
```

### 2. Build Images

```bash
# Backend (Go binary)
make build-backend

# Frontend (Nginx + static files)
make build-frontend

# Or build both:
make build-all
```

### 3. Deploy dengan Podman

```bash
# Start all services (Postgres + Backend + Nginx)
./deploy/podman-deploy.sh start

# Wait for services to be ready (~10 detik)
./deploy/podman-deploy.sh status
```

### 4. Access Application

```
Frontend: http://localhost:5173
Default credentials:
  Username: superadmin
  Password: admin123
```

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

Backend (in `deploy/podman-deploy.sh` or systemd service):

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host (inside pod) |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `pos` | Database username |
| `DB_PASSWORD` | (auto-generated) | Database password |
| `DB_NAME` | `retail_pos` | Database name |
| `GIN_MODE` | `release` | Gin framework mode |

Postgres (in deployment script):

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_USER` | `pos` | DB superuser |
| `POSTGRES_PASSWORD` | (auto-generated) | DB password |
| `POSTGRES_DB` | `retail_pos` | Initial database |

Frontend:
- `VITE_API_URL` set to `/api` during build (npx will proxy to backend)

### Changing Ports

Edit `deploy/podman-deploy.sh`:

```bash
HOST_FRONTEND_PORT=8080   # External port (host)
```

Or set environment variable before running:
```bash
HOST_FRONTEND_PORT=80 ./deploy/podman-deploy.sh start
# Note: port 80 requires rootful podman or capability
```

### Custom Database Credentials

```bash
# Create .env file
cp deploy/.env.example .env
# Edit .env with your values

# Or set inline
DB_USER=myuser DB_PASSWORD=mypass ./deploy/podman-deploy.sh start
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

```bash
# Ensure both servers are running
./deploy/podman-deploy.sh start

# Run tests
npx playwright test --reporter=list

# With UI (headed mode)
npx playwright test --headed --slowMo 1000
```

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
├── web/                 # Frontend source
│   ├── index.html       # Single page app
│   ├── app.css          # Styles
│   └── vite.config.js   # Build config
├── internal/            # Backend logic (clean architecture)
│   ├── delivery/http/   # HTTP handlers
│   ├── domain/          # Business entities
│   ├── repository/      # Data access
│   └── usecase/         # Business logic
├── database/
│   ├── migrations/      # SQL migrations
│   └── seeds/           # Seed data (users, roles, products)
├── deploy/
│   ├── backend/         # Backend Dockerfile
│   ├── frontend/        # Frontend Dockerfile + nginx.conf
│   ├── podman-deploy.sh # Deployment script
│   ├── docker-compose.yml
│   ├── retail-pos.service # systemd unit
│   └── PRODUCTION-DEPLOYMENT.md
├── tests/
│   └── e2e/             # Playwright E2E tests
└── Makefile             # Commands shortcut
```

## 🔑 Default Credentials

| Role | Username | Password | Permissions |
|------|----------|----------|-------------|
| Superadmin | `superadmin` | `admin123` | All permissions |
| Admin | `admin` | `admin123` | User management, reports |
| Manager | `manager` | `admin123` | Inventory, sales view |
| Cashier | `cashier` | `admin123` | POS only |

⚠️ **Change these in production!** Edit `database/seeds/004_users.sql` before first deployment.

## 🆘 Troubleshooting

### Frontend shows "503 Service Temporarily Unavailable"

```bash
# Check nginx logs
./deploy/podman-deploy.sh logs frontend

# Likely causes:
# 1. Backend not running → check backend logs
# 2. Rate limit exceeded → wait 1 minute or adjust nginx.conf
# 3. Port conflict → check if port 5173 already in use
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
| GET | `/api/products` | List products | Yes |
| POST | `/api/sales` | Create sale | Yes |
| GET | `/api/inventory/export` | Export inventory | Yes |
| GET | `/api/admin/users` | List users (admin) | Yes |
| POST | `/api/admin/users` | Create user (admin) | Yes |
| GET | `/ws` | WebSocket hub | Yes (token query) |

Full OpenAPI spec can be generated from handlers.

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
