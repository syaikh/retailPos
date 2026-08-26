# Print Agent (Go)

Local print agent for the POS. Receives receipt payloads from the browser and
dispatches them to a printer as **ESC/POS** bytes. Dependency-free, single binary.

This replaces the previous Node agent (`tools/print-agent` historically shipped a
Node version). The frontend (`web/src/shared/services/print-service.ts`) talks to
it over `POST /print`.

## Build & run

```bash
cd tools/print-agent
go build -o print-agent ./cmd/print-agent
PORT=9123 PRINT_TRANSPORT=file ./print-agent
```

A flag-driven launcher is also provided (`print-agent.sh`); it builds the binary
on first run (or with `-b`) and translates flags to env vars:

```bash
./print-agent.sh -t file -p 9123 -o /tmp/receipt-out      # file transport
./print-agent.sh -t tcp --tcp-addr 192.168.1.50:9100      # network printer
./print-agent.sh -t serial --serial-device /dev/ttyUSB0   # USB-serial printer
./print-agent.sh -t file -p 9123 --token s3cret --allowed-origins http://localhost:5173
```

Flags: `-t/--transport`, `-p/--port`, `-o/--output-dir`, `--tcp-addr`,
`--serial-device`, `--token`, `--allowed-origins`, `-b/--build`, `-h/--help`.

In `file` mode (default, no hardware needed) receipts are written as ESC/POS
`.bin` files to `PRINT_OUTPUT_DIR` (default OS temp dir), e.g.
`/tmp/receipt-print-<jobid>.bin`. Inspect them to validate the renderer.

## Endpoints

| Method | Path | Purpose |
| ------ | ---- | ------- |
| GET | `/health` | Agent + printer health |
| GET | `/printer` | Configured printer + connection status |
| POST | `/print` | Enqueue a print job (idempotent by `job_id`) |
| GET | `/print/jobs/{id}` | Job status |
| POST | `/print/jobs/{id}/retry` | Retry a failed job |

`POST /print` body (matches `print-service.ts`):

```json
{
  "invoice": "INV-1",
  "data": { "invoice_number": "INV-1", "items": [ ... ], "total_amount": 10000, ... },
  "branding": { "storeName": "My Store", "storeAddress": "...", "storePhone": "...", "receiptHeader": "...", "receiptFooter": "..." }
}
```

The agent returns `202 Accepted` with `{ "job_id", "status": "queued" }`. On
duplicate `job_id` it returns the existing job (no reprint).

## Transports

| `PRINT_TRANSPORT` | Config | Use |
| ----------------- | ------ | --- |
| `file` (default) | `PRINT_OUTPUT_DIR` | Dev/CI — ESC/POS `.bin` output |
| `tcp` | `PRINT_TCP_ADDR=host:port` | Network thermal printer (port 9100) |
| `serial` | `PRINT_SERIAL_DEVICE=/dev/ttyUSB0` | USB-serial thermal printer |

> Real USB (vendor/product discovery via libusb) is a future enhancement; most
> 58mm "USB" printers expose a serial device node and work with `serial`.

## Configuration

| Env | Default | Description |
| --- | --- | --- |
| `PORT` | `9123` | Listen port |
| `PRINT_TRANSPORT` | `file` | `file` \| `tcp` \| `serial` |
| `PRINT_OUTPUT_DIR` | OS temp | File transport output dir |
| `PRINT_TCP_ADDR` | — | `host:port` for tcp |
| `PRINT_SERIAL_DEVICE` | — | serial device path |
| `PRINT_TOKEN` | — | optional bearer token (localhost CORS is the main control) |
| `ALLOWED_ORIGINS` | `*` | comma-separated allowed CORS origins |

## Security

Listens on all interfaces by default for `PORT`; restrict/forward as needed for
your deployment. For localhost use, the browser-origin CORS check is the primary
control; `PRINT_TOKEN` is optional hardening (it would ship in the frontend
bundle, so treat it as obfuscation, not real auth).

## Testing

```bash
cd tools/print-agent
go test ./...
```
