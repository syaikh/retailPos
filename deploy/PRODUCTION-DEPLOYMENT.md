# Retail POS System - Production Deployment Guide (Podman)

## Overview

This guide covers deploying the Retail POS System in production using **Podman containers** with a **pod architecture**. The system consists of:

- **Nginx** (port 8081, published as 5173): Serves static frontend and reverse proxies API/WebSocket
- **Go Backend** (port 8080, published as 8080): REST API + WebSocket server
- **PostgreSQL** (port 5432, published as 5432 on the Podman paths only): Database

All three containers run in a **single Podman pod** with shared network namespace.

There is no TLS listener in these images. `deploy/nginx/nginx.conf` has exactly
one `listen 8081;` and no `ssl` block, and the frontend container publishes only
5173. Terminating TLS is left to a reverse proxy in front of the stack — see
*TLS Termination*. An earlier version of this guide claimed 80/443; that came from
the retired systemd unit, and it never matched the shipped nginx config.

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│              Host Machine (Podman)                 │
├─────────────────────────────────────────────────────┤
│  Pod: retail-pos-pod (shared network)              │
│  ├─ Container: Nginx (8081 → published 5173)      │
│  ├─ Container: Backend (8080 → published 8080)    │
│  └─ Container: Postgres (5432 → published 5432)   │
└─────────────────────────────────────────────────────┘

Network flow:
  [Client] → Nginx (5173)
              ├─ / → serves static files (frontend)
              ├─ /api/ → proxies to Backend (localhost:8080)
              └─ /ws/ → upgrades to WebSocket (Backend)
```

Postgres is published so the script's `seed` target can connect from the host.
That makes the database reachable on **every host interface**, not just loopback.
The compose path does not publish 5432 at all. If you do not need host-side
`psql`, drop the `PublishPort`/`-p` line and run migrations and seeding through a
temporary forward instead.

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

### Step 2: Create the Secret File

The backend **requires** `JWT_SECRET` and panics at startup without it. It does
not generate one on first run. Secrets live in a single root-only file that
`podman-deploy.sh` both sources and passes to the container via `--env-file`, so
they stay out of `argv` and the process table. They are still readable by
anyone who can run `podman inspect backend`, so keep the file mode `600` and
treat the host as the trust boundary.

```bash
sudo mkdir -p /etc/retail-pos
sudo tee /etc/retail-pos/backend.env >/dev/null <<EOF
DB_PASSWORD=$(openssl rand -hex 24)
JWT_SECRET=$(openssl rand -hex 32)
JWT_SECRET_REFRESH=$(openssl rand -hex 32)
EOF
sudo chmod 600 /etc/retail-pos/backend.env
```

Keep `JWT_SECRET_REFRESH` separate so access and refresh tokens can be rotated
independently. When it is unset the backend reuses `JWT_SECRET` for both.

Non-secret settings (ports, image tags, `CORS_ORIGIN`, `DB_USER`) can be
overridden either by exporting them in the shell before running the script or by
adding them to the same file. `deploy/.env.example` documents every supported
key and its default.

`podman-deploy.sh` validates this file before creating any container and reports
all missing values at once, so a misconfiguration fails immediately instead of
producing a backend crash loop.

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

## Database TLS

The backend defaults to `sslmode=require` in production (`ENV=production`) and
`disable` in development. The stock `postgres:18-alpine` image ships with
`ssl=off`, so **`require` will not work out of the box** — the backend fails
with `server refused TLS connection` and, because of `--restart
unless-stopped`, crash-loops.

`podman-deploy.sh` reads `SHOW ssl` after the database starts and refuses to
continue if the server cannot satisfy the configured mode, naming this
decision explicitly rather than letting it surface as a mystery crash loop.

Two supported options.

### Option A — mount a certificate (recommended)

The backend and database share a pod-internal network, so a self-signed
certificate is sufficient to satisfy `require`. It protects against anything
reading traffic on that network, but note it provides no authenticity guarantee
unless you also set `DB_SSLMODE=verify-full` with a CA the server's hostname
matches.

```bash
sudo mkdir -p /etc/retail-pos/pg-tls
sudo openssl req -new -x509 -days 3650 -nodes -text \
  -out /etc/retail-pos/pg-tls/server.crt \
  -keyout /etc/retail-pos/pg-tls/server.key \
  -subj "/CN=postgres"
sudo chmod 600 /etc/retail-pos/pg-tls/server.key
sudo chown 1000:1000 /etc/retail-pos/pg-tls/server.key
```

Then add to the `podman run` for postgres in `deploy/podman-deploy.sh`:

```bash
    -v /etc/retail-pos/pg-tls:/etc/pg-tls:ro,z \
    -c ssl=on \
    -c ssl_cert_file=/etc/pg-tls/server.crt \
    -c ssl_key_file=/etc/pg-tls/server.key \
