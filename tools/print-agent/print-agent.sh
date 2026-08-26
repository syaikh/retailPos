#!/bin/bash
# Print Agent launcher with CLI flags.
# Usage: ./print-agent.sh [flags]
#   -t, --transport <file|tcp|serial>   PRINT_TRANSPORT  (default: file)
#   -p, --port <port>                   PORT             (default: 9123)
#   -o, --output-dir <dir>              PRINT_OUTPUT_DIR (default: OS temp dir)
#       --tcp-addr <host:port>          PRINT_TCP_ADDR   (tcp transport)
#       --serial-device <path>          PRINT_SERIAL_DEVICE (serial transport)
#       --token <token>                 PRINT_TOKEN      (optional bearer auth)
#       --allowed-origins <csv>         ALLOWED_ORIGINS  (default: reflect origin)
#   -b, --build                         force (re)build the binary
#   -h, --help                          show this usage
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

BUILD=0
while [ $# -gt 0 ]; do
  case "$1" in
    -t|--transport)     PRINT_TRANSPORT="$2"; shift 2;;
    -p|--port)          PORT="$2"; shift 2;;
    -o|--output-dir)    PRINT_OUTPUT_DIR="$2"; shift 2;;
    --tcp-addr)         PRINT_TCP_ADDR="$2"; shift 2;;
    --serial-device)    PRINT_SERIAL_DEVICE="$2"; shift 2;;
    --token)            PRINT_TOKEN="$2"; shift 2;;
    --allowed-origins)  ALLOWED_ORIGINS="$2"; shift 2;;
    -b|--build)         BUILD=1; shift;;
    -h|--help)          sed -n '2,13p' "$SCRIPT_DIR/print-agent.sh"; exit 0;;
    *)                  echo "Unknown flag: $1" >&2; exit 1;;
  esac
done

BIN="$SCRIPT_DIR/print-agent-bin"
if [ "$BUILD" -eq 1 ] || [ ! -x "$BIN" ]; then
  echo "Building print-agent..."
  go build -o "$BIN" ./cmd/print-agent
fi

# Export only the variables that were explicitly set (others fall back to defaults).
[ -n "${PRINT_TRANSPORT+x}" ]    && export PRINT_TRANSPORT
[ -n "${PORT+x}" ]               && export PORT
[ -n "${PRINT_OUTPUT_DIR+x}" ]   && export PRINT_OUTPUT_DIR
[ -n "${PRINT_TCP_ADDR+x}" ]     && export PRINT_TCP_ADDR
[ -n "${PRINT_SERIAL_DEVICE+x}" ] && export PRINT_SERIAL_DEVICE
[ -n "${PRINT_TOKEN+x}" ]         && export PRINT_TOKEN
[ -n "${ALLOWED_ORIGINS+x}" ]    && export ALLOWED_ORIGINS

echo "Starting print-agent (transport=${PRINT_TRANSPORT:-file}, port=${PORT:-9123})..."
exec "$BIN"
