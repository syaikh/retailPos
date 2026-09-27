# Print agent: production install

The print agent runs one instance **per till**, on the till's own machine,
against that till's own printer. It is not part of the container pod, and it
does not talk to the backend. It exists because ESC/POS is a printer dialect,
not an HTTP concern, and a browser cannot open a serial port.

`tools/print-agent` is a separate Go module from the backend. It has its own CI
jobs (format, build, vet, test with the race detector, lint, vulnerability
check), so it is no longer the one part of the repository that nothing tests.

## Why a host service and not a container

The agent supports `file`, `tcp` and `serial` transports. A USB or serial
thermal printer is a device node on the host, and passing it into a container
requires a privileged passthrough that would hand the container every host
device. A till with a network printer could be containerised, but requiring a
host service means one method that works for every printer, including the
cheap USB ones most shops actually have.

## Install

```bash
# 1. Build for the till's own CPU architecture, not your workstation's.
cd tools/print-agent
CGO_ENABLED=0 go build -trimpath -o /tmp/print-agent ./cmd/print-agent
scp /tmp/print-agent till:/tmp/print-agent
# on the till:
install -Dm755 /tmp/print-agent ~/.local/bin/print-agent

# 2. Secret file. Same location and ownership as the backend's.
sudo mkdir -p /etc/retail-pos
sudo tee /etc/retail-pos/print-agent.env >/dev/null <<'EOF'
ENV=production
PORT=9123
PRINT_TRANSPORT=serial
PRINT_SERIAL_DEVICE=/dev/ttyUSB0
PRINT_OUTPUT_DIR=/var/lib/retail-pos/print-agent
PRINT_TOKEN=
ALLOWED_ORIGINS=https://pos.example.com
EOF
sudo chown "$USER" /etc/retail-pos/print-agent.env
sudo chmod 600 /etc/retail-pos/print-agent.env
sudo usermod -aG dialout "$USER"   # serial access; log out and back in
```

`ALLOWED_ORIGINS` must be the exact origin of the POS frontend, the same value
as the backend's `CORS_ORIGIN`. With `ENV=production` the agent refuses to start
without it, and refuses `*` as well. This is the only access control the browser
path has: see the warning below.

For a network printer use `PRINT_TRANSPORT=tcp` with `PRINT_TCP_ADDR`, and drop
the `dialout` group. The `file` transport writes ESC/POS bytes to a directory
instead of a printer; it is for testing and is not a production transport.

## Enable at boot

```bash
mkdir -p ~/.config/systemd/user
cp tools/print-agent/print-agent.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now print-agent.service
loginctl enable-linger "$USER"   # otherwise it stops when you log out
```

The unit is a user unit, so it needs no root and nothing is enabled
system-wide. It runs unprivileged with `ProtectSystem=strict`, and only `/tmp`
is writable. If you set `PRINT_OUTPUT_DIR` to anything outside `/tmp`, add a
`ReadWritePaths=` line for it or the agent will fail to open the file.

## Check it

```bash
systemctl --user status print-agent.service
journalctl --user -u print-agent.service -f
curl -fsS http://127.0.0.1:9123/health
```

The startup line reports the listen address, transport, allowed origins, and
whether a bearer token is enforced. It never prints the token itself.

## Two things to understand before you rely on it

**`PRINT_TOKEN` is not usable from a browser.** The agent enforces it as a
bearer token if it is set, but the frontend does not send an `Authorization`
header, so setting it will make every print return 401. Leave it empty. The
field exists for a future non-browser client; it is not a security control you
can switch on today. Do not assume a configured token protects anything.

**Origin allowlisting is the only real control, and it is a weak one.** The
agent listens on `0.0.0.0:9123` because the browser is a different machine, and
`ALLOWED_ORIGINS` is a check on the `Origin` header. That stops a random website
from printing to your till, which is the realistic threat, but it does not stop
anything that can reach port 9123 on the LAN directly, and it does not stop
someone who can read the frontend's origin. Treat the agent as a device on the
trusted shop LAN:

- Never publish or forward port 9123 off the store LAN.
- Keep it off any shared or guest network.
- If the tills are on a shared network, put the agent behind a proxy that
  authenticates each till, and treat the agent itself as reachable.

The queue is in memory only. A restart loses queued jobs, and a job that has
been sent to the transport cannot be recalled.
