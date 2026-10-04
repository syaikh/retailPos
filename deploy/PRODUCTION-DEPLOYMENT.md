# Retail POS System - Production Deployment Guide (Podman)

## Overview

This guide covers deploying the Retail POS System in production using **Podman containers** with a **pod architecture**. The system consists of:

- **Nginx** (port 8081, published as 5173): Serves static frontend and reverse proxies API/WebSocket
- **Go Backend** (port 8080, published as 127.0.0.1:8080): REST API + WebSocket server
- **PostgreSQL** (port 5432, published as 127.0.0.1:5432 on the Podman paths only): Database

All three containers run in a **single Podman pod** with shared network namespace.

Only 5173 is published on a routable interface, because a browser on the store
LAN is the frontend's real client. 8080 and 5432 are bound to `127.0.0.1` and are
unreachable from the network; `internal/config/deploy_test.go` fails the build if
that changes.

There is no TLS listener in these images. `deploy/nginx/default.conf.template` has exactly
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
│  ├─ Container: Nginx (8081 → published 5173)                │
│  ├─ Container: Backend (8080 → published 127.0.0.1:8080)    │
│  └─ Container: Postgres (5432 → published 127.0.0.1:5432)   │
└─────────────────────────────────────────────────────┘

Network flow:
  [Client] → Nginx (5173)
              ├─ / → serves static files (frontend)
              ├─ /api/ → proxies to Backend (localhost:8080)
              └─ /ws → upgrades to WebSocket (Backend)
```

Postgres and the API are published on `127.0.0.1` only. The backend reaches
postgres over the pod's own network namespace, so no host port is needed for the
running system; the loopback listener exists because the script's `seed` target
and host-side `psql` connect from the host.

Earlier revisions published both as bare `PORT:PORT`, which podman binds to
`0.0.0.0`. That put the database and the unauthenticated API in reach of anything
that could route to the machine, over a link that `DB_SSLMODE=disable` then
carries in cleartext. The compose path publishes neither port at all.

---

## Prerequisites

### 1. System Requirements

- **OS:** Linux (tested on Fedora, RHEL, CentOS, Ubuntu)
- **Podman:** v5.x (install: `sudo dnf install podman` or `sudo apt install podman`). Quadlet requires 4.4+, and `podman generate systemd` was removed in 5.0, so anything older cannot use the boot-time units in `deploy/quadlet/`.
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

# Install the frontend build dependencies once. web/dist is gitignored and the
# frontend Dockerfile only COPYs it, so the bundle has to be built on this host.
(cd web && npm ci)

# Build both images. The script rebuilds web/dist whenever web/src is newer than
# it, then packages that bundle into the frontend image — so never call
# `podman build -f deploy/frontend/Dockerfile` directly, it will happily package
# a stale dist and report success.
./deploy/podman-deploy.sh build

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

The three values above are all `podman-deploy.sh` needs: it derives
`POSTGRES_USER`/`POSTGRES_DB` from `DB_USER`/`DB_NAME` and reuses `DB_PASSWORD`
as `POSTGRES_PASSWORD`. **`docker-compose.yml` and `deploy/quadlet/` do not** —
they pass the secret file straight to the database container, which only
understands the `POSTGRES_` names and otherwise falls back to a role and
database both called `postgres`. If you use either of those, add these three
lines to the same file:

```bash
POSTGRES_USER=pos
POSTGRES_DB=retail_pos
POSTGRES_PASSWORD=<same value as DB_PASSWORD>
```

Non-secret settings (ports, image tags, `CORS_ORIGIN`, `DB_USER`) can be
overridden either by exporting them in the shell before running the script or by
adding them to the same file. `deploy/.env.example` documents every supported
key and its default.

`podman-deploy.sh` validates this file before creating any container and reports
all missing values at once, so a misconfiguration fails immediately instead of
producing a backend crash loop.

### Step 3: Deploy with Script

The stock `postgres:18-alpine` image ships `ssl=off` while the backend defaults
to `DB_SSLMODE=require` in production, so decide now which posture you want (see
[Database TLS](#database-tls)):

```bash
# Single-host: database reachable only on 127.0.0.1 inside the pod network.
echo 'DB_SSLMODE=disable' | sudo tee -a /etc/retail-pos/backend.env

