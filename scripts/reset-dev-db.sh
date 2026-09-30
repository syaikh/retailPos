#!/usr/bin/env bash
# Reset the DEVELOPMENT database to exact first-install state.
#
# Re-migrating cannot do this: database/migrations/000_baseline.sql is purely
# additive (123 ON CONFLICT guards, forward-only setval, and a single DELETE that
# touches only schema_migrations). Replaying it over a drifted database leaves
# every extra store, user, role, grant, and sale in place. A true fresh install
# therefore means drop -> create -> bootstrap -> replay the same migrations a new
# deployment would, then verify the result.
#
# Scope is deliberately narrow. This script refuses to run against anything but
# the postgres-dev container's retail_pos database; the production volumes
# (retail-pos-postgres-data, pgdata_retail) are never named. See
# docs/design/dev-db-fresh-install-reset-plan.md.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATION_DIR="$ROOT/database/migrations"
UPLOAD_DIR="$ROOT/uploads/logos"
E2E_TOKEN_CACHE="/tmp/retail-pos-e2e-tokens.v1.json"

ALLOW_ASSUME_YES=0
KEEP_UPLOADS=0
RESEED=0

die() { printf 'reset-dev-db: %s\n' "$1" >&2; exit 1; }
info() { printf '\n== %s\n' "$1"; }

