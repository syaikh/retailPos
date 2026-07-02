#!/usr/bin/env bash
# =============================================================================
# Script : seed-daily-dev.sh
# Lokasi : project root
# Fungsi : Seed transaksi harian untuk lingkungan development
# Usage   : ./seed-daily-dev.sh [YYYY-MM-DD] [--min N] [--max N] [--cashier-id ID]
#                                  [--store-id ID] [--no-audit] [--help]
# =============================================================================

set -euo pipefail

# ── Defaults ──────────────────────────────────────────────────────────────
TARGET_DATE=""
MIN_COUNT=10
MAX_COUNT=50
CASHIER_ID=0
STORE_ID=0
NO_AUDIT=false

# ── Usage ─────────────────────────────────────────────────────────────────
show_help() {
  cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Daily transaction seeder — generates 10–50 completed sales for a single day.

Opsi:
  YYYY-MM-DD              Target date (default: hari ini)
  --min  N                Jumlah transaksi minimum (default: 10)
  --max  N                Jumlah transaksi maksimum (default: 50)
  --cashier-id ID         Paksa ID kasir tertentu (0 = acak, default: 0)
  --store-id  ID          Paksa ID toko tertentu (0 = acak, default: 0)
  --no-audit              Jangan tulis ke audit_logs (default: tulis)
  -h, --help              Tampilkan bantuan ini

Contoh:
  ./seed-daily-dev.sh
  ./seed-daily-dev.sh 2025-01-15 --min 20 --max 40
  ./seed-daily-dev.sh --store-id 1 --cashier-id 3
EOF
  exit 0
}

# ── Parse args ──────────────────────────────────────────────────────────────
if [[ ${1:-} == "--help" || ${1:-} == "-h" ]]; then
  show_help
fi

ARGS=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    20[0-9][0-9]-[0-1][0-9]-[0-3][0-9])
      TARGET_DATE="$1"
      shift
      ;;
    --min)
      MIN_COUNT="$2"
      shift 2
      ;;
    --max)
      MAX_COUNT="$2"
      shift 2
      ;;
    --cashier-id)
      CASHIER_ID="$2"
      shift 2
      ;;
    --store-id)
      STORE_ID="$2"
      shift 2
      ;;
    --no-audit)
      NO_AUDIT=true
      shift
      ;;
    --help|-h)
      show_help
      ;;
    *)
      echo "⚠️  Argumen tidak dikenali: $1" >&2
      show_help
      ;;
  esac
done

# ── Build flags ────────────────────────────────────────────────────────────
FLAGS="-daily.min=$MIN_COUNT -daily.max=$MAX_COUNT"
[[ $CASHIER_ID -gt 0 ]] && FLAGS="$FLAGS -daily.cashier-id=$CASHIER_ID"
[[ $STORE_ID   -gt 0 ]] && FLAGS="$FLAGS -daily.store-id=$STORE_ID"
[[ -n $TARGET_DATE ]]  && FLAGS="$FLAGS -daily.date=$TARGET_DATE"
[[ $NO_AUDIT == true ]] && FLAGS="$FLAGS -daily.no-audit"

# ── Info banner ─────────────────────────────────────────────────────────────
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  🗓  Daily Transaction Seeder  |  INV-YYYY-NNNNNN"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Flags : $FLAGS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# ── Run ─────────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "$SCRIPT_DIR"
go run ./cmd/dummy $FLAGS
