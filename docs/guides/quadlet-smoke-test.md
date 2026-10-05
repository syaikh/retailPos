# Quadlet smoke test

A manual procedure to confirm the `deploy/quadlet/` units actually start and stay
up on a real machine, which is the one thing static verification cannot answer.
`systemd-analyze verify` and `podman quadlet print` both parse these files
cleanly, and neither of them has ever executed a container.

**You probably do not need this.** The supported deployment path is
`./deploy/podman-deploy.sh` (or `make deploy`). Quadlet exists only for installs
where systemd should start the stack at boot. If you are not using it, use the
script and skip this document.

## Before you start

Run this on a **test machine, not the live till**. It starts real containers and
binds ports 8000, 8080 and 5432. Check nothing else is holding them:

```bash
ss -ltnp | grep -E ':(8000|8080|5432)\b' || echo "all three ports are free"
```

You need the secret file to exist and to be readable by your own account. A
root-owned `0600` file cannot be read by a rootless unit, which is the most
common reason this test fails at step 5:

```bash
sudo chown "$USER" /etc/retail-pos/backend.env
sudo chmod 600 /etc/retail-pos/backend.env
```

## 1. Build the images

Quadlet starts containers; it never builds them. Nothing else in this path would
build them either, which is why the script has a `build` subcommand of its own:

```bash
./deploy/podman-deploy.sh build
```

## 2. Install the units

```bash
mkdir -p ~/.config/containers/systemd
cp deploy/quadlet/* ~/.config/containers/systemd/
systemctl --user daemon-reload
```

`loginctl enable-linger` keeps the units running after you log out. Without it
they stop when your last session ends, which looks like the test failing:

```bash
loginctl enable-linger "$USER"
```

## 3. Start, and watch

```bash
systemctl --user enable --now retail-pos-postgres.service \
  retail-pos-backend.service retail-pos-frontend.service
journalctl --user -u retail-pos-backend.service -f
```

## 4. Check it stayed up

The number that matters is `NRestarts`. Zero means it came up cleanly. Anything
climbing means a crash loop:

```bash
systemctl --user show -p NRestarts retail-pos-backend.service
curl -fsS http://localhost:8080/health
curl -fsSI http://localhost:8000/
```

`curl` on 8080 only works from this machine now, and that is correct: 8080 and
5432 are bound to `127.0.0.1` and are not reachable from the network. The
frontend on 8000 is the only port intended to be public, because a browser on
the store LAN is its real client.

## 5. Confirm the security posture

Worth doing once, because it is the whole point of the loopback bindings:

```bash
# From this machine: should succeed.
curl -fsS http://127.0.0.1:8080/health > /dev/null && echo "loopback OK"

# From another machine on the LAN: should fail to connect.
# (run that machine, not this one)
curl -fsS --max-time 5 http://<till-ip>:8080/health || echo "correctly refused"
```

## Reading a failure

| What you see | Cause | Fix |
| --- | --- | --- |
| `NRestarts` climbing, log mentions TLS | `DB_SSLMODE=require` against a database with no certificate. See the note below. | Set `DB_SSLMODE=disable` in the secret file, or mount certificates |
| `NRestarts` climbing, `permission denied` on `/etc/retail-pos/backend.env` | Secret file owned by root | The `chown` in *Before you start* |
| `no such image` | Images not built | Step 1 |
| `Unit not found` after `daemon-reload` | Quadlet generator did not run | Confirm `deploy/quadlet/*.pod` is in `~/.config/containers/systemd/` and that `podman quadlet list` shows it |
| Units vanish after logout | No linger | `loginctl enable-linger "$USER"` |

### On `DB_SSLMODE`

`deploy/.env.example` ships `DB_SSLMODE=disable`, which is correct for a
single-host install: with 5432 bound to `127.0.0.1` and compose not publishing
the port at all, no other machine can open a database connection, so there is
nothing on the wire to intercept.

If your `/etc/retail-pos/backend.env` was created before that default changed and
still says `require`, the backend will refuse to start, because no manifest in
this repository mounts a certificate on the database container. Change it to
`disable`, or follow the certificate procedure in
`deploy/PRODUCTION-DEPLOYMENT.md`.

## Undo

Complete and reversible. The database volume is separate from the pod, so
`podman pod rm` does not destroy the data:

```bash
systemctl --user disable --now retail-pos-backend.service \
  retail-pos-frontend.service retail-pos-postgres.service
rm -f ~/.config/containers/systemd/retail-pos*
systemctl --user daemon-reload
systemctl --user reset-failed
podman pod rm -f retail-pos-pod
```

`podman volume ls` will still show the database volume. Removing it is a
separate, destructive step and is not part of this procedure.