```

Restart the database (`./deploy/podman-deploy.sh start postgres`) and confirm:

```bash
podman exec postgres psql -U pos -tAc "SHOW ssl;"   # must print: on
```

### Option B — disable TLS on the pod network

Defensible for a single-host deployment where the database is only reachable
from a sibling container on the pod's private network, and acceptable only as a
recorded decision. Set it in the secret file so the intent is explicit:

```bash
echo 'DB_SSLMODE=disable' | sudo tee -a /etc/retail-pos/backend.env
```

The backend logs a warning at every startup when it sees this, by design:

```
WARN database TLS disabled in production. Ensure the database is unreachable
from outside the deployment network, and record this decision.
```

Do not set `DB_SSLMODE=disable` if the database port is ever published to the
host or reachable from another machine.

---

## Database Migrations & Fresh-DB Spin-up

Migrations are SQL files in `database/migrations/` (currently `000_squash.sql`, `001`–`007`, `031`–`032`). They are **not** run automatically by the backend server — you must run them explicitly:

```bash
./deploy/podman-deploy.sh migrate   # applies every *.sql in database/migrations/
```

On a **fresh database** (or a fresh Postgres container), `migrate` now bootstraps the three prerequisites that `000_squash.sql` depends on before applying any migration:

1. `CREATE EXTENSION IF NOT EXISTS pgcrypto`
2. `CREATE SEQUENCE IF NOT EXISTS invoice_seq START 1`
3. `CREATE TABLE IF NOT EXISTS schema_migrations (...)` (tracks applied files)

It then applies each migration in sorted filename order with `ON_ERROR_STOP=1` and records each applied file in `schema_migrations`. Because `000_squash.sql` is idempotent and clears stale `00*.sql` tracking rows on each run, `migrate` can be re-run safely against an already-migrated database.

Migrations produce the full schema plus reference data: roles (5), permissions (85), role grants, the `superadmin`/`admin`/`manager`/`cashier`/`staff` users, payment methods, and customer groups (Walk-in/Member/VIP). **They do not create stores, products, customers, or sales** — run `./deploy/podman-deploy.sh seed` afterwards for dummy/business data.

> **Important:** Apply migrations **before** deploying a new server binary — several migrations carry ordering constraints (see `AGENTS.md` "Deployment" section for the full list).

---

## Manual Deployment (without script)

Equivalent to what `podman-deploy.sh start` does, for when you want explicit
control. Note the required settings — this path is easy to get wrong, which is
why the script exists. See [Database TLS](#database-tls) for `DB_SSLMODE`.

```bash
# 0. Secret file must already exist (see Step 2)
sudo test -r /etc/retail-pos/backend.env || { echo "missing secret file"; exit 1; }

# 1. Create pod with ports
#    5173→8081, not 80→80: nginx.conf has a single `listen 8081;`. Publishing
#    80/443 reaches nothing (this is the defect the retired systemd unit had).
#    5432 is published only so host-side psql/seed can connect; drop it if you do
#    not need that.
podman pod create --name retail-pos-pod -p 5173:8081 -p 8080:8080 -p 5432:5432

# 2. Create persistent volume for Postgres
podman volume create retail-pos-postgres-data

# 3. Start PostgreSQL container
#    POSTGRES_PASSWORD is passed without a value so it is read from the exported
#    environment rather than appearing in the process table.
set -a; sudo -E . /etc/retail-pos/backend.env; set +a
podman run -d \
  --pod retail-pos-pod \
  --name postgres \
  -e POSTGRES_USER=pos \
  -e POSTGRES_PASSWORD \
  -e POSTGRES_DB=retail_pos \
  -v retail-pos-postgres-data:/var/lib/postgresql \
  --restart unless-stopped \
  docker.io/library/postgres:18-alpine

# 4. Start backend. --env-file supplies DB_PASSWORD, JWT_SECRET and
#    JWT_SECRET_REFRESH; nothing secret appears in argv.
podman run -d \
  --pod retail-pos-pod \
  --name backend \
  --env-file /etc/retail-pos/backend.env \
  -e DB_HOST=localhost \
  -e DB_PORT=5432 \
  -e DB_USER=pos \
  -e DB_NAME=retail_pos \
  -e DB_SSLMODE=require \
  -e PORT=8080 \
  -e ENV=production \
  -e LOG_LEVEL=info \
  -e CORS_ORIGIN=https://pos.example.com \
  -e COOKIE_SECURE=true \
  -e GIN_MODE=release \
  --restart unless-stopped \
  localhost/retail-pos-backend:latest