# ...or, once a certificate and key are mounted on the database container:
# echo 'DB_SSLMODE=require' | sudo tee -a /etc/retail-pos/backend.env
```

`podman-deploy.sh` runs `SHOW ssl` after the database starts and refuses to
continue if the database cannot satisfy the configured mode.

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

Open browser: **http://your-server-ip:5173** (the frontend is published on host
port `5173`; `HOST_FRONTEND_PORT` changes it).

Bootstrap login credentials:
- Username: `superadmin`
- Password: `admin123`

**These are bootstrap credentials, not production credentials.** `000_baseline.sql`
flags every seeded account (`superadmin`, `manager`, `supervisor`,
`cashier`, `inventory_staff`, `finance`) with `must_change_password`, so the first login is forced through
a password rotation — the backend answers HTTP 428 for every protected call
until it is done. Follow
[`docs/guides/first-time-installation.md`](../docs/guides/first-time-installation.md)
for the full first-run flow (migrate → rotate → finish the Default Store →
verify readiness).

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

Migrations are SQL files in `database/migrations/`: `000_baseline.sql` (the squashed Version 1 baseline) plus `054`–`060`. The 32 migrations the baseline replaces live in `database/migrations/archive/pre-squash-migrations.tar.gz` — never execute an archived file against a live database. They are **not** run automatically by the backend server — you must run them explicitly:

```bash
./deploy/podman-deploy.sh migrate   # applies every *.sql in database/migrations/
```

On a **fresh database** (or a fresh Postgres container), `migrate` now bootstraps the three prerequisites that `000_baseline.sql` depends on before applying any migration:

1. `CREATE EXTENSION IF NOT EXISTS pgcrypto`
2. `CREATE SEQUENCE IF NOT EXISTS invoice_seq START 1`
3. `CREATE TABLE IF NOT EXISTS schema_migrations (...)` (tracks applied files)

It then applies each migration in sorted filename order with `ON_ERROR_STOP=1` and records each applied file in `schema_migrations`. **Every migration must therefore be permanently re-runnable** (`IF NOT EXISTS`, `DO`-guarded constraints, `ON CONFLICT DO NOTHING`, `DROP … IF EXISTS`) — the runner never consults the ledger before applying a file. `000_baseline.sql` is idempotent throughout and, as its final step, replaces the ledger rows of the 32 migrations it superseded with its own, so after a fresh baseline install `schema_migrations` holds one row per migration file.

`059_store_fk_integrity.sql` adds four `store_id` foreign keys and **validates existing rows**, so it aborts on the first orphan. Run `./scripts/audit-store-fk-orphans.sh` first; if it reports rows, remap them instead of weakening the migration.

Migrations produce the full schema plus reference data: roles (6), permissions (91), role grants (272), the `superadmin`/`manager`/`supervisor`/`cashier`/`inventory_staff`/`finance` users (all flagged for forced first-login password rotation), payment methods, and customer groups (Walk-in/Member/VIP). They also seed a placeholder **Default Store** (with those users assigned to it) — but **no products, customers, or sales**. Finish the first store per [`docs/guides/first-time-installation.md`](../docs/guides/first-time-installation.md); run `./deploy/podman-deploy.sh seed` only when you want dummy/demo business data.

> **Important:** Apply migrations **before** deploying a new server binary — see the `AGENTS.md` "Deployment" section.

---

## Two-Host Deployment (frontend separate from backend + DB)

The frontend image is parameterised for this: nginx reads `BACKEND_HOST` and
`BACKEND_PORT` from the container environment and substitutes them into its
upstream at startup, so the same image works all-in-one or split. Changing the
address needs no rebuild.

The recommended split keeps the stateful pair together:

| Host | Runs |
|------|------|
| Frontend host (internet-facing) | nginx frontend only |
| Backend host (private) | backend + PostgreSQL |

Postgres stays bound to `127.0.0.1` exactly as it is today — no certificate to
mount, no `5432` exposed, and `DB_SSLMODE=require` is satisfied over loopback —
while backend↔DB latency stays local. Only the static frontend faces the network.

Start the backend host as usual (its own `BACKEND_HOST` is irrelevant):

```bash
# backend host: database + API
./deploy/podman-deploy.sh start postgres
./deploy/podman-deploy.sh start backend
# The backend is not published by default. Add a port forward scoped to the
# frontend host's address, e.g. append -p 10.0.0.5:8080:8080 when creating the pod.
```

Run only the frontend image on the frontend host, pointed at the backend host:

```bash
podman run -d --name frontend -p 5173:8081 \
  -e BACKEND_HOST=10.0.0.5 \
  -e BACKEND_PORT=8080 \
  localhost/retail-pos-frontend:latest
