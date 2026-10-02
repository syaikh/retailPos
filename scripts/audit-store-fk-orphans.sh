#!/usr/bin/env bash
# Store FK orphan audit — read-only preflight for Wave 6
# (docs/design/master-data-store-boundary-audit.md, "Wave 0" open item +
# "Wave 6 — Schema integrity").
#
# WHY: `059_store_fk_integrity.sql` adds `REFERENCES stores(id)` to customers,
# users, goods_receipts and purchase_orders. Those four carry a `store_id`
# integer with no FK, so nothing has ever stopped a row from pointing at a
# store that does not exist. Adding the constraint validates every existing row
# and the migration will hard-fail on the first orphan. This script finds those
# rows *before* that happens, so the failure is a report rather than an outage.
#
# Also checks the Wave 5 / migration 058 prerequisite: `suppliers.store_id` must
# be all-NULL. 058 drops that column and aborts if any row is non-NULL, so this
# turns "the migration will refuse to run" into "here are the exact rows to
# remap first" (see README "Supplier negotiation terms").
#
# READ-ONLY BY CONSTRUCTION. Every statement is a SELECT, and the session is
# pinned with `default_transaction_read_only=on`, so a bug in this script cannot
# write to the database it inspects. Safe to point at production.
#
# USAGE
#   scripts/audit-store-fk-orphans.sh                     # configured DB (.env)
#   scripts/audit-store-fk-orphans.sh --database prod_snapshot
#   scripts/audit-store-fk-orphans.sh --sample 50
#
# Connection follows the same precedence as the Go server's loader: a real
# environment variable wins, and .env only fills gaps.
#
# EXIT CODES
#   0  no orphans and no supplier outliers — Wave 6 is unblocked
#   1  orphans or outliers found (details on stdout)
#   2  misuse, or psql/.env unavailable

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SAMPLE=20

usage() {
	# Print the leading comment block — every line between the shebang and the
	# first code line — stripping the "# " prefix. Deriving the range keeps the
	# help text from silently truncating when the block grows or shrinks.
	awk 'NR == 1 { next } /^#/ { sub(/^# ?/, ""); print; next } { exit }' "${BASH_SOURCE[0]}"
}

die() {
	printf 'audit-store-fk-orphans: %s\n' "$1" >&2
	exit 2
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	--database | -d)
		[[ $# -ge 2 ]] || die "--database needs a value"
		DB_NAME="$2"
		shift 2
		;;
	--sample | -s)
		[[ $# -ge 2 ]] || die "--sample needs a value"
		[[ "$2" =~ ^[0-9]+$ ]] || die "--sample must be a non-negative integer"
		SAMPLE="$2"
		shift 2
		;;
	--help | -h)
		usage
		exit 0
		;;
	*) die "unknown argument: $1 (try --help)" ;;
	esac
done

command -v psql >/dev/null 2>&1 || die "psql not found on PATH"

# .env fills gaps only — an exported variable always wins, matching
# godotenv.Load() in internal/config.
if [[ -f "$ROOT/.env" ]]; then
	while IFS= read -r line; do
		[[ "$line" =~ ^[[:space:]]*# ]] && continue
		[[ "$line" =~ ^[[:space:]]*$ ]] && continue
		key="${line%%=*}"
		val="${line#*=}"
		key="${key//[[:space:]]/}"
		val="${val//\"/}"
		val="${val//\'/}"
		case "$key" in
		DB_HOST | DB_PORT | DB_USER | DB_PASSWORD | DB_NAME)
			[[ -n "${!key:-}" ]] || printf -v "$key" '%s' "$val"
			;;
		esac
	done <"$ROOT/.env"
fi

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-pos}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-retail_pos}"

export PGPASSWORD="$DB_PASSWORD"
# Pins the session read-only: this script cannot write even by accident.
export PGOPTIONS='-c default_transaction_read_only=on'

printf 'Store FK orphan audit\n'
printf '  target : %s@%s:%s/%s\n' "$DB_USER" "$DB_HOST" "$DB_PORT" "$DB_NAME"
printf '  mode   : read-only (default_transaction_read_only=on)\n\n'

# A missing table or column means the schema is not the one this audit was
# written against — that is a finding, not something to paper over.
query() {
	psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
		-v ON_ERROR_STOP=1 -tAX -c "$1" 2>&1
}

printf '%-18s %14s %10s  %s\n' "TABLE" "ROWS(W/ STORE)" "ORPHANS" "VERDICT"
printf '%-18s %14s %10s  %s\n' "------------------" "--------------" "----------" "---------"

failures=0
orphan_failures=0
supplier_failures=0