# 5. Start frontend
podman run -d \
  --pod retail-pos-pod \
  --name frontend \
  --restart unless-stopped \
  localhost/retail-pos-frontend:latest
```

Required and easy to omit: `JWT_SECRET` (without it the backend panics),
`ENV=production` (without it you get text logs at `debug`), `CORS_ORIGIN` (the
`FRONTEND_URL` variable that older docs mention is read by no code),
`COOKIE_SECURE=true` (without it the refresh-token cookie is issued without the
`Secure` flag), and `DB_SSLMODE`.

---

## Using Docker Compose (Alternative)

`deploy/docker-compose.yml` is a supported alternative, aligned with the script:
both read `/etc/retail-pos/backend.env` for secrets, and both set `ENV`,
`CORS_ORIGIN`, `COOKIE_SECURE` and `DB_SSLMODE` explicitly.

```bash
# Build images first
podman build -t retail-pos-backend -f deploy/backend/Dockerfile .
podman build -t retail-pos-frontend -f deploy/frontend/Dockerfile .

# The secret file must exist first; compose fails fast if it does not.
sudo mkdir -p /etc/retail-pos
sudo tee /etc/retail-pos/backend.env >/dev/null <<EOF
DB_PASSWORD=$(openssl rand -hex 24)
JWT_SECRET=$(openssl rand -hex 32)
JWT_SECRET_REFRESH=$(openssl rand -hex 32)
EOF
sudo chmod 600 /etc/retail-pos/backend.env

# Start with compose
podman compose -f deploy/docker-compose.yml up -d

# Check status
podman compose -f deploy/docker-compose.yml ps

# View logs
podman compose -f deploy/docker-compose.yml logs -f

# Stop
podman compose -f deploy/docker-compose.yml down
```

Override the secret file location with `ENV_FILE=/path/to/backend.env`.

Two compose-specific notes:

- Compose resolves `${VAR}` from the host shell or a project `.env` file, never
  from a service's `env_file`. Secrets are therefore passed through `env_file`
  only; the manifest contains no `${DB_PASSWORD:-default}` fallback, because such
  a default resolves on the host and would silently become the real password.
- `DB_SSLMODE` defaults to `require` here as well, so the same mounted-certificate
  requirement described above applies. The compose file has no `SHOW ssl`
  preflight equivalent to the script's, so if the database starts with `ssl=off`
  the backend will exit rather than wait.

---

## Auto-Start on Boot (systemd + Quadlet)

`podman generate systemd` is no longer an option. It was deprecated in Podman 4.4
and **removed in Podman 5.0**; the command does not exist on a current host (this
one runs 5.8.7). The replacement is Quadlet, which ships with Podman and reads
unit definitions from a directory rather than generating them.

`deploy/quadlet/` holds the four units:

| File | Role |
|------|------|
| `retail-pos.pod` | network namespace and the published ports |
| `retail-pos-postgres.container` | PostgreSQL 18, credentials from the secret file |
| `retail-pos-backend.container` | Go API on 8080 |
| `retail-pos-frontend.container` | nginx on 8081, published as 5173 |

### Install (rootless, per-user systemd)

```bash
# Quadlet starts containers; it does not build them. Build the images first.
./deploy/podman-deploy.sh build

# The secret file must already exist (see deploy/.env.example).
#
# It has to be owned by the user running the service. The `sudo tee` above
# creates it as root, and a rootless user service runs as $USER — a root-owned
# 0600 file is unreadable to it, so the backend would fail to start with a bare
# permission error. Chown it, and keep the mode at 600.
sudo chown "$USER" /etc/retail-pos/backend.env
sudo chmod 600 /etc/retail-pos/backend.env

# User services only start at boot if the user manager is kept alive past
# logout. Without this, the stack comes up on your next login instead.
loginctl enable-linger "$USER"

