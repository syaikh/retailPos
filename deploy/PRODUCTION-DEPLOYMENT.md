# Retail POS System - Production Deployment Guide (Podman)

## Overview

This guide covers deploying the Retail POS System in production using **Podman containers** with a **pod architecture**. The system consists of:

- **Nginx** (port 80/443): Serves static frontend and reverse proxies API/WebSocket
- **Go Backend** (port 8080): REST API + WebSocket server
- **PostgreSQL** (port 5432): Database

All three containers run in a **single Podman pod** with shared network namespace.

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│              Host Machine (Podman)                 │
├─────────────────────────────────────────────────────┤
│  Pod: retail-pos-pod (shared network)              │
│  ├─ Container: Nginx (ports 80, 443 → 80, 443)    │
│  ├─ Container: Backend (port 8080 → 8080)         │
│  └─ Container: Postgres (port 5432 → 5432)        │
└─────────────────────────────────────────────────────┘

Network flow:
  [Client] → Nginx (80/443)
              ├─ / → serves static files (frontend)
              ├─ /api/ → proxies to Backend (localhost:8080)
              └─ /ws/ → upgrades to WebSocket (Backend)
```

---

## Prerequisites

### 1. System Requirements

- **OS:** Linux (tested on Fedora, RHEL, CentOS, Ubuntu)
- **Podman:** v4.0+ (install: `sudo dnf install podman` or `sudo apt install podman`)
- **Git:** For cloning repository
- **Make:** (optional) for using Makefile

### 2. Optional: Rootless vs Rootful

**Rootless (recommended for security):**
```bash
# Ensure user is in podman group
sudo usermod -aG podman $USER
newgrp podman
```

**Rootful (simpler for servers):**
```bash
# Use sudo for podman commands
sudo podman ...
```

---

## Quick Start (5 minutes)

### Step 1: Clone and Build Images

```bash
# Clone repository
git clone <your-repo-url>
cd retail-pos-system

# Build backend image
podman build -t retail-pos-backend:latest -f deploy/backend/Dockerfile .

# Build frontend image
podman build -t retail-pos-frontend:latest -f deploy/frontend/Dockerfile .

# Verify images
podman images | grep retail-pos
```

### Step 2: Configure Environment

Copy example environment file and adjust if needed:

```bash
cp deploy/.env.example .env
# Edit .env with your database credentials
nano .env
```

### Step 3: Deploy with Script

```bash
# Make script executable
chmod +x deploy/podman-deploy.sh

# Start all services
./deploy/podman-deploy.sh start

# Check status
./deploy/podman-deploy.sh status

# View logs
./deploy/podman-deploy.sh logs
```

### Step 4: Access Application

Open browser: **http://your-server-ip**

Login credentials:
- Username: `superadmin`
- Password: `admin123`

---

## Database Migrations & Fresh-DB Spin-up

Migrations are SQL files in `database/migrations/` (currently `000_squash.sql` through `030_consolidate_seed_permissions.sql`). They are **not** run automatically by the backend server — you must run them explicitly:

```bash
./deploy/podman-deploy.sh migrate   # applies every *.sql in database/migrations/
```

On a **fresh database** (or a fresh Postgres container), `migrate` now bootstraps the three prerequisites that `000_squash.sql` depends on before applying any migration:

1. `CREATE EXTENSION IF NOT EXISTS pgcrypto`
2. `CREATE SEQUENCE IF NOT EXISTS invoice_seq START 1`
3. `CREATE TABLE IF NOT EXISTS schema_migrations (...)` (tracks applied files)

It then applies each migration in sorted filename order with `ON_ERROR_STOP=1` and records each applied file in `schema_migrations`. Because `000_squash.sql` is idempotent and clears stale `00*.sql` tracking rows on each run, `migrate` can be re-run safely against an already-migrated database.

Migrations produce the full schema plus reference data: roles (5), permissions (74), role grants, the `superadmin`/`admin`/`manager`/`cashier`/`staff` users, payment methods, and customer groups (Walk-in/Member/VIP). **They do not create stores, products, customers, or sales** — run `./deploy/podman-deploy.sh seed` afterwards for dummy/business data.

> **Important:** Apply migrations **before** deploying a new server binary — several migrations carry ordering constraints (see `AGENTS.md` "Deployment" section for the full list).

---

## Manual Deployment (without script)

If you prefer manual control:

```bash
# 1. Create pod with ports
podman pod create --name retail-pos-pod -p 80:80 -p 443:443 -p 8080:8080

# 2. Create persistent volume for Postgres
podman volume create retail-pos-postgres-data

# 3. Start PostgreSQL container
podman run -d \
  --pod retail-pos-pod \
  --name postgres \
  -e POSTGRES_USER=pos \
  -e POSTGRES_PASSWORD=securepassword123 \
  -e POSTGRES_DB=retail_pos \
  -v retail-pos-postgres-data:/var/lib/postgresql/data \
  --restart unless-stopped \
  postgres:15-alpine

