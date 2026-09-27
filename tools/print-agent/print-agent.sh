#!/bin/bash
# Print Agent launcher with CLI flags.
# Usage: ./print-agent.sh [--env-file <file>] [flags]
#   -e, --env-file <file>             load KEY=VALUE pairs, then apply flags
#   -t, --transport <file|tcp|serial>   PRINT_TRANSPORT  (default: file)
#   -p, --port <port>                   PORT             (default: 9123)
#   -a, --bind-address <addr>           PRINT_BIND_ADDRESS (default: 0.0.0.0)
#   -o, --output-dir <dir>              PRINT_OUTPUT_DIR (default: OS temp dir)
#       --tcp-addr <host:port>          PRINT_TCP_ADDR   (tcp transport)
#       --serial-device <path>          PRINT_SERIAL_DEVICE (serial transport)
#       --allowed-origins <csv>         ALLOWED_ORIGINS  (default: reflect origin)
#   -b, --build                         force (re)build the binary
#   -h, --help                          show this usage
#
# There is no --token flag and no PRINT_TOKEN setting. Bearer auth was supported
# and removed: the only client is a browser, so a token would have to be built
# into the page where anyone can read it, which makes it useless as a secret. It
# had no other client, and enabling it made every print return 401. Access is
# controlled by ALLOWED_ORIGINS plus keeping the agent on a trusted network.
#
# Configuration lives in a secret file, one per till:
#
#   printf 'ENV=production\nALLOWED_ORIGINS=https://pos.example.com\nPRINT_SERIAL_DEVICE=/dev/ttyUSB0\n' \
#     | sudo tee /etc/retail-pos/print-agent.env >/dev/null
#   sudo chown "$USER" /etc/retail-pos/print-agent.env
#   sudo chmod 600 /etc/retail-pos/print-agent.env
#   ./print-agent.sh --env-file /etc/retail-pos/print-agent.env
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

BUILD=0
ENV_FILE=""

# The env file is sourced before the other flags are parsed, so an explicit flag
# wins over the file. Reading it in the same pass would invert that: sourcing
# afterwards would silently overwrite whatever the operator just typed, which is
# the wrong way round for a value they are correcting on the command line.
ARGS=()
while [ $# -gt 0 ]; do
  case "$1" in
    -e|--env-file) ENV_FILE="$2"; shift 2;;
    *)             ARGS+=("$1"); shift;;
  esac
done
set -- ${ARGS[@]+"${ARGS[@]}"}

if [ -n "$ENV_FILE" ]; then
  if [ ! -r "$ENV_FILE" ]; then
    echo "error: cannot read env file: $ENV_FILE" >&2
    exit 1
  fi
  set -a
  # shellcheck disable=SC1090
  . "$ENV_FILE"
  set +a
fi

while [ $# -gt 0 ]; do
  case "$1" in
    -t|--transport)     PRINT_TRANSPORT="$2"; shift 2;;
    -p|--port)          PORT="$2"; shift 2;;
    -a|--bind-address)  PRINT_BIND_ADDRESS="$2"; shift 2;;
    -o|--output-dir)    PRINT_OUTPUT_DIR="$2"; shift 2;;
    --tcp-addr)         PRINT_TCP_ADDR="$2"; shift 2;;
    --serial-device)    PRINT_SERIAL_DEVICE="$2"; shift 2;;
    --token)
      echo "error: --token is removed. Bearer auth was removed because the only client" >&2
      echo "       is a browser, and a token shipped to a browser is not a secret." >&2
      echo "       Delete PRINT_TOKEN from the env file too; see --help." >&2
      exit 1
      ;;
    --allowed-origins)  ALLOWED_ORIGINS="$2"; shift 2;;
    -b|--build)         BUILD=1; shift;;
    -h|--help)          sed -n '2,27p' "$SCRIPT_DIR/print-agent.sh"; exit 0;;
    *)                  echo "Unknown flag: $1" >&2; exit 1;;
  esac
done

BIN="$SCRIPT_DIR/print-agent-bin"
if [ "$BUILD" -eq 1 ] || [ ! -x "$BIN" ]; then
  echo "Building print-agent..."
  go build -o "$BIN" ./cmd/print-agent
fi

# Export only the variables that were explicitly set (others fall back to defaults).
for name in ENV PRINT_TRANSPORT PORT PRINT_BIND_ADDRESS PRINT_OUTPUT_DIR \
            PRINT_TCP_ADDR PRINT_SERIAL_DEVICE ALLOWED_ORIGINS; do
  if [ -n "${!name+x}" ]; then
    export "${name?}"
  fi
done

echo "Starting print-agent (transport=${PRINT_TRANSPORT:-file}, port=${PORT:-9123})..."
exec "$BIN"