mkdir -p ~/.config/containers/systemd
cp deploy/quadlet/* ~/.config/containers/systemd/
systemctl --user daemon-reload

# Enable *all three*, postgres included. `start` alone is not enough: an enabled
# backend whose database unit is not enabled will not come back after a reboot,
# because nothing starts postgres. Backend already declares
# Requires=/After=retail-pos-postgres.service, so ordering is handled.
systemctl --user enable --now retail-pos-postgres.service \
  retail-pos-backend.service retail-pos-frontend.service

# Inspect
systemctl --user status retail-pos-backend.service
journalctl --user -u retail-pos-backend.service -f
curl -fsS http://localhost:8080/health
curl -fsS http://localhost:5173/
```

For a **rootful** host, copy the files to `/etc/containers/systemd/` and drop
`--user` from every command.

### Known difference from `podman-deploy.sh`

Compose can gate the backend on `condition: service_healthy`. Quadlet cannot: it
orders the *start* of postgres, not its readiness. On a cold boot the backend may
start before `initdb` finishes and fail to connect. `Restart=always` in the unit
recovers from this, but it is a genuine behavioural difference.

For the stricter path — an explicit `pg_isready` wait and a `SHOW ssl` check
against the configured `DB_SSLMODE` — use `./deploy/podman-deploy.sh`. That script
remains the recommended path for a first deployment; Quadlet is for hosts that
have already been proven and just need boot persistence.

### If you are migrating from the old `retail-pos.service`

That unit has been deleted. It was broken: it set no `JWT_SECRET`, so the backend
could not have started; it read `POSTGRES_PASSWORD_FILE`/`DB_PASSWORD_FILE` from
`/run/secrets/db_password`, a path nothing ever created; it published ports 80
and 443 while `nginx.conf` only listens on 8081; and its `Documentation=` URL
still pointed at `github.com/your-repo`. `Documentation=https://github.com/your-repo/retail-pos-system`
was the least of its problems. See Recommendation 7 in
`docs/audits/production-deploy-config-audit-2026-09-26.md`.

---

## SSL/TLS Configuration (HTTPS)

`deploy/nginx/nginx.conf` has a single `listen 8081;` and **no TLS server block**,
so no container in this repository terminates HTTPS. Terminate it on the host, in
front of the published port.

### Using Let's Encrypt with Certbot

```bash
# Install certbot
sudo dnf install certbot python3-certbot-nginx   # Fedora/RHEL
# or: sudo apt install certbot python3-certbot-nginx   # Ubuntu

# The standalone challenge needs port 80, which nothing above publishes, so it
# can run directly without stopping the application.
sudo certbot certonly --standalone -d yourdomain.com

# Install a host nginx that terminates TLS and proxies to the container
sudo mkdir -p /etc/nginx/conf.d
# see deploy/nginx/nginx.conf for the upstream path; proxy_pass to
# 127.0.0.1:5173 and proxy /api to 127.0.0.1:8080, including the
# Upgrade/Connection headers the WebSocket handler needs.

# Then set CORS_ORIGIN to the public origin in the secret file and restart.
sudo systemctl --user restart retail-pos-backend.service
```

### Auto-Renewal

`certbot` installs a timer unit, so renewal needs no cron job. Confirm it:

```bash
sudo systemctl list-timers | grep certbot
```

If renewal reloads host nginx, add the hook to the certbot unit:

```bash
sudo systemctl edit certbot-renew.timer
# [Service]
# ExecStartPost=/usr/bin/systemctl reload nginx
```

---

## Environment Variables

### Backend (deploy/backend/Dockerfile)

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host (`localhost` inside a pod) |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `pos` | Database username |
| `DB_PASSWORD` | (required) | Database password |
| `DB_NAME` | `retail_pos` | Database name |
| `DB_SSLMODE` | `disable` dev / `require` prod | libpq TLS mode |
| `PORT` | `8080` | Container listen port |
| `ENV` | `development` | `production` enables the strict defaults |
| `LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |
| `CORS_ORIGIN` | dev origin | Single browser origin |
| `COOKIE_SECURE` | `false` | Must be `true` in production |
| `GIN_MODE` | `debug` | `release` in production |

### Postgres

The official image only understands the `POSTGRES_` names, and defaults both the
role and the database to `postgres` when they are absent. The secret file must
therefore carry **both** spellings, and they must agree:

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_USER` | (falls back to `postgres`) | DB user — keep equal to `DB_USER` |
| `POSTGRES_PASSWORD` | (required) | DB password — keep equal to `DB_PASSWORD` |
| `POSTGRES_DB` | (falls back to `POSTGRES_USER`) | DB name — keep equal to `DB_NAME` |

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
# Check what's using ports 5173/8080/5432
sudo ss -tulpn | grep -E ':5173|:8080|:5432'

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

The stack publishes 5173 (frontend), 8080 (backend API) and 5432 (database, on
the Podman paths). Only 5173 needs to be reachable by users; 8080 should be
reachable only from whatever terminates TLS in front of it.

```bash
# Allow only necessary ports
sudo firewall-cmd --permanent --add-port=5173/tcp
sudo firewall-cmd --permanent --add-port=8080/tcp
# Only if you need host-side psql / the seed target:
sudo firewall-cmd --permanent --add-port=5432/tcp
sudo firewall-cmd --reload
```

Do not open 80/443 expecting this stack to answer — nothing listens on them. If
you front it with a TLS-terminating proxy, open 443 for that proxy instead.

### 4. Regular Updates

```bash
# Update images regularly. Match the major version the stack is deployed on
# (18); pulling an older major against a live data directory is not an update.
podman pull postgres:18-alpine
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