```

Then set the backend's `CORS_ORIGIN` to the frontend's public origin and terminate
TLS on the frontend host (see [SSL/TLS Configuration](#ssltls-configuration-https)).

> **Note:** the default pod forward binds the backend to `127.0.0.1`, which
> another machine cannot reach. Publish it on the private interface and scope a
> host firewall rule to the frontend host only. Do **not** attempt to split the
> database onto a third host unless you have outgrown one DB machine: that
> requires enabling TLS on Postgres and exposing `5432`, which the co-located
> layout avoids.

---

## Manual Deployment (without script)

Equivalent to what `podman-deploy.sh start` does, for when you want explicit
control. Note the required settings — this path is easy to get wrong, which is
why the script exists. See [Database TLS](#database-tls) for `DB_SSLMODE`.

```bash
# 0. Secret file must already exist (see Step 2)
sudo test -r /etc/retail-pos/backend.env || { echo "missing secret file"; exit 1; }

# 1. Create pod with ports
#    5173→8081, not 80→80: the nginx template has a single `listen 8081;`. Publishing
#    80/443 reaches nothing (this is the defect the retired systemd unit had).
#    8080 and 5432 are bound to 127.0.0.1, never 0.0.0.0: a bare `-p 8080:8080`
#    makes them reachable from anywhere that can route to this machine. 5432 is
#    published at all only so host-side psql/seed can connect.
podman pod create --name retail-pos-pod -p 5173:8081 -p 127.0.0.1:8080:8080 -p 127.0.0.1:5432:5432

# 2. Create persistent volumes
podman volume create retail-pos-postgres-data
# Store logo uploads, mounted at /app/uploads by the backend below. podman-deploy.sh
# creates and mounts this for you; create it by hand only for this manual path and
# for Quadlet, neither of which runs that script.
podman volume create retail-pos-uploads

# 3. Start PostgreSQL container
#    POSTGRES_PASSWORD is passed without a value so it is read from the exported
#    environment rather than appearing in the process table. Sourcing has to
#    happen in *your* shell: `sudo -E . file` would look for a command named "."
#    and, even if it worked, exports in a subshell never reach this one.
set -a; . /etc/retail-pos/backend.env; set +a
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
  -v retail-pos-uploads:/app/uploads \
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

# 5. Start frontend. The image defaults to BACKEND_HOST=127.0.0.1:8080, which is
#    correct inside this pod; pass them explicitly only for a split deployment
#    (see Two-Host Deployment).
podman run -d \
  --pod retail-pos-pod \
  --name frontend \
  -e BACKEND_HOST=127.0.0.1 \
  -e BACKEND_PORT=8080 \
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
# Build images first (same staleness-checked build the podman script uses)
./deploy/podman-deploy.sh build

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
and 443 while the nginx template only listens on 8081; and its `Documentation=` URL
still pointed at `github.com/your-repo`. `Documentation=https://github.com/your-repo/retail-pos-system`
was the least of its problems. See Recommendation 7 in
`docs/audits/production-deploy-config-audit-2026-09-26.md`.

---

## SSL/TLS Configuration (HTTPS)

`deploy/nginx/default.conf.template` has a single `listen 8081;` and **no TLS server block**,
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
# see deploy/nginx/default.conf.template for the upstream path; proxy_pass to
# 127.0.0.1:5173 and proxy /api to 127.0.0.1:8080, including the
# Upgrade/Connection headers the WebSocket handler needs.

# Then set CORS_ORIGIN to the public origin in the secret file and restart.
systemctl --user restart retail-pos-backend.service
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

- **Backend:** `curl http://localhost:8080/health` (no auth; there is no `/api/stats` route)
- **Frontend:** `curl http://localhost:5173/` should return HTML (nginx also proxies `/health`)
- **Database:** `podman exec postgres pg_isready -U pos`

Container names depend on how you deployed: `podman-deploy.sh` creates
`postgres`, `backend`, and `frontend`, while the Quadlet units in
`deploy/quadlet/` create `retail-pos-postgres`, `retail-pos-backend`, and
`retail-pos-frontend`. The commands below assume the script.

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
podman stop backend

# Restore
zcat backup_20260429.sql.gz | podman exec -i postgres psql -U pos retail_pos

# Restart backend
podman start backend
```

### Backup Volume

```bash
# Stop services
podman pod stop retail-pos-pod

# Backup volume
podman volume export retail-pos-postgres-data > postgres-volume.tar

# Restore volume
podman volume import retail-pos-postgres-data postgres-volume.tar

# Uploads (store logo) is a separate named volume and is NOT in the database
# backup above. Back it up too, or the shop's logo is lost on a bare restore.
podman volume export retail-pos-uploads > uploads-volume.tar