# check <table> <labelled identifier expression for the sample row>
check() {
	local table="$1" id_cols="$2"
	local total orphans

	total="$(query "SELECT count(*) FROM public.${table} WHERE store_id IS NOT NULL")"
	if [[ "$total" == *"ERROR"* || -z "$total" ]]; then
		printf '%-18s %14s %10s  %s\n' "$table" "-" "-" "SCHEMA MISMATCH"
		printf '  %s\n\n' "$total"
		failures=$((failures + 1))
		return
	fi

	orphans="$(query "SELECT count(*) FROM public.${table} t
		WHERE t.store_id IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM public.stores s WHERE s.id = t.store_id)")"
	if [[ "$orphans" == *"ERROR"* || -z "$orphans" ]]; then
		printf '%-18s %14s %10s  %s\n' "$table" "$total" "-" "QUERY FAILED"
		printf '  %s\n\n' "$orphans"
		failures=$((failures + 1))
		return
	fi

	if [[ "$orphans" -gt 0 ]]; then
		printf '%-18s %14s %10s  %s\n' "$table" "$total" "$orphans" "ORPHANS"
		failures=$((failures + 1))
		orphan_failures=$((orphan_failures + 1))
		if [[ "$SAMPLE" -gt 0 ]]; then
			printf '  sample (up to %d):\n' "$SAMPLE"
			# shellcheck disable=SC2086 # id_cols is a fixed literal list
			query "SELECT '    ' || ${id_cols} || '  store_id=' || t.store_id
				FROM public.${table} t
				WHERE t.store_id IS NOT NULL
				  AND NOT EXISTS (SELECT 1 FROM public.stores s WHERE s.id = t.store_id)
				ORDER BY t.id LIMIT ${SAMPLE}" | grep -v '^$'
		fi
		printf '\n'
	else
		printf '%-18s %14s %10s  %s\n' "$table" "$total" "$orphans" "ok"
	fi
}

check customers "'id=' || t.id || '  name=' || t.name"
check users "'id=' || t.id || '  username=' || t.username"
check goods_receipts "'id=' || t.id || '  gr_number=' || t.gr_number"
check purchase_orders "'id=' || t.id || '  po_number=' || t.po_number"

# 058 prerequisite: before 058 is applied, suppliers.store_id must be all-NULL
# (the migration drops the column and aborts otherwise). After 058 the column is
# *gone*, which is the desired end state — so absence is a pass, not a mismatch.
# Migration state is therefore read from information_schema rather than assumed.
if [[ -f "$ROOT/database/migrations/058_supplier_terms_store_scope.sql" ]]; then
	sup_total="$(query "SELECT count(*) FROM public.suppliers")"
	if [[ "$sup_total" == *"ERROR"* || -z "$sup_total" ]]; then
		printf '%-18s %14s %10s  %s\n' "suppliers" "-" "-" "SCHEMA MISMATCH"
		printf '  %s\n' "$sup_total"
		failures=$((failures + 1))
	else
		sup_col="$(query "SELECT count(*) FROM information_schema.columns
			WHERE table_schema='public' AND table_name='suppliers' AND column_name='store_id'")"
		if [[ "$sup_col" == "0" ]]; then
			# 058 already applied here: the column it drops no longer exists.
			printf '%-18s %14s %10s  %s\n' "suppliers" "$sup_total" "-" "ok (058 applied)"
		else
			sup_bad="$(query "SELECT count(*) FROM public.suppliers WHERE store_id IS NOT NULL")"
			if [[ "$sup_bad" == *"ERROR"* || -z "$sup_bad" ]]; then
				printf '%-18s %14s %10s  %s\n' "suppliers" "$sup_total" "-" "QUERY FAILED"
				printf '  %s\n' "$sup_bad"
				failures=$((failures + 1))
			elif [[ "$sup_bad" -gt 0 ]]; then
				printf '%-18s %14s %10s  %s\n' "suppliers" "$sup_total" "$sup_bad" "058 BLOCKED"
				failures=$((failures + 1))
				supplier_failures=$((supplier_failures + 1))
				printf '  suppliers.store_id must be all-NULL; 058 drops the column and aborts otherwise.\n'
				if [[ "$SAMPLE" -gt 0 ]]; then
					printf '  rows to remap (up to %d):\n' "$SAMPLE"
					query "SELECT '    id=' || id || '  name=' || name || '  store_id=' || store_id
						FROM public.suppliers WHERE store_id IS NOT NULL
						ORDER BY id LIMIT ${SAMPLE}" | grep -v '^$'
				fi
				printf '\n'
			else
				printf '%-18s %14s %10s  %s\n' "suppliers" "$sup_total" "$sup_bad" "ok (058 ready)"
			fi
		fi
	fi
fi

if ((failures > 0)); then
	printf 'FAILED: %d check(s) need attention.\n' "$failures" >&2
	if ((orphan_failures > 0)); then
		printf 'Wave 6 (059_store_fk_integrity.sql) must not be applied until the %d orphaned row(s)\n' "$orphan_failures" >&2
		printf 'above are resolved — that is data migration, deliberately out of scope for the\n' >&2
		printf 'constraint work. Re-run this script to confirm before scheduling the migration.\n' >&2
	fi
	if ((supplier_failures > 0)); then
		printf 'Migration 058 (supplier_terms_store_scope) must not be applied until the suppliers\n' >&2
		printf 'above are remapped into store-scoped product_suppliers rows.\n' >&2
	fi
	exit 1
fi

printf 'Clean: no orphaned store_id values, no suppliers.store_id outliers.\n'
