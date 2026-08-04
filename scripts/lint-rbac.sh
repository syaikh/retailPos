#!/usr/bin/env bash
# RBAC Sprint 0 — annotation-based lint (docs/audits/rbac-sprint0-audit.md §14).
#
# Memastikan tidak ada authorization berbasis role di frontend:
#   - Semua keputusan akses via permission (rbac.can/canAny/canAll + Permissions registry).
#   - Role usage (rbac.userRole / role checks / role_id) hanya boleh untuk presentasi,
#     ownership, atau compatibility layer — WAJIB diberi annotation pada baris sebelumnya:
#       // @display-only, // @ownership-only, atau // @compatibility-layer
#   - String literal permission dalam rbac.can(...) DILARANG (wajib konstanta registry).
#
# File yang di-exempt (implementasi RBAC / sudah dikategorikan di audit doc):
#   - shared/composables/useRBAC.svelte.ts  (inti komposable, menangani objek role)
#   - modules/admin/components/UserTable.svelte (role_id mapping sudah dikategorikan)
# File test (__tests__, *.test.ts, *.spec.ts) tidak di-scan.

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEB_SRC="$ROOT/web/src"

EXEMPT_FILES=(
  "shared/composables/useRBAC.svelte.ts"
  "modules/admin/components/UserTable.svelte"
)

ANNOTATIONS='@(display-only|ownership-only|compatibility-layer)'

# Role-based authz / legacy getters / string-literal permission calls.
PATTERN="\brole[[:space:]]*(===|!==)|\buserRole[[:space:]]*(===|!==)|switch[[:space:]]*\([[:space:]]*role|role\?\.name|roles\.includes\(|role_id[[:space:]]*(===|!==)[[:space:]]*[0-9]|getUserRoleName|allowedStockRoles|allowedInventoryRoles|ADMIN_ROLES|MANAGER_ROLES|rbac\.isCashier|rbac\.(can|canAny|canAll)\([^)]*['\"]"

errors=0
scanned=0

while IFS= read -r -d '' file; do
  rel="${file#"$WEB_SRC"/}"

  case "$rel" in
    *.test.ts|*.spec.ts|*/__tests__/*) continue ;;
  esac
  for ex in "${EXEMPT_FILES[@]}"; do
    [[ "$rel" == "$ex" ]] && continue 2
  done

  matches="$(grep -nE "$PATTERN" "$file" 2>/dev/null || true)"
  [[ -z "$matches" ]] && continue
  scanned=$((scanned + 1))

  while IFS=: read -r ln rest; do
    [[ -z "$ln" ]] && continue

    # Annotation pada baris yang sama → boleh.
    if grep -qE "$ANNOTATIONS" <<<"$rest"; then
      continue
    fi

    # Annotation pada blok non-blank sebelum baris match (window maks 10 baris).
    start="$(awk -v n="$ln" 'NR < n && /^[[:space:]]*$/ { last=NR } END { s=last+1; m=n-10; if (s<m) s=m; print s }' "$file")"
    if sed -n "${start},$((ln - 1))p" "$file" 2>/dev/null | grep -qE "$ANNOTATIONS"; then
      continue
    fi

    printf 'RBAC-LINT: %s:%s: %s\n' "$rel" "$ln" "$rest"
    errors=$((errors + 1))
  done <<<"$matches"
done < <(find "$WEB_SRC" -type f \( -name '*.svelte' -o -name '*.ts' \) -print0)

if ((errors > 0)); then
  printf 'FAILED: %d lint violation(s).\n' "$errors" >&2
  printf 'Annotate legitimate role usage with @display-only / @ownership-only / @compatibility-layer on the line before.\n' >&2
  exit 1
fi

printf 'RBAC lint clean (%d file(s) scanned, 0 violations).\n' "$scanned"
