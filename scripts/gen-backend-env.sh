#!/usr/bin/env bash
# Generate the Podman deploy secret file (/etc/retail-pos/backend.env).
#
# WHY: deploy/podman-deploy.sh validates DB_PASSWORD and JWT_SECRET before it
# will start the backend (validate_backend_config). Missing values make the
# backend panic on startup with "JWT_SECRET environment variable is required"
# (internal/config/config.go:195-198). This script creates the file with strong
# random values so a first deploy does not fail that check.
#
# deploy/podman-deploy.sh reads the file two ways:
#   * sourced (set -a; . "$ENV_FILE") so the script itself can use the values;
#   * passed to the backend container with --env-file, keeping the secrets out
#     of the process table (see the comments at podman-deploy.sh:25-53).
#
# The file holds:
#   DB_PASSWORD         PostgreSQL (and POSTGRES_PASSWORD) password.
#   JWT_SECRET          256-bit secret for access tokens.
#   JWT_SECRET_REFRESH  separate 256-bit secret for refresh tokens.
#
# Refuses to overwrite an existing file unless --force: rotating JWT_SECRET
# invalidates every issued token, and changing DB_PASSWORD after the database
# has been initialised locks the backend out of it.
#
# USAGE
#   scripts/gen-backend-env.sh                 # writes /etc/retail-pos/backend.env
#   scripts/gen-backend-env.sh --path ./backend.env
#   scripts/gen-backend-env.sh --force         # rotate (replaces existing)
#
# EXIT CODES
#   0  file written
#   1  refused (exists without --force), bad option, or openssl missing

set -euo pipefail

# Honor ENV_FILE the same way podman-deploy.sh does, so both agree on the path.
ENV_FILE="${ENV_FILE:-/etc/retail-pos/backend.env}"
FORCE=0

die() { printf 'gen-backend-env: %s\n' "$1" >&2; exit 1; }
info() { printf 'gen-backend-env: %s\n' "$1"; }

usage() {
  # Print the leading comment block — every line between the shebang and the
  # first code line — stripping the "# " prefix, so the help text cannot drift
  # from the documentation above.
  awk 'NR == 1 { next } /^#/ { sub(/^# ?/, ""); print; next } { exit }' "${BASH_SOURCE[0]}"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --path)
      [[ $# -ge 2 ]] || die "--path needs a value"
      ENV_FILE="$2"
      shift 2
      ;;
    --force|-f) FORCE=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) usage >&2; die "unknown option: $1" ;;
  esac
done

command -v openssl >/dev/null 2>&1 || die "openssl not found; install it or write the values by hand"

if [[ -e "$ENV_FILE" && "$FORCE" -ne 1 ]]; then
  die "$ENV_FILE already exists (use --force to rotate; this invalidates all existing tokens)"
fi

# Root-requiring operations run via sudo when the caller is not already root,
# matching the manual `sudo mkdir`/`sudo tee` steps documented in podman-deploy.sh.
as_root() {
  if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    "$@"
  else
    sudo "$@"
  fi
}

# The deploy is run by the invoking (non-root) user under rootless podman, and
# podman-deploy.sh *sources* this file as that user. It must therefore be
# readable by them: the directory traversable, the file owned by them. Running
# under rootful podman still works — root reads any owner.
owner="${SUDO_USER:-$(id -un)}"
dir="$(dirname "$ENV_FILE")"

umask 077
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

{
  printf 'DB_PASSWORD=%s\n' "$(openssl rand -hex 24)"
  printf 'JWT_SECRET=%s\n' "$(openssl rand -hex 32)"
  printf 'JWT_SECRET_REFRESH=%s\n' "$(openssl rand -hex 32)"
} > "$tmp"

as_root mkdir -p "$dir"
# Traverse-only (711), not listable, so the secret is not visible in directory
# listings while the deploy user can still reach it. Skip "." and the caller's
# cwd so a custom --path never changes the permissions of a working directory.
if [[ "$dir" != "." && "$dir" != "$PWD" ]]; then
  as_root chmod 711 "$dir"
fi
as_root install -m 600 -o "$owner" "$tmp" "$ENV_FILE"

info "wrote $ENV_FILE (mode 600, owner $owner)"
info "keys: DB_PASSWORD, JWT_SECRET, JWT_SECRET_REFRESH"
info "next: DB_SSLMODE=disable ./deploy/podman-deploy.sh start"
