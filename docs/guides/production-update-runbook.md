# Production Update Runbook (routine code release)

Audience: **someone who is not expected to know the codebase.** Follow the
steps in order, on the machine that runs the shop's server. Every command is
copy-pasteable.

Use this runbook whenever new code has been pushed to `master` and the live
shop must pick it up. It uses the documented deployment path
(`deploy/podman-deploy.sh`, see [PRODUCTION-DEPLOYMENT.md](../../deploy/PRODUCTION-DEPLOYMENT.md)).

> **No database changes? No frontend changes?** Then this is a one-command
> update: step 3's `git log` check tells you. If `database/` or `web/` changed
> in the release, stop and read [Appendix A](#appendix-a-when-the-release-also-changes-database-or-frontend)
> first — the extra steps matter.

---

## Before you start

| You need | Why |
|---|---|
| SSH access to the production server | That is where the update runs |
| The project folder on that server (a `git` clone) | `git pull` downloads the new code |
| The stack already running (`./deploy/podman-deploy.sh status` shows three containers) | This is an *update*, not a first install. For a fresh machine follow [`docs/guides/first-time-installation.md`](first-time-installation.md) instead |

**Language cheat-sheet** (skip if these are familiar):

- **Commit** — a saved version of the code, named like `73975409`. The server
  must be on the same commits as GitHub before you rebuild.
- **Image** — a packaged, runnable copy of the app. `podman build` makes a new
  one from the freshly pulled code; containers are started from images.
- **Container** — the running program. Replacing the `backend` container is what
  applies the update.
- **Migration** — a numbered SQL file that upgrades the database schema. The
  deploy script runs them automatically *before* starting a new binary.

If a command needs administrator rights, prefix it with `sudo` (rootful Podman).
If `./deploy/podman-deploy.sh status` works without errors today, you do not
need `sudo`.

---

## Step 1 — Log in and go to the project folder

```bash
ssh your-user@your-server
cd /path/to/retail-pos-system      # the folder you cloned from GitHub
```

## Step 2 — Check the server is healthy *before* you touch it

```bash
./deploy/podman-deploy.sh status
curl -fsS http://localhost:8080/health && echo " backend OK"
```

Both must look good. If the stack is already broken, fix that first — an update
on top of a broken deployment makes the cause harder to identify.

## Step 3 — Pull the new code

```bash
git pull
git log --oneline -3
```

`git log` must show the new commits at the top, matching GitHub. If it shows
something else, stop: the clone may be on the wrong branch, detached, or unable
to reach GitHub.

Identify which of the two situations you are in:

```bash
git diff --name-only @{u}~2..@{u} | cut -d/ -f1 | sort -u
```

- Output is only `docs` and `internal` (typical) → **no database or frontend
  changes**. Continue to Step 4.
- Output contains `database` or `web` → also read
  [Appendix A](#appendix-a-when-the-release-also-changes-database-or-frontend).

## Step 4 — Take a safety net (30 seconds)

```bash
mkdir -p backups
podman exec postgres pg_dump -U pos retail_pos | gzip > backups/pre_deploy_$(date +%Y%m%d_%H%M).sql.gz
podman tag localhost/retail-pos-backend:latest localhost/retail-pos-backend:rollback
```

- The first line compresses a full copy of the shop's data into `backups/`.
  You almost certainly will not need it, but it is the only way back if
  something unexpected happens.
- The second line makes a spare name (`rollback`) for the image currently in
  use, so going back is instant. `podman build` overwrites `:latest`.

## Step 5 — Run the update

```bash
./deploy/podman-deploy.sh start backend
```

That single command is the whole update. It:

1. checks the configuration and the database TLS mode;
2. runs migrations **first** (if none changed, this is a fast, harmless re-run —
   every migration is written to be safely re-runnable);
3. builds a new backend image from the code you just pulled;
4. stops and removes the old `backend` container, starts the new one, and
   waits until it answers `/health`.

**What you should see:** a few `log_info` lines ending with something like
`Backend is healthy`. During the swap the API is unreachable for a few seconds —
that is normal and expected.

**Do not run** `stop`, `restart`, or `build frontend` for a routine update.
The database keeps running untouched, and the frontend bundle is only rebuilt
when `web/` actually changed (the script reports `web/dist is up to date`).

## Step 6 — Verify

```bash
./deploy/podman-deploy.sh status
curl -fsS http://localhost:8080/health && echo " backend OK"
curl -fsS -o /dev/null -w '%{http_code}\n' http://localhost:8000/
./deploy/podman-deploy.sh logs backend | tail -30
```

Expected: all three containers `running`, `backend OK`, HTTP `200`, and log
lines with **no** repeating `error`/`panic`.

Then two quick checks in the browser — they cover the code that actually
changed in this release:

1. **Dashboard** opens and shows today's totals (the revenue query moved to
   another module).
2. **Consignment → Add term products**: type a product name; matching products
   still appear (that search now runs through a port instead of direct SQL).

**If any check fails** → go to [Rollback](#rollback).

## Step 7 — Done

Leave the old image and backup in place for a week:

- `localhost/retail-pos-backend:rollback` — instant image rollback,
- `backups/pre_deploy_*.sql.gz` — database backup.

Then delete them when you are confident:

```bash
podman rmi localhost/retail-pos-backend:rollback
rm backups/pre_deploy_*.sql.gz
```

---

## Rollback

Use this **only** when Step 6 fails. It restores the previous binary; the
database is not touched (there were no schema changes to undo).

```bash
podman stop backend && podman rm backend
podman tag localhost/retail-pos-backend:rollback localhost/retail-pos-backend:latest
./deploy/podman-deploy.sh start backend
```

No `rollback` image (for example after a server reboot cleared tags)? Roll the
code back instead:

```bash
git checkout 4dda0dcb            # the commit before the update
./deploy/podman-deploy.sh start backend
git checkout master              # return the clone to master when finished
```

Restoring data is a last resort, and requires the backend stopped:

```bash
podman stop backend
zcat backups/pre_deploy_YYYYMMDD_HHMM.sql.gz | podman exec -i postgres psql -U pos retail_pos
podman start backend
```

---

## Appendix A: When the release also changes database or frontend

| Changed folder | What to do extra |
|---|---|
| `database/` | Nothing manual — `start backend` runs migrations before the new binary, and aborts rather than starting if a migration fails. Read the migration's entry in `AGENTS.md` for any prerequisite (e.g. `059` needs `scripts/audit-store-fk-orphans.sh` to pass first). |
| `web/` | Nothing manual — `build_image frontend` notices `web/dist` is stale and runs `npm run build` for you. It needs `npm` and `web/node_modules` on the server (`(cd web && npm ci)` once). Then run `./deploy/podman-deploy.sh start frontend` as well. |
| both | Do `start backend` first (migrations gate the binary), then `start frontend`. |

**Order always matters:** migrations → new binary. The script enforces this;
never start a new backend by hand against an unmigrated database.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `Migrations failed; refusing to start a new backend binary...` | A migration aborted. The old binary is still running — the shop is up on the old version. | Read the printed error. Fix the reported data/schema problem, then re-run `./deploy/podman-deploy.sh start backend`. |
| `POSTGRES_PASSWORD is not set` | `/etc/retail-pos/backend.env` (or the env file your deploy uses) is missing/unreadable | Restore the secret file (mode `600`, owned by the user running Podman) |
| Backend container restarts in a loop | Backend cannot reach the database, or `JWT_SECRET` missing | `./deploy/podman-deploy.sh logs backend`, then `podman exec postgres pg_isready -u pos` |
| `web/dist is stale or missing and npm was not found` | Frontend changed but Node is not installed on the server | Build the bundle where Node exists: `(cd web && npm ci && npm run build)`, or install Node on the server |
| `port is already allocated` | Another process/pod holds 8000/8080/5432 | `scripts/kill-port.sh <port>` or see PRODUCTION-DEPLOYMENT.md → Troubleshooting |
| Frontend shows a blank page after update | Stale cached bundle or nginx serving old files | `./deploy/podman-deploy.sh restart frontend`; hard-reload the browser (Ctrl+Shift+R) |

---

## Related guides

- [Production Deployment Guide (Podman)](../../deploy/PRODUCTION-DEPLOYMENT.md) — architecture, TLS, backups, Quadlet
- [First-Time Installation Runbook](first-time-installation.md) — fresh installs only
- `AGENTS.md` → *Deployment / Migration Ordering* — which migration does what