usage() {
  cat <<'EOF'
Usage: scripts/reset-dev-db.sh [options]

Drops the development database and rebuilds it from database/migrations/*.sql,
leaving the app in exact first-install state.

Options:
  --yes           Skip the interactive RESET confirmation
  --keep-uploads  Do not clear uploads/logos/
  --reseed        Run ./seed-dev.sh afterwards (NOT a fresh install — the
                  seeder recreates stores/payment_methods/customer_groups with
                  new ids, so stores(id=1) and customers(id=1) drift off the
                  baseline invariant)
  -h, --help      Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --yes|-y)       ALLOW_ASSUME_YES=1 ;;
    --keep-uploads) KEEP_UPLOADS=1 ;;
    --reseed)       RESEED=1 ;;
    -h|--help)      usage; exit 0 ;;
    *)              usage >&2; die "unknown option: $1" ;;
  esac
  shift
done

# --------------------------------------------------------------------------
# Configuration
#
# Shell exports win over .env, matching godotenv.Load in cmd/server/main.go:44
# so this script sees the same values the server would.
# --------------------------------------------------------------------------
load_dotenv() {
  local file="$ROOT/.env" line key value
  if [[ ! -f "$file" ]]; then
    return 0
  fi
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%%#*}"
    line="${line#"${line%%[![:space:]]*}"}"
    if [[ -z "$line" || "$line" != *=* ]]; then
      continue
    fi
    key="${line%%=*}"
    value="${line#*=}"
    key="${key%"${key##*[![:space:]]}"}"
    value="${value#\"}"; value="${value%\"}"
    value="${value#\'}"; value="${value%\'}"
    if [[ ! "$key" =~ ^[A-Za-z_][A-Za-z0-9_]* ]] || [[ -n "${!key+x}" ]]; then
      continue
    fi
    export "$key=$value"
  done < "$file"
}
load_dotenv

DB_HOST="${DB_HOST:-localhost}"
DB_USER="${DB_USER:-pos}"
DB_NAME="${DB_NAME:-retail_pos}"
DB_PORT="${DB_PORT:-${DATABASE_PORT:-5432}}"
DB_PASSWORD="${DB_PASSWORD:-}"
export PGPASSWORD="$DB_PASSWORD"

if [[ -n "${DATABASE_PORT:-}" && "$DATABASE_PORT" != "$DB_PORT" ]]; then
  printf 'reset-dev-db: warning: DATABASE_PORT=%s differs from DB_PORT=%s; using DB_PORT\n' \
    "$DATABASE_PORT" "$DB_PORT"
fi

# --------------------------------------------------------------------------
# Guards — hard refusal, no override
# --------------------------------------------------------------------------
if [[ "${ENV:-development}" == "production" ]]; then
  die "refusing to run with ENV=production"
fi
if [[ "$DB_HOST" != "localhost" && "$DB_HOST" != "127.0.0.1" ]]; then
  die "refusing to run against non-localhost DB_HOST=$DB_HOST"
fi
if [[ "$DB_NAME" != "retail_pos" ]]; then
  die "refusing to touch database '$DB_NAME' (expected 'retail_pos')"
fi

if command -v podman >/dev/null 2>&1; then
  serving="$(podman ps --format '{{.Names}} {{.Ports}}' 2>/dev/null \
    | awk -v p=":$DB_PORT->" 'index($0, p) {print $1; exit}')"
  if [[ -z "$serving" ]]; then
    die "no container publishes port $DB_PORT — is postgres-dev running?"
  fi
  if [[ "$serving" != "postgres-dev" ]]; then
    die "port $DB_PORT is served by container '$serving', not 'postgres-dev'"
  fi
else
  # The localhost + retail_pos guards above still bound the blast radius, so this
  # is a warning rather than a refusal: refusing would break a legitimate
  # podman-less setup (a native postgres on localhost:5433, a rootless shell with
  # a minimal PATH). The trade-off is documented in AGENTS.md — the container
  # identity check is best-effort when podman is not on PATH.
  printf 'reset-dev-db: warning: podman not found; skipped the container identity check.\n' >&2
  printf 'reset-dev-db: continuing on the localhost + database-name guards alone.\n' >&2
fi

if ! command -v psql >/dev/null 2>&1; then
  die "psql not found on PATH"
fi
if ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -q; then
  die "no PostgreSQL server reachable on $DB_HOST:$DB_PORT"
fi

psql_q() { psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "${2:-$DB_NAME}" -v ON_ERROR_STOP=1 -tAq -c "$1"; }

# psql_q_bare is the pre-migration report helper: the tables it reads only exist
# once a migration has run, so failures are non-fatal by design.
psql_q_bare() { psql_q "$1" 2>/dev/null || true; }

# --------------------------------------------------------------------------
# Preflight — record what is about to be destroyed, and capture the properties
# the recreated database must inherit.
# --------------------------------------------------------------------------
info "Current state of $DB_NAME on $DB_HOST:$DB_PORT (this will be destroyed)"

psql_q_bare "SELECT format('%s|%s', label, n) FROM (
               SELECT 'schema_migrations' AS label, count(*) AS n FROM schema_migrations
               UNION ALL SELECT 'stores',            count(*) FROM stores
               UNION ALL SELECT 'users',             count(*) FROM users
               UNION ALL SELECT 'roles',             count(*) FROM roles
               UNION ALL SELECT 'permissions',       count(*) FROM permissions
               UNION ALL SELECT 'role_permissions',  count(*) FROM role_permissions
               UNION ALL SELECT 'customer_groups',   count(*) FROM customer_groups
               UNION ALL SELECT 'payment_methods',   count(*) FROM payment_methods
               UNION ALL SELECT 'app_settings',      count(*) FROM app_settings
               UNION ALL SELECT 'products',          count(*) FROM products
               UNION ALL SELECT 'customers',         count(*) FROM customers
               UNION ALL SELECT 'sales',             count(*) FROM sales
               UNION ALL SELECT 'sale_items',        count(*) FROM sale_items
               UNION ALL SELECT 'shifts',            count(*) FROM shifts
               UNION ALL SELECT 'audit_logs',        count(*) FROM audit_logs
               UNION ALL SELECT 'import_jobs',       count(*) FROM import_jobs
               UNION ALL SELECT 'dead_letter_events',count(*) FROM dead_letter_events
               UNION ALL SELECT 'refresh_tokens',    count(*) FROM refresh_tokens
             ) t ORDER BY label;" \
  | while IFS='|' read -r label n; do printf '  %-22s %s\n' "$label" "$n"; done

# Owner, encoding, and collation are inherited from the database being replaced
# so a reset never silently changes sort order or file permissions. If it does
# not exist yet, template1 provides the same answer.
if exists="$(psql_q "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME';" postgres)" \
   && [[ -n "$exists" ]]; then
  src_db="$DB_NAME"
else
  src_db="template1"
  printf 'reset-dev-db: %s does not exist; using %s defaults.\n' "$DB_NAME" "$src_db"
fi
db_owner="$(psql_q "SELECT pg_get_userbyid(datdba) FROM pg_database WHERE datname = '$src_db';" postgres)"
db_enc="$(psql_q "SELECT pg_encoding_to_char(encoding) FROM pg_database WHERE datname = '$src_db';" postgres)"
db_coll="$(psql_q "SELECT datcollate FROM pg_database WHERE datname = '$src_db';" postgres)"
db_ctype="$(psql_q "SELECT datctype FROM pg_database WHERE datname = '$src_db';" postgres)"

if [[ "$ALLOW_ASSUME_YES" -ne 1 ]]; then
  printf '\nThis permanently deletes every row in %s.\n' "$DB_NAME"
  printf 'Type RESET to continue: '
  # `read` returns non-zero on EOF, which under `set -e` would abort the script
  # with no message at all — the silent case being `make db-fresh < /dev/null` or
  # any piped/CI invocation. Fails safe either way, but say why.
  if ! read -r reply; then
    die "aborted (no input available; pass --yes to skip this prompt)"
  fi
  if [[ "$reply" != "RESET" ]]; then
    die "aborted"
  fi
fi

# --------------------------------------------------------------------------
# 1. Drop
#
# WITH (FORCE) (PostgreSQL 13+) evicts the running backend's pooled connections
# instead of failing on them. Sibling databases — retail_pos_test,
# retail_pos_seeder_test, pos_squash_verify — are never named.
# --------------------------------------------------------------------------
info "Dropping $DB_NAME"
psql_q "DROP DATABASE IF EXISTS \"$DB_NAME\" WITH (FORCE);" postgres

# --------------------------------------------------------------------------
# 2. Recreate
# --------------------------------------------------------------------------
info "Creating $DB_NAME (owner=$db_owner encoding=$db_enc collation=$db_coll)"
psql_q "CREATE DATABASE \"$DB_NAME\" OWNER \"$db_owner\" ENCODING '$db_enc'
        LC_COLLATE '$db_coll' LC_CTYPE '$db_ctype' TEMPLATE template0;" postgres

# --------------------------------------------------------------------------
# 3. Bootstrap — mirrors deploy/podman-deploy.sh:350-359. The baseline also
#    creates all three of these itself (:46, :50, :907, :1320), so this is
#    parity with the production runner rather than a requirement.
# --------------------------------------------------------------------------
info "Bootstrapping SQL prerequisites"
psql_q "CREATE EXTENSION IF NOT EXISTS pgcrypto;
        CREATE SEQUENCE IF NOT EXISTS invoice_seq START 1;
        CREATE TABLE IF NOT EXISTS schema_migrations (
          filename varchar(255) PRIMARY KEY,
          applied_at timestamptz NOT NULL DEFAULT now()
        );" >/dev/null

# --------------------------------------------------------------------------
# 4. Migrate — lexical order, non-recursive so archive/ is never touched.
#    Replays every file rather than consulting the ledger, matching all three
#    non-test runners and the replay contract in AGENTS.md.
# --------------------------------------------------------------------------
info "Applying migrations from database/migrations/*.sql"
shopt -s nullglob
migration_files=("$MIGRATION_DIR"/*.sql)
if [[ ${#migration_files[@]} -eq 0 ]]; then
  die "no .sql files found in $MIGRATION_DIR"
fi
applied_names=()
for sql_file in "${migration_files[@]}"; do
  name="$(basename "$sql_file")"
  printf '  -> %s\n' "$name"
  psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 -q -f "$sql_file" >/dev/null
  psql_q "INSERT INTO schema_migrations (filename) VALUES ('$name') ON CONFLICT DO NOTHING;" >/dev/null
  # Recorded as we go so the ledger comparison in step 5 needs no count passed
  # from the shell into SQL. Derived from this array, never hard-coded: today that
  # is one file (000_baseline.sql squashed 32 migrations into one), and it grows by
  # one per file added later (AGENTS.md: "New migrations must start at 054_*.sql").
  applied_names+=("$name")
done
shopt -u nullglob

# --------------------------------------------------------------------------
# 5. Verify
#
# 000_baseline.sql:5138-5211 already ran its self-check inside the migration
# transaction and would have aborted the apply above. Those checks use '<' rather
# than '=' so a database that was already in use still passes; the assertions
# below are the stricter fresh-install shape.
# --------------------------------------------------------------------------
info "Verifying fresh-install state"

# Ledger check, in bash rather than SQL. internal/shared/testdb.go:90-114 asserts
# len(ledger) == len(files); the same invariant is checked here against the array of
# names just applied, so adding 054_*.sql needs no edit to this script.
#
# Comparing the *names* rather than a count is deliberately stricter: a wrong count
# is the only failure a count can see, whereas a set difference also catches a
# ledger holding a stale or misspelled filename that happens to be the right length
# (e.g. an archived pre-squash name). Both directions are reported.
ledger_drift=0
for expected_name in "${applied_names[@]}"; do
  # Compare output, not exit status: a SELECT that matches no rows still exits 0
  # under -tAq, so testing `$?` would treat an absent row as present.
  if [[ "$(psql_q "SELECT 1 FROM schema_migrations WHERE filename = '$expected_name';")" != "1" ]]; then
    printf '  schema_migrations is missing %s\n' "$expected_name" >&2
    ledger_drift=1
  fi
done
# Anything in the ledger that is not a file we just applied is also drift.
while IFS= read -r ledger_name; do
  [[ -n "$ledger_name" ]] || continue
  found=0
  for expected_name in "${applied_names[@]}"; do
    if [[ "$ledger_name" == "$expected_name" ]]; then
      found=1
      break
    fi
  done
  if [[ "$found" -eq 0 ]]; then
    printf '  schema_migrations has unexpected row %s\n' "$ledger_name" >&2
    ledger_drift=1
  fi
done < <(psql_q "SELECT filename FROM schema_migrations ORDER BY filename;")
if [[ "$ledger_drift" -ne 0 ]]; then
  die "fresh-install verification failed: schema_migrations does not match database/migrations/*.sql (${#applied_names[@]} file(s) applied)"
fi
printf '  schema_migrations matches all %d migration file(s)\n' "${#applied_names[@]}"

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -tAq -c "
  SELECT format('%s|%s', label, n) FROM (
    SELECT 'stores'             AS label, (SELECT count(*) FROM stores)             AS n
    UNION ALL SELECT 'users',            (SELECT count(*) FROM users)
    UNION ALL SELECT 'roles',            (SELECT count(*) FROM roles)
    UNION ALL SELECT 'permissions',      (SELECT count(*) FROM permissions)
    UNION ALL SELECT 'role_permissions', (SELECT count(*) FROM role_permissions)
    UNION ALL SELECT 'payment_methods',  (SELECT count(*) FROM payment_methods)
    UNION ALL SELECT 'customer_groups',  (SELECT count(*) FROM customer_groups)
    UNION ALL SELECT 'app_settings',     (SELECT count(*) FROM app_settings)
    UNION ALL SELECT 'schema_migrations',(SELECT count(*) FROM schema_migrations)
  ) t ORDER BY label;" \
  | while IFS='|' read -r label n; do printf '  %-22s %s\n' "$label" "$n"; done

# Quoted heredoc (<<'SQL'), so nothing here is shell-expanded — that is what keeps
# the DO $$ ... $$ dollar-quoting intact. The ledger is compared to the migration
# directory in bash above, precisely so no value has to be passed in here.
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -q <<'SQL'
DO $$
DECLARE
    n   bigint;
    tbl text;
    bad text := '';
BEGIN
    SELECT count(*) INTO n FROM schema_migrations WHERE filename = '000_baseline.sql';
    IF n <> 1 THEN bad := bad || format(' baseline_row=%s/1', n); END IF;

    -- The nine seeded tables, at their exact first-install counts.
    SELECT count(*) INTO n FROM roles;            IF n <> 6   THEN bad := bad || format(' roles=%s/6', n); END IF;
    SELECT count(*) INTO n FROM permissions;      IF n <> 86  THEN bad := bad || format(' permissions=%s/86', n); END IF;
    SELECT count(*) INTO n FROM role_permissions; IF n <> 264 THEN bad := bad || format(' role_permissions=%s/264', n); END IF;
    SELECT count(*) INTO n FROM payment_methods;  IF n <> 5   THEN bad := bad || format(' payment_methods=%s/5', n); END IF;
    SELECT count(*) INTO n FROM customer_groups;  IF n <> 3   THEN bad := bad || format(' customer_groups=%s/3', n); END IF;
    SELECT count(*) INTO n FROM app_settings;     IF n <> 7   THEN bad := bad || format(' app_settings=%s/7', n); END IF;
    SELECT count(*) INTO n FROM stores;           IF n <> 1   THEN bad := bad || format(' stores=%s/1', n); END IF;
    SELECT count(*) INTO n FROM customers;        IF n <> 1   THEN bad := bad || format(' customers=%s/1', n); END IF;
    SELECT count(*) INTO n FROM users;            IF n <> 6   THEN bad := bad || format(' users=%s/6', n); END IF;

    -- The placeholder store and the walk-in customer checkout falls back to.
    SELECT count(*) INTO n FROM stores WHERE id = 1 AND name = 'Default Store';
    IF n <> 1 THEN bad := bad || ' stores(id=1) is not Default Store'; END IF;
    SELECT count(*) INTO n FROM customers WHERE id = 1 AND is_walk_in;
    IF n <> 1 THEN bad := bad || ' customers(id=1, walk-in) missing'; END IF;

    -- Every seed account must sit behind the 428 gate with the published password.
    SELECT count(*) INTO n FROM users
    WHERE username IN ('superadmin','manager','supervisor','cashier',
                       'inventory_staff','finance')
      AND must_change_password;
    IF n <> 6 THEN bad := bad || format(' must_change_password=%s/6', n); END IF;
    SELECT count(*) INTO n FROM users
    WHERE username IN ('superadmin','manager','supervisor','cashier',
                       'inventory_staff','finance')
      AND password_hash = crypt('admin123', password_hash);
    IF n <> 6 THEN bad := bad || format(' admin123_verified=%s/6', n); END IF;

    -- Everything outside the nine seeded tables must be empty. Derived from
    -- pg_class so a future 054_*.sql migration is covered without editing here.
    FOR tbl IN
        SELECT c.relname FROM pg_class c
        JOIN pg_namespace nsp ON nsp.oid = c.relnamespace
        WHERE nsp.nspname = 'public' AND c.relkind = 'r'
          AND c.relname <> 'schema_migrations'
          AND c.relname <> ALL (ARRAY['roles','permissions','role_permissions',
                                       'payment_methods','customer_groups','stores',
                                       'customers','app_settings','users'])
        ORDER BY c.relname
    LOOP
        EXECUTE format('SELECT count(*) FROM public.%I', tbl) INTO n;
        IF n <> 0 THEN bad := bad || format(' %s=%s', tbl, n); END IF;
    END LOOP;

    IF bad <> '' THEN
        RAISE EXCEPTION 'fresh-install verification failed:%', bad;
    END IF;

    RAISE NOTICE 'fresh-install verification passed';
END $$;
SQL

# --------------------------------------------------------------------------
# 6. Uploads — app_settings.logo_path died with the database, so any file left in
#    uploads/logos is orphaned. The directory exists solely because of
#    internal/appsettings/handler.go:55-66 and is gitignored.
# --------------------------------------------------------------------------
if [[ "$KEEP_UPLOADS" -eq 1 ]]; then
  info "Keeping uploads in $UPLOAD_DIR (--keep-uploads)"
elif [[ -d "$UPLOAD_DIR" ]]; then
  info "Clearing orphaned logo uploads"
  find "$UPLOAD_DIR" -mindepth 1 -maxdepth 1 -type f -print -delete
else
  info "No uploads directory to clear"
fi

if [[ "$RESEED" -eq 1 ]]; then
  info "Re-seeding demo data (--reseed) — this is NOT a fresh install"
  "$ROOT/seed-dev.sh"
fi

# --------------------------------------------------------------------------
# 7. Runbook
# --------------------------------------------------------------------------
cat <<EOF

============================================================
 Database reset to first-install state. Verification passed.
============================================================

 1. Restart the backend. Caches survive a database swap:
      - run-dev.sh : press 'r' + Enter
      - Ristretto, 10 min TTL   (internal/wiring/wiring.go:419-448)
      - branding cache, 60 s   (internal/appsettings/handler.go:105-133)

 2. Clear browser state for http://localhost:5173. The reset cannot reach it:
      - the refresh_token cookie now points at a deleted refresh_tokens row
      - sessionStorage.access_token was minted against the old user set
    DevTools -> Application -> Storage -> Clear site data, or in the console:
      localStorage.clear(); sessionStorage.clear();
    This also resets pos.theme -> dark, pos.locale -> en, and pos.printConfig.

 3. Delete the e2e token cache if you plan to run Playwright:
      rm -f $E2E_TOKEN_CACHE

 4. Verify the first-install experience:
      - log in as superadmin / admin123
      - the field takes the USERNAME 'superadmin', not the email address
      - expect the HTTP 428 forced-rotation dialog
        (internal/middleware/auth.go:23-55)
      - rotate the password first: the 428 gate covers every protected route
        except /api/change-password, /api/logout and /api/validate, so
        /api/stores/1/readiness also answers 428 until the rotation lands
      - only then does GET /api/stores/1/readiness report ready: false with
        exactly two blockers: storage_location and catalog
        (internal/store/service.go:174-225)
EOF
