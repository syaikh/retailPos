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

# 2. Secret file. Same location and ownership as the backend's. This exact path
#    is what print-agent.service reads, so the two must not drift apart.
sudo mkdir -p /etc/retail-pos
sudo tee /etc/retail-pos/print-agent.env >/dev/null <<'EOF'
ENV=production
PORT=9123
PRINT_TRANSPORT=serial
PRINT_SERIAL_DEVICE=/dev/ttyUSB0
ALLOWED_ORIGINS=https://pos.example.com
# Cash drawer. Both lines are optional and shown for reference: unset gives
# pin 0 (drawer connector pin 2) and a 25 ms pulse, which is what every drawer
# sold locally expects. Change them only if the drawer does not fire or fires
# with the wrong polarity.
#PRINT_DRAWER_PIN=0
#PRINT_DRAWER_PULSE_MS=25
EOF
sudo chown "$USER" /etc/retail-pos/print-agent.env
sudo chmod 600 /etc/retail-pos/print-agent.env
sudo usermod -aG dialout "$USER"   # serial access; log out and back in
```

`PRINT_OUTPUT_DIR` is absent above on purpose. The service runs with
`ProtectSystem=strict`, so only `/tmp` is writable. A serial or tcp printer never
touches the output directory, so setting one would look correct in the env file
and then fail to open at runtime.

`ALLOWED_ORIGINS` must be the exact origin of the POS frontend, the same value
as the backend's `CORS_ORIGIN`. With `ENV=production` the agent refuses to start
without it, and refuses `*` as well. This is the only access control the browser
path has: see the warning below.

For a network printer use `PRINT_TRANSPORT=tcp` with `PRINT_TCP_ADDR`, and drop
the `dialout` group. The `file` transport writes ESC/POS bytes to a directory
instead of a printer; it is for testing and is not a production transport.

**Choose `tcp` whenever the fleet is not all Linux.** Everything above is
systemd, and `serial` is configured by a device node plus group membership —
the agent exposes no baud, parity or flow-control setting at all, so the line
rate is whatever the driver defaults to. That path is exercised on Linux and
nowhere else. A printer with an Ethernet port gives every till the same
`PRINT_TCP_ADDR=192.168.x.x:9100`, needs no group membership, no device
permissions and no per-OS cable convention, and lets a Windows till run the
same binary under a service wrapper with no OS-specific configuration. It is
the default recommendation for a mixed shop; [hardware
recommendations](hardware-recommendations.md) lists locally sold printers with
an Ethernet port.

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

The startup line reports the listen address, the transport, and the exact
origins it will accept:

```
[print-agent] listening on 0.0.0.0:9123 transport=serial origins=[http://192.168.1.10:8000] auth=origin-allowlist
```

Read that line on a new till. If the origin is not your website's address, the
agent is misconfigured. The agent holds no credential, so there is no secret in
that output and nothing to redact from a bug report.

## Several tills, one print agent each

This is the normal shape of a shop: one agent per printer PC, and any number of
PCs. Install the agent on each printer PC exactly as above, and set the **same**
`ALLOWED_ORIGINS` on every one of them, because it checks that the request came
from your website. Do not give each agent a different origin allowlist.

You do **not** need a separate website build per PC. The frontend works out
which agent to use in this order:

1. The address saved on that register's print settings screen (stored in that
   one browser)
2. `VITE_PRINT_AGENT_URL`, baked into the build
3. `http://localhost:9123` — the default

Leave `VITE_PRINT_AGENT_URL` **unset** and every register finds the agent on its
own PC, so one build serves the whole shop. Setting it names a single address for
every register at once, which in a multi-PC shop means most tills print to the
wrong machine or to nothing.

Because the per-register setting outranks the build-time one, a bad
`VITE_PRINT_AGENT_URL` is fixable per PC from the print settings screen rather
than being permanent. It is still a bad default, and the print settings screen
warns when the address in use is not localhost while in silent mode.

`VITE_PRINT_AGENT_URL` is legitimate for two cases: development, where the
frontend dev server runs on a different machine from the agent, and a kiosk setup
where printers are on a server rather than at the till. In a normal shop, leave
it unset.

## Cash drawer

A cash drawer is not a device on the PC. The drawers sold in Indonesia are
**RJ11 and printer-driven**: the drawer's cable goes into the printer's drawer
(kick-out) port, the printer supplies the pulse that throws the latch, and
nothing at all plugs into the till. There is no driver, no port and no second
agent — opening the drawer is a matter of sending the printer four bytes.

That is why it rides the print queue rather than getting a route of its own:

```http
POST /print
{ "type": "drawer" }
```

The agent turns that into an `ESC p <pin> <pulse>` pulse and hands it to
whichever transport the printer is on. Reusing `POST /print` means the drawer
inherits the `ALLOWED_ORIGINS` check and `job_id` idempotency the receipt path
already has; drawer job ids carry a `drawer-` prefix so a pulse can never
collide with a `print-` receipt job.

### When the drawer opens

- **After a sale, automatically** — but only when the sale was paid in cash,
  including a split with a cash part. A QRIS or card sale has nothing to hand
  back and does not open it.
- **Manually, from the print settings panel** — in the cart footer, click ⚙
  next to the Preview/Silent toggle and press **Open Drawer**. It answers
  "Signal sent — the drawer should have opened" or reports that the agent could
  not be reached. Use it on a new till, and after moving or replacing the
  printer.

Neither path shows a receipt preview, opens the browser print dialog, or
consults the print mode: preview vs silent is a choice about how a *receipt* is
presented, and the drawer cannot be presented on screen instead. When no agent
answers, the drawer stays quiet on purpose — the receipt that follows already
raises the agent-unreachable notification, and a second toast on every cash
sale is noise. The manual button, which has no receipt behind it, does report
the outcome.

### Wiring

1. Use the port the printer labels **`DK` / `DR` / "drawer"** — the case, not
   the shape of the plug. On printers that also carry an RS-232 socket that
   port is RJ11 too and sits right next to the drawer one; most 58 mm units
   have only the drawer port, so there is nothing to confuse it with.
2. The RJ11 lead ships with the drawer and only fits one way. Do not force it.
3. Power the printer **off** before plugging or unplugging a drawer; the port
   is live.
4. The drawer has no cable to the PC and no adapter of its own. If a listing
   offers a "USB cash drawer", it is not the kind of drawer this drives.

**Check the printer and the drawer as a pair, before buying.** The cheapest
58 mm units are sometimes cut down to USB alone with no drawer port, and a
drawer whose microswitch or cable does not match will not fire. Confirm the
SKU's spec sheet says *drawer kick-out* and pick from
[hardware recommendations](hardware-recommendations.md), where that check is
already made for models sold locally.

### Configuration

| Env | Default | Meaning |
| --- | --- | --- |
| `PRINT_DRAWER_PIN` | `0` | `ESC p` pin select: `0` or `2` drives drawer connector **pin 2** (the normal polarity), `1` or `5` drives **pin 5** (reversed). Every other value is refused at startup. |
| `PRINT_DRAWER_PULSE_MS` | `25` | Pulse length in milliseconds. Leave unset unless the drawer is too weak to latch or so violent it strains; an unset or zero value uses the encoder's 25 ms default. |

Both are validated in **every** environment, dev included, not only in
production: a pin that drives nothing is a configuration error, and it is far
better heard at startup than as a drawer that silently never opens on the first
cash sale of the day.

Pin choice follows wiring convention, not printer brand: connector pin 2 is
the usual one and is what `0` and `2` both select, so change the pin only if
the drawer's own manual specifies otherwise.

## What protects the agent, and what does not

**There is no token.** `PRINT_TOKEN` was supported and has been removed. The
only client is a browser, so a token would have to be built into the page where
anyone can read it through developer tools, which makes it useless as a secret.
It had no other client, and because the browser never sent it, setting it made
**every print return 401** — a silent, total outage from a setting that looked
like it improved security. If an old env file still has a `PRINT_TOKEN` line, the
agent ignores it and logs a warning rather than refusing to start, so leftover
config cannot take a till offline. Delete the line.

**Origin allowlisting is the control, and it is a bounded one.** The
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
