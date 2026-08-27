# Receipt Print Bridge (Silent Printing)

## Problem

The original POS printed receipts through the browser's native `window.print()`,
which opens the OS/browser print dialog (the "print preview + confirm" step).
That dialog is the bottleneck: every sale forces a manual confirmation.

In real high-volume retail (supermarkets, convenience stores) receipts print
**automatically** to a 58mm thermal printer the moment a sale completes — the
cashier never sees or confirms a preview. Browser `window.print()` will always
prompt (browsers forbid silent printing from web code for security), so the
free web-only path can never match real-world speed.

## Goal

Add a *silent* print path that routes the receipt to a local **print agent**,
which talks to a real thermal printer (ESC/POS) or a virtual/PDF printer —
no browser dialog. Keep the existing *preview* path as a fallback and as the
default, so the system is fully usable without any printer hardware.

## Architecture

```
                         ┌─────────────────────────────┐
  Sale completes ───────▶│  print-service.printReceipt  │
                         └──────────────┬──────────────┘
                                        │ reads printConfig.mode
                          ┌─────────────┴──────────────┐
                    preview (default)            silent
                          │                            │
                set printReceipt store +       POST /print  ─────▶  local print agent
                window.print() (dialog)                     (tools/print-agent, Go)
                                                                  │ transport
                                                           file │ tcp │ serial
```

### Frontend

- **`shared/stores/printConfig.svelte.ts`** — `mode` (`preview` | `silent`) and
  `agentUrl`, persisted to `localStorage`, seeded from `VITE_PRINT_MODE` /
  `VITE_PRINT_AGENT_URL`. Per-browser config. When built with
  `VITE_PRINT_MODE=silent`, the mode is **locked**: `localStorage` and the UI
  toggle cannot revert it to `preview`, and the store exposes a `locked` flag.
- **`shared/services/print-service.ts`** — single `printReceipt(data)` entry
  point returning a `PrintResult`. In `silent` mode it `POST`s the payload to the
  agent and, if the agent is unreachable, returns `{ ok: false }` (the caller
  surfaces a Retry/Dismiss toast). It never falls back to the browser dialog.
  In `preview` mode it renders the 58mm overlay and calls `window.print()`.
- **`app/components/PrintModeToggle.svelte`** — cart UI (Preview/Silent
  segmented control + gear editor for the agent URL with a `/health` test).
  When `printConfig.locked` is true the segmented control is replaced by a
  `Silent` badge; the gear editor stays available.
- Wiring: POS auto-print, POS manual print, and `TransactionDrawer` reprint all
  call `print-service` instead of setting the store + `window.print()` directly.

### Print agent (`tools/print-agent/`)

Dependency-free Go service (single binary; `go run ./cmd/print-agent`):

- `GET /health` → `{ ok, printer: { connected, type } }` (used by the UI Test
  button; CORS-enabled).
- `GET /printer` → configured printer + connection status.
- `POST /print` → body `{ invoice, data, branding }`; creates an idempotent job
  (by `job_id`) and returns `202 { job_id, status: "queued" }`. Renders ESC/POS
  and dispatches via the configured transport:
  - `file` (default) — writes the ESC/POS byte stream to a `receipt-<job>.bin`
    file. **Primary no-hardware test path.**
  - `tcp` — sends ESC/POS bytes to a network thermal printer (`PRINT_TCP_ADDR`).
  - `serial` — writes ESC/POS bytes to a USB-serial device
    (`PRINT_SERIAL_DEVICE`, e.g. `/dev/ttyUSB0`).
- `GET /print/jobs/{id}` → job status; `POST /print/jobs/{id}/retry` → retry a
  failed job.

## Agent contract

`POST /print`
```json
{
  "invoice": "INV-000123",
  "data": { "invoice_number": "...", "items": [...], "total_amount": 10000, ... },
  "branding": { "storeName": "...", "storeAddress": "...", "storePhone": "...",
                "receiptHeader": "...", "receiptFooter": "..." }
}
```
Response: `202 { "job_id": "print-...", "status": "queued" }`.

## Testing without a 58mm printer

1. Start the agent: `cd tools/print-agent && go run ./cmd/print-agent` (default
   `file` transport).
2. In the POS cart, flip **Print** → **Silent** (gear → Test shows "connected").
3. Complete a sale: no dialog appears; the agent writes `receipt-<job>.bin` to the
   temp dir — inspect it (or feed it to a printer later) to validate the ESC/POS
   output. `tcp` / `serial` target real printers.

Automated coverage:
- Agent: `go test ./...` (ESC/POS renderer, queue/idempotency, API, transports).
- Frontend: `print-service.test.ts` (preview opens dialog; silent POSTs + skips
  dialog; silent reports failure and does NOT fall back on agent failure) and
  `printConfig.test.ts`.

## Out of scope / future

- **Per-register global config:** `printConfig` is currently per-browser
  `localStorage`. Promoting `print_mode` / `print_agent_url` into the backend
  `app_settings` KV would make it a true per-register setting shared across
  cashiers on the same terminal.
- **WebUSB direct-to-printer:** an alternative to the agent that talks to a USB
  thermal printer from the browser, but with USB-permission friction per print.
- **Silent-mode E2E:** covered by `tests/e2e/silent-print.spec.ts`, which boots
  the real Go print agent (`file` transport) and drives a POS sale in silent mode,
  asserting the job reaches the agent, renders to a `.bin`, and that no browser
  print dialog opens (no preview fallback).
