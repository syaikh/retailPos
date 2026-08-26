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
               window.print() (dialog)                     (tools/print-agent)
                                                                 │ routes by
                                                                 │ PRINT_TARGET
                                                          file │ pdf │ thermal
```

### Frontend

- **`shared/stores/printConfig.svelte.ts`** — `mode` (`preview` | `silent`) and
  `agentUrl`, persisted to `localStorage`, seeded from `VITE_PRINT_MODE` /
  `VITE_PRINT_AGENT_URL`. Per-browser config.
- **`shared/services/print-service.ts`** — single `printReceipt(data)` entry
  point. In `silent` mode it `POST`s the payload to the agent; in `preview`
  mode it renders the 58mm overlay and calls `window.print()`. If the agent is
  unreachable in `silent` mode, it **falls back to preview** so a receipt is
  always produced.
- **`app/components/PrintModeToggle.svelte`** — cart UI (Preview/Silent
  segmented control + gear editor for the agent URL with a `/health` test).
- Wiring: POS auto-print, POS manual print, and `TransactionDrawer` reprint all
  call `print-service` instead of setting the store + `window.print()` directly.

### Print agent (`tools/print-agent/`)

Zero-dependency Node service (runs on a bare `node` install):

- `GET /health` → `{ ok, target }` (used by the UI Test button; CORS-enabled).
- `POST /print` → body `{ invoice, data, branding }`; routes by `PRINT_TARGET`:
  - `file` — writes a self-contained 58mm HTML receipt to disk (openable in a
    browser). **Primary no-hardware test path.**
  - `pdf` — same HTML into a `pdf/` dir; if `PRINT_PDF_PRINTER` is set, also
    spools via CUPS `lp`.
  - `thermal` — builds an ESC/POS byte stream; writes a `.bin` file by default,
    or writes straight to `PRINT_SERIAL_PORT` (e.g. `/dev/ttyUSB0`) when set.

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
Response: `{ "ok": true, "target": "file", "path": "/tmp/receipt-INV-000123.html" }`.

## Testing without a 58mm printer

1. Start the agent: `cd tools/print-agent && PRINT_TARGET=file node index.js`.
2. In the POS cart, flip **Print** → **Silent** (gear → Test shows "connected").
3. Complete a sale: no dialog appears; `cat /tmp/receipt-<INV>.html` (or open it)
   shows the formatted 58mm receipt. This validates the whole web→agent→output
   pipeline. `thermal` additionally dumps a `escpos-<INV>.bin` for inspection.

Automated coverage:
- Agent: `node --test` (HTML render, ESC/POS bytes, file/thermal write).
- Frontend: `print-service.test.ts` (preview opens dialog; silent POSTs + skips
  dialog; silent falls back on agent failure) and `printConfig.test.ts`.

## Out of scope / future

- **Per-register global config:** `printConfig` is currently per-browser
  `localStorage`. Promoting `print_mode` / `print_agent_url` into the backend
  `app_settings` KV would make it a true per-register setting shared across
  cashiers on the same terminal.
- **WebUSB direct-to-printer:** an alternative to the agent that talks to a USB
  thermal printer from the browser, but with USB-permission friction per print.
- **Silent-mode E2E:** would require a harnessed `tools/print-agent` instance
  and different assertions than the existing `print-receipt.spec.ts` (preview
  only). Intentionally not covered by E2E yet.