# Restore volume
podman volume import retail-pos-uploads uploads-volume.tar
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
podman logs backend
podman logs frontend
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
podman exec frontend cat /etc/nginx/conf.d/default.conf
```

### "Network error. Please try again" on login

This indicates frontend cannot reach backend. Fix:

```bash
# Ensure backend is running
podman ps | grep backend

# Test API directly
curl http://localhost:8080/health

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

# Configure nginx upstream (in deploy/nginx/default.conf.template, via BACKEND_HOST)
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

The backend image runs as `retailpos` and the frontend image as `nginx`
(both UID 1000). The `postgres:18-alpine` container runs as its own
`postgres` user.

### 2. Secrets Management

Secrets belong in `/etc/retail-pos/backend.env` (mode `600`), which is what
`podman-deploy.sh`, `docker-compose.yml`, and the Quadlet units all read — the
backend has **no** `DB_PASSWORD_FILE` support, so the older
`/run/secrets/db_password` recipe described in
[the retired-unit post-mortem](#if-you-are-migrating-from-the-old-retail-posservice)
never worked and must not be reintroduced.

Understand the limit of `--env-file`/`env_file`: it keeps values out of `argv`
and the process table, but they are still readable by anyone who can run
`podman inspect backend`. Keep the file mode `600` and treat the host as the
trust boundary; if that is not enough isolation, move to Podman secrets or an
external secret manager.

### 3. Firewall Configuration

Only **5173** is published on all interfaces. 8080 and 5432 are bound to
`127.0.0.1` (and `docker-compose.yml` publishes neither), so a reverse proxy on
the same host reaches the backend over loopback and no firewall rule is needed
for it — adding one for 8080/5432 would expose them to the network anyway.

```bash
# The only port that needs to be open
sudo firewall-cmd --permanent --add-port=5173/tcp
sudo firewall-cmd --reload
```

Do not open 80/443 expecting this stack to answer — nothing listens on them. If
you front it with a TLS-terminating proxy, open 443 for that proxy instead. If
you ever republish 8080 or 5432 on a routable interface, that host is outside
the design this repository assumes and `DB_SSLMODE=disable` (the single-host
default) stops being appropriate.

### 4. Regular Updates

```bash
# Update images regularly. Match the major version the stack is deployed on
# (18); pulling an older major against a live data directory is not an update.
podman pull postgres:18-alpine
./deploy/podman-deploy.sh build

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

# Remove volumes (WARNING: deletes all data, including the store logo)
podman volume rm retail-pos-postgres-data retail-pos-uploads
```

There is no separate network to remove: the three containers share the pod's
own network namespace (`--network bridge` for the pod), and `podman-deploy.sh`
never creates a `retail-pos-network`.

---

## Static frontend hosting

The frontend is a Vite build served by nginx; there is no Python static server
in the current stack. If you are migrating from an older `python3 -m
http.server` deployment:

1. **Drop the Python server** – nginx (`deploy/nginx/default.conf.template`) serves the built assets directly
2. **Single command deployment** – `./deploy/podman-deploy.sh start`
3. **Auto-start on boot** – Quadlet user units in `deploy/quadlet/`
4. **Better performance** – Nginx > Python HTTP server
5. **HTTPS ready** – Just add SSL certs

---

## Next Steps

- [ ] Set up SSL certificates with Let's Encrypt
- [ ] Configure log rotation (journald + logrotate)
- [ ] Set up monitoring (`GET /metrics` exposes EventBus and report-refresh counters as JSON — scrape it or wrap it in an exporter; it is not Prometheus text format)
- [ ] Add automated backups (cron job for pg_dump)
- [ ] Deploy to multiple servers with load balancer
- [ ] CI/CD pipeline for automatic image builds

## Related guides

These are separate documents because they are procedures in their own right,
not part of the main deployment flow:

- [Quadlet smoke test](../../docs/guides/quadlet-smoke-test.md) — proving the
  `deploy/quadlet/` units actually start and stay up. Read this if you are using
  the systemd path; ignore it if you use `./deploy/podman-deploy.sh`.
- [Print agent production install](../../docs/guides/print-agent-production.md) —
  installing `tools/print-agent` on a till, per-printer transports, and its
  security model. The agent is a host service, not part of the pod.

---

## Support

For issues, check:
- Logs: `./deploy/podman-deploy.sh logs`
- Systemd (Quadlet): `journalctl --user -u retail-pos-backend -f` (drop `--user` on a rootful host)
- Podman: `podman pod ps` and `podman ps -a`

Full documentation: see [README.md](../../README.md), plus
[docs/guides/](../../docs/guides/) for the print agent, Quadlet smoke test, and
first-time installation.