# 4. Start backend
podman run -d \
  --pod retail-pos-pod \
  --name backend \
  -e DB_HOST=localhost \
  -e DB_PORT=5432 \
  -e DB_USER=pos \
  -e DB_PASSWORD=securepassword123 \
  -e DB_NAME=retail_pos \
  -e GIN_MODE=release \
  --restart unless-stopped \
  retail-pos-backend:latest

# 5. Start frontend
podman run -d \
  --pod retail-pos-pod \
  --name frontend \
  --restart unless-stopped \
  retail-pos-frontend:latest
```

---

## Using Docker Compose (Alternative)

If you prefer Docker Compose syntax:

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

---

## Systemd Integration (Auto-Start on Boot)

For production servers, use systemd to manage the pod:

### Option A: Generated Systemd Service (recommended)

```bash
# Generate systemd unit from existing pod/containers
podman generate systemd --name retail-pos-pod --new --files

# This creates:
#  - retail-pos-pod.service
#  - retail-pos-pod-backend.service
#  - retail-pos-pod-postgres.service
#  - retail-pos-pod-frontend.service

# Move to systemd directory
sudo cp retail-pos-pod*.service /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload

# Enable and start
sudo systemctl enable --now retail-pos-pod.service

# Check status
sudo systemctl status retail-pos-pod.service
```

### Option B: Use Pre-made Service File

```bash
# Copy the provided service file
sudo cp deploy/retail-pos.service /etc/systemd/system/retail-pos.service

# Create posuser if needed
sudo useradd -r -s /sbin/nologin posuser || true

# Adjust paths in service file if needed (WorkingDirectory)
sudo sed -i 's|/home/posuser/retail-pos-system|/opt/retail-pos|g' /etc/systemd/system/retail-pos.service

# Move application to /opt/retail-pos
sudo mkdir -p /opt/retail-pos
sudo cp -r . /opt/retail-pos/
sudo chown -R posuser:posuser /opt/retail-pos

# Build images (as root or with podman)
cd /opt/retail-pos
sudo podman build -t retail-pos-backend -f deploy/backend/Dockerfile .
sudo podman build -t retail-pos-frontend -f deploy/frontend/Dockerfile .

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable retail-pos
sudo systemctl start retail-pos
sudo systemctl status retail-pos
```

---

## SSL/TLS Configuration (HTTPS)

### Using Let's Encrypt with Certbot

```bash
# Install certbot
sudo dnf install certbot python3-certbot-nginx   # Fedora/RHEL
# or: sudo apt install certbot python3-certbot-nginx   # Ubuntu

# Stop nginx temporarily to obtain cert
sudo systemctl stop retail-pos-pod   # or nginx container

# Obtain certificate
sudo certbot certonly --standalone -d yourdomain.com

# Copy certificate to nginx config directory
sudo mkdir -p /etc/retail-pos/ssl
sudo cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem /etc/retail-pos/ssl/
sudo cp /etc/letsencrypt/live/yourdomain.com/privkey.pem /etc/retail-pos/ssl/
sudo chmod 600 /etc/retail-pos/ssl/privkey.pem

# Update deploy/nginx/nginx.conf to enable SSL (uncomment SSL section)
# Then restart
sudo systemctl start retail-pos
```

### Auto-Renewal

```bash
# Add cron job
sudo crontab -e
# Add: 0 12 * * * /usr/bin/certbot renew --quiet --post-hook "systemctl restart retail-pos"
```

---

## Environment Variables

### Backend (deploy/backend/Dockerfile)

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host (inside pod use `localhost`) |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `pos` | Database username |
| `DB_PASSWORD` | (required) | Database password |
| `DB_NAME` | `retail_pos` | Database name |
| `GIN_MODE` | `release` | Gin mode (`release` for prod) |

### Postgres

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_USER` | `pos` | DB user |
| `POSTGRES_PASSWORD` | (required) | DB password |
| `POSTGRES_DB` | `retail_pos` | DB name |

### Frontend

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_URL` | `/api` | Base URL for API (set via Docker build arg) |

---

## Monitoring & Health Checks

### Check Service Status

```bash
./deploy/podman-deploy.sh status
```

### View Logs

```bash
# All logs
./deploy/podman-deploy.sh logs

# Specific service
./deploy/podman-deploy.sh logs backend
./deploy/podman-deploy.sh logs postgres
./deploy/podman-deploy.sh logs frontend
```

### Health Endpoints

- **Backend:** `curl http://localhost:8080/api/stats` (requires auth)
- **Frontend:** `curl http://localhost/` should return HTML
- **Database:** `podman exec postgres pg_isready -U pos`

---

## Backup & Restore

### Backup Database

```bash
# Create backup
podman exec postgres pg_dump -U pos retail_pos > backup_$(date +%Y%m%d).sql

# Compress
gzip backup_*.sql
```

### Restore Database

```bash
# Stop backend temporarily
podman stop retail-pos-backend

# Restore
zcat backup_20260429.sql.gz | podman exec -i postgres psql -U pos retail_pos

# Restart backend
podman start retail-pos-backend
```

### Backup Volume

```bash
# Stop services
podman pod stop retail-pos-pod

# Backup volume
podman volume export retail-pos-postgres-data > postgres-volume.tar

# Restore volume
podman volume import retail-pos-postgres-data postgres-volume.tar
```

---

## Troubleshooting

### Pod won't start (port already in use)

```bash
# Check what's using ports 80/443/8080
sudo ss -tulpn | grep -E ':80|:443|:8080'

# Kill conflicting process
sudo systemctl stop nginx   # if nginx is running
sudo systemctl stop apache2 # if apache is running
```

### Container crashes on startup

```bash
# Check logs
podman logs retail-pos-backend
podman logs retail-pos-frontend
podman logs postgres

# Common issues:
# - Database not ready: ensure postgres is healthy first
# - Migration errors: check Go backend logs
```

### Database connection errors

```bash
# Verify postgres is accepting connections
podman exec postgres psql -U pos -d retail_pos -c "SELECT 1;"

# Check backend env
podman exec backend env | grep DB_
```

### Frontend shows blank page

```bash
# Check nginx logs
podman logs frontend

# Verify dist files exist
podman exec frontend ls -la /usr/share/nginx/html/

# Check nginx config
podman exec frontend cat /etc/nginx/nginx.conf
```

### "Network error. Please try again" on login

This indicates frontend cannot reach backend. Fix:

```bash
# Ensure backend is running
podman ps | grep backend

# Test API directly
curl http://localhost:8080/api/stats

# If backend not responding, check logs
podman logs backend
```

---

## Scaling & Performance

### Horizontal Scaling (Multiple Backend Instances)

```bash
# Create separate network for load balancing
podman network create retail-pos-lb

# Run multiple backend instances
podman run -d --network retail-pos-lb --name backend1 ...
podman run -d --network retail-pos-lb --name backend2 ...

# Configure nginx upstream (in nginx.conf)
upstream backend {
    server backend1:8080;
    server backend2:8080;
}
```

### Resource Limits

Add to `podman run` commands:

```bash
--memory=512m \
--cpus=1.0 \
--pids-limit=100
```

Or use `docker-compose.yml` resource sections.

---

## Security Hardening

### 1. Use Non-Root Containers (already implemented)

All containers run as non-root user (`nginx` UID 1000, `retailpos` UID 1000).

### 2. Secrets Management

Store passwords in file instead of environment:

```bash
# Create secret file
echo "securepassword" > /etc/retail-pos/db_password.txt
chmod 600 /etc/retail-pos/db_password.txt

# Use in podman run
podman run ... -e DB_PASSWORD_FILE=/run/secrets/db_password \
  -v /etc/retail-pos/db_password.txt:/run/secrets/db_password:ro ...
```

### 3. Firewall Configuration

```bash
# Allow only necessary ports
sudo firewall-cmd --permanent --add-port=80/tcp
sudo firewall-cmd --permanent --add-port=443/tcp
sudo firewall-cmd --reload
```

### 4. Regular Updates

```bash
# Update images regularly
podman pull postgres:15-alpine
podman build -t retail-pos-backend:latest -f deploy/backend/Dockerfile .
podman build -t retail-pos-frontend:latest -f deploy/frontend/Dockerfile .

# Restart services
./deploy/podman-deploy.sh restart
```

---

## Uninstall / Cleanup

```bash
# Stop and remove everything
./deploy/podman-deploy.sh stop

# Remove images
podman rmi retail-pos-frontend retail-pos-backend

# Remove volume (WARNING: deletes all data!)
podman volume rm retail-pos-postgres-data

# Remove network
podman network rm retail-pos-network
```

---

## Migration from Python HTTP Server

Currently frontend uses `python3 -m http.server`. After containerization:

1. **No need for Python server** – Nginx serves static files directly
2. **Single command deployment** – `./deploy/podman-deploy.sh start`
3. **Auto-start on boot** – systemd service
4. **Better performance** – Nginx > Python HTTP server
5. **HTTPS ready** – Just add SSL certs

---

## Next Steps

- [ ] Set up SSL certificates with Let's Encrypt
- [ ] Configure log rotation (journald + logrotate)
- [ ] Set up monitoring (Prometheus metrics from backend)
- [ ] Add automated backups (cron job for pg_dump)
- [ ] Deploy to multiple servers with load balancer
- [ ] CI/CD pipeline for automatic image builds

---

## Support

For issues, check:
- Logs: `./deploy/podman-deploy.sh logs`
- Systemd: `sudo journalctl -u retail-pos -f`
- Podman: `podman pod ps` and `podman ps -a`

Full documentation: See README.md (to be created).
