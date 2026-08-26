# Receipt Print Bridge — Silent ESC/POS Printing

## Problem

The original POS printed receipts through the browser's native `window.print()`,
which opens the OS/browser print dialog. Every sale therefore requires manual
confirmation before the receipt is printed.

For a high-volume retail POS, the desired workflow is:

```text
Payment completed
      ↓
Transaction committed
      ↓
Receipt print job queued
      ↓
POS immediately ready for next transaction
      ↓
Receipt prints automatically in background
```

The browser preview remains useful for development, manual preview, and
environments where no print agent or thermal printer is configured, but it
should not be the primary production printing path.

## Goal

Implement a production-oriented silent receipt-printing path using:

- a standalone **Go local print agent** running on each POS terminal;
- **ESC/POS** as the production receipt format;
- an asynchronous **print queue**;
- USB and TCP/IP printer transports;
- idempotent print jobs to prevent accidental duplicate receipts;
- printer health/status reporting;
- the existing browser preview path as an explicit/manual option.

The print agent must be independent from the main Go backend. Its responsibility
is physical printer communication, not sales or payment business logic.

## Architecture

```text
                         ┌────────────────────────┐
                         │       Go Backend       │
                         │                        │
                         │ Sales / Payment        │
                         │ Inventory              │
                         └───────────┬────────────┘
                                     │
                              Sale completed
                                     │
                                     ▼
                         ┌────────────────────────┐
                         │       Svelte POS       │
                         │                        │
                         │ print-service.ts       │
                         └───────────┬────────────┘
                                     │
                              localhost HTTP
                                     │
                                     ▼
                  ┌────────────────────────────────────┐
                  │       Go Print Agent                │
                  │                                    │
                  │  HTTP API                          │
                  │       ↓                            │
                  │  Print Queue                       │
                  │       ↓                            │
                  │  ESC/POS Renderer                  │
                  │       ↓                            │
                  │  Printer Transport                 │
                  │    ├── USB                         │
                  │    └── TCP/IP                      │
                  └────────────────┬───────────────────┘
                                   │
                            ESC/POS bytes
                                   │
                         ┌─────────┴─────────┐
                         ▼                   ▼
                    USB Printer         Network Printer
```

### Transaction and printing flow

Printing is a side effect of a completed transaction. It must not determine
whether the transaction succeeds.

```text
Cashier → Pay
           ↓
     Backend commits sale
           ↓
      Sale successful
           │
           ├───────────────→ POS ready for next sale
           │
           ▼
      Print job created
           ↓
      Local print agent
           ↓
        ESC/POS
           ↓
       Print receipt
```

If the printer is unavailable:

```text
Transaction = COMPLETED

Print job = FAILED / RETRYABLE

POS:
  "Receipt printer unavailable"

  [Retry Print] [Dismiss]
```

The sale must never be rolled back merely because receipt printing failed.

---

# Frontend

## `shared/stores/printConfig.svelte.ts`

Keep the existing print configuration abstraction, but evolve it toward:

```text
printConfig
├── enabled
├── mode
├── agentUrl
├── printerId
└── autoPrint
```

Supported modes:

```text
preview
silent
```

`preview` uses the existing browser receipt preview.

`silent` submits a print job to the local Go print agent.

The current implementation persists configuration in browser `localStorage`.
This is acceptable for the initial implementation, but the long-term target is
per-register configuration stored by the backend so all cashiers using the same
terminal share the same printer configuration.

## `shared/services/print-service.ts`

Keep `print-service.ts` as the single frontend printing entry point.

All of these must use it:

- automatic receipt printing after sale completion;
- manual receipt printing;
- receipt reprinting from transaction history;
- future print operations.

The service must not contain USB, ESC/POS, CUPS, or OS-specific logic.

Its responsibility is only:

```text
Receipt data
    ↓
Select print mode
    ├── preview → browser preview
    └── silent  → local print agent
```

### Silent printing behavior

Do not automatically open browser print preview when the print agent is
unreachable.

Instead:

```text
Sale completed
      ↓
Print submission fails
      ↓
Transaction remains completed
      ↓
Show printer error/status
      ↓
Allow retry
```

This prevents an unexpected browser modal from interrupting the cashier's
workflow.

The browser preview should remain an explicit/manual fallback rather than an
automatic silent-mode failure path.

> **Product Decision**
>
> Silent mode never falls back automatically to browser printing. If the local
> print agent is unavailable, or a silent print job fails, the transaction
> remains completed and the print job is reported as failed/retryable. The POS
> provides Retry/Dismiss actions (the manual "Print Receipt" action is the retry
> path). Browser printing is available only through explicit Preview / manual
> printing. This keeps `print-service.ts` consistent with the stated production
> behavior and removes the semantic inconsistency of selecting "Silent" and
> silently receiving a browser dialog instead.

---

# Current Hardware Constraint

The development environment does **not currently have a physical 58 mm thermal
printer**.

Therefore, the implementation must not depend on physical printer hardware
during the initial development phase.

The architecture should be developed and tested using a `FileTransport`, while
USB and TCP/IP transports remain production targets to be validated when a
physical printer becomes available.

This gives the project the following progression:

```text
Development now
    ↓
Go Print Agent
    ↓
ESC/POS Renderer
    ↓
FileTransport
    ↓
receipt.bin
    ↓
Validate generated ESC/POS output

Later
    ↓
Physical 58 mm printer
    ↓
USB / TCP Transport
    ↓
Real receipt
```

The receipt renderer must target a **58 mm ESC/POS paper profile**, but it must
not depend on a particular printer model.

The renderer and transport must remain independent:

```text
Receipt
   ↓
ESC/POS Renderer
   ↓
[]byte
   ↓
Transport
   ├── File      ← development / CI
   ├── TCP       ← production target
   └── USB       ← production target
```

## Development Strategy Without Physical Hardware

Development should proceed in stages.

### Stage 1 — FileTransport

Implement the entire print pipeline without physical hardware:

```text
Svelte POS
    ↓
POST /print
    ↓
Go Print Agent
    ↓
Print Queue
    ↓
ESC/POS Renderer
    ↓
FileTransport
    ↓
receipt-<job-id>.bin
```

This allows the following to be developed and tested immediately:

- HTTP API;
- request validation;
- receipt model;
- ESC/POS formatting;
- print queue;
- idempotency;
- retry handling;
- job lifecycle;
- frontend integration;
- silent-print workflow;
- error handling.

### Stage 2 — ESC/POS Output Validation

The generated `.bin` files should be treated as the canonical output of the
renderer.

Tests should verify important ESC/POS sequences such as:

- initialization;
- text encoding;
- alignment;
- bold;
- font sizing;
- item columns;
- totals;
- payment;
- change;
- QR/barcode commands where applicable;
- line feeds;
- paper cut.

Do not require a physical printer for these tests.

### Stage 3 — Physical Printer Integration

Once a 58 mm thermal printer is available:

```text
Existing tested pipeline
        ↓
Replace FileTransport
        ↓
USBTransport / TCPTransport
        ↓
Physical printer
```

The ESC/POS renderer should not need to change merely because the transport
changes.

This is an important architectural requirement.

### Stage 4 — Hardware Compatibility Testing

After acquiring the printer, verify:

- 58 mm paper width;
- supported character encoding;
- printable columns;
- font sizes;
- bold;
- alignment;
- QR code rendering;
- barcode rendering if used;
- paper feed;
- automatic cutter;
- USB communication;
- TCP/IP communication if applicable;
- behavior after printer disconnect/reconnect.

Hardware-specific behavior should be isolated inside the relevant transport or
printer implementation.

---

# Print Agent

## Technology

Replace the current Node print agent with a standalone Go application.

The print agent should be distributed as a single executable and installed as
a local service/daemon on each POS terminal.

Example:

```text
tools/print-agent/
```

### Responsibilities

The print agent owns:

- local HTTP API;
- request validation;
- print queue;
- job lifecycle;
- idempotency;
- ESC/POS rendering;
- printer discovery/configuration;
- printer transports;
- printer health;
- retry handling;
- logging.

The print agent does **not** own:

- sales;
- payments;
- inventory;
- transaction state;
- pricing;
- customer data/business rules.

---

# Print Agent API

## `GET /health`

Returns the health of the agent and its configured printer.

```json
{
  "ok": true,
  "printer": {
    "connected": true,
    "name": "Receipt Printer",
    "type": "usb"
  }
}
```

## `GET /printer`

Returns the configured printer and connection status.

```json
{
  "id": "receipt",
  "name": "XP-80",
  "type": "usb",
  "status": "ready"
}
```

## `POST /print`

Creates an idempotent print job.

```json
{
  "job_id": "print-01JXYZ...",
  "type": "receipt",
  "copies": 1,
  "receipt": {
    "invoice_number": "INV-000123",
    "created_at": "2026-08-26T10:15:00+07:00",
    "items": [
      {
        "name": "Mineral Water",
        "quantity": 2,
        "unit_price": 5000,
        "original_price": 6000,
        "pricing_rule_name": "Promo",
        "pricing_type": "discount"
      }
    ],
    "total_amount": 10000,
    "subtotal_dpp": 9009,
    "tax": 991,
    "payment_method": "split",
    "payments": [
      { "method": "cash", "amount": 5000 },
      { "method": "qris", "amount": 5000, "reference_number": "QR123" }
    ],
    "cash_received": 5000,
    "change_due": 0,
    "customer_name": "Jane",
    "total_savings": 2000
  },
  "branding": {
    "store_name": "My Store",
    "address": "...",
    "phone": "...",
    "header": "...",
    "footer": "Thank you"
  }
}
```

The `receipt` object mirrors the Svelte `ReceiptData` contract already used by the
POS. The agent must treat every monetary field as precomputed **display data** and
must **never recompute** `subtotal_dpp`, `tax`, `total_amount`, or `change_due`.

- Indonesian PPN (11%) is carried by `subtotal_dpp` (DPP) + `tax` (PPN). Do not
  derive tax from a raw subtotal.
- Split tenders are carried by the `payments[]` array; `payment_method` is
  `"split"` when more than one payment is present. The singular `cash_received` /
  `change_due` describe only the cash portion.
- Per-item `original_price` / `pricing_rule_name` drive the savings line
  (`total_savings`). Omit `original_price` when there is no discount.

The endpoint should enqueue the job and return quickly:

```http
202 Accepted
```

```json
{
  "job_id": "print-01JXYZ...",
  "status": "queued"
}
```

## `GET /print/jobs/{job_id}`

Returns:

```text
queued
printing
completed
failed
```

## `POST /print/jobs/{job_id}/retry`

Retries a failed print job without creating a new logical job.

---

# Idempotency

Print jobs must be idempotent.

A network failure can occur after the agent has printed a receipt but before the
frontend receives the HTTP response. A retry must therefore not blindly print
the same job twice.

```text
First request:
  job-123 → create + print

Retry:
  job-123 → return existing job
```

A transaction number and a print job ID are different concepts.

```text
Transaction:
  TRX-000123

Print jobs:
  JOB-001 → original receipt
  JOB-002 → cashier reprint
  JOB-003 → customer copy
```

The `job_id` is generated by the **frontend** before calling `POST /print`, not by
the agent. The agent treats `job_id` as the authoritative deduplication key:

- first submission with a new `job_id` → create + enqueue + print;
- any subsequent submission with the **same `job_id`** → return the existing job
  status (do **not** re-enqueue and do **not** print again);

A secondary natural key, `invoice_number + receipt_type + copy_index`, MAY be used
to catch retries that lost the `job_id`, but `job_id` remains authoritative.

---

# Print Queue

Printing must be asynchronous.

```text
POST /print
    ↓
Validate
    ↓
Create/lookup job
    ↓
Queue
    ↓
202 Accepted
```

A background worker processes the queue:

```text
Print Queue
    ↓
Get next job
    ↓
Render ESC/POS
    ↓
Connect printer
    ↓
Write bytes
    ↓
Success → completed
Failure → failed/retryable
```

---

# Receipt Representation

Do not send HTML to the print agent.

Do not make the print agent convert HTML/PDF into ESC/POS.

Use structured receipt data:

```text
Receipt Data
    ↓
ESC/POS Renderer
    ↓
[]byte
    ↓
Printer Transport
```

The backend remains the source of truth for transaction data. The agent must
never calculate business totals or alter transaction values.

---

# ESC/POS Renderer

Recommended abstraction:

```go
type Renderer interface {
    Render(receipt Receipt) ([]byte, error)
}
```

The ESC/POS implementation should handle:

- printer initialization;
- text;
- alignment;
- bold;
- font size;
- columns;
- separators;
- QR codes;
- barcodes;
- line feeds;
  - paper cutting;
  - character encoding / codepage selection (enable a codepage such as CP858 or
    CP437 so Indonesian `Rp` and accented store-name/footer text render
    correctly).

The renderer must be independently testable without a physical printer.

---

# Printer Transport

Separate ESC/POS rendering from physical printer communication.

Because there is currently no physical printer available, `FileTransport`
should be implemented first.

USB and TCP/IP are production targets and should be implemented and tested
after the required hardware is available.


```go
type Transport interface {
    Write([]byte) error
    Close() error
}
```

Architecture:

```text
EscPosRenderer
      ↓
  []byte
      ↓
Transport
  ├── USBTransport
  ├── TCPTransport
  └── FileTransport
```

---

# USB Transport

USB should be supported as a production transport, but implementation and
hardware validation can be deferred until a physical 58 mm printer is
available.


The Go agent communicates directly with the USB thermal printer and sends raw
ESC/POS bytes.

The implementation must handle:

- vendor ID;
- product ID;
- USB interface;
- endpoint discovery;
- device connection;
- connection recovery;
- OS-level permissions.

Linux deployment may require udev rules so the print-agent service can access
the printer without running as root.

The exact USB implementation should be validated against the actual printer
models selected for deployment.

---

# TCP/IP Transport

Support raw TCP/IP thermal printers as a production target.

This transport can be implemented before owning a printer by testing against a
local TCP test server that captures the bytes sent by the agent. Final
validation still requires an actual compatible thermal printer.


```text
Go Print Agent
      ↓
TCP
      ↓
192.168.x.x:9100
      ↓
Thermal Printer
```

The agent sends the same ESC/POS bytes used by USB transport.

---

# File Transport

Keep a file transport for development and automated tests.

```text
Receipt
   ↓
ESC/POS
   ↓
FileTransport
   ↓
receipt.bin
```

This allows ESC/POS output to be tested without physical printer hardware.

---

# PDF / HTML Output

The existing HTML/PDF output may remain as a development/testing feature.

It must not be the primary production thermal-print path.

Production thermal printing should be:

```text
Receipt
   ↓
ESC/POS
   ↓
Thermal Printer
```

The browser preview remains useful for development, manual preview, environments
without the print agent, and receipt-layout debugging.

---

# Security

The print agent listens only on localhost by default:

```text
127.0.0.1
```

Do not expose the print API on `0.0.0.0` unless explicitly required.

The agent should validate the browser origin and/or use a locally generated
authentication token:

```http
Authorization: Bearer <local-agent-token>
```

This prevents arbitrary websites from silently submitting print jobs.

---

# Printer Configuration

Move away from using environment variables as the primary production
configuration.

Environment variables may still be useful for development and CI.

Production configuration should represent the actual printer:

```yaml
printer:
  id: receipt
  type: usb
  vendor_id: 0x1234
  product_id: 0x5678
```

or:

```yaml
printer:
  id: receipt
  type: tcp
  host: 192.168.1.50
  port: 9100
```

The frontend should refer to a logical printer ID rather than hard-coding
hardware details.

---

# Suggested Go Project Structure

```text
tools/
└── print-agent/
    ├── cmd/
    │   └── print-agent/
    │       └── main.go
    │
    ├── internal/
    │   ├── api/
    │   │   ├── handler.go
    │   │   └── routes.go
    │   │
    │   ├── queue/
    │   │   ├── queue.go
    │   │   └── worker.go
    │   │
    │   ├── receipt/
    │   │   ├── model.go
    │   │   └── renderer.go
    │   │
    │   ├── escpos/
    │   │   ├── encoder.go
    │   │   └── commands.go
    │   │
    │   ├── transport/
    │   │   ├── transport.go
    │   │   ├── usb.go
    │   │   ├── tcp.go
    │   │   └── file.go
    │   │
    │   ├── printer/
    │   │   └── manager.go
    │   │
    │   └── config/
    │       └── config.go
    │
    ├── go.mod
    └── README.md
```

---

# Frontend Components

Keep:

```text
shared/
├── stores/
│   └── printConfig.svelte.ts
│
└── services/
    └── print-service.ts

app/
└── components/
    └── PrintSettings.svelte
```

The print settings UI should expose:

```text
Receipt Printer
────────────────────────────

Status: ● Ready

Printer: XP-80
Connection: USB

[ Test Print ]

Mode:
(●) Silent
( ) Preview
```

Detailed printer configuration belongs in an appropriate settings/admin area,
not in the transaction cart.

---

# Error Handling

Printing errors must not affect transaction state.

```text
Transaction
  COMPLETED

Print
  FAILED
```

The system should provide:

- retry;
- reprint;
- printer status;
- useful error logging.

Avoid automatically opening the browser print dialog because the agent failed.

---

# Reprint Behavior

Reprinting is a new print job referencing an existing completed transaction.

```text
Transaction TRX-000123
        │
        ├── JOB-001 original
        │
        └── JOB-002 reprint
```

A reprint must never modify the original transaction.

---

# Testing

The absence of physical printer hardware must not block development.

The test strategy therefore has three layers:

```text
Unit Tests
    ↓
FileTransport / captured ESC/POS bytes
    ↓
TCP test server
    ↓
Physical printer integration tests
```

## Print Agent unit tests

Test independently:

### ESC/POS renderer

- initialization;
- alignment;
- bold;
- item formatting;
- totals;
- payment;
- change;
- QR/barcode;
- paper cut;
- receipt width.

### Queue

- enqueue;
- ordering;
- success;
- failure;
- retry;
- duplicate job ID;
- concurrent requests.

### Transports

- file transport;
- TCP transport with a local test server;
- USB transport with hardware/integration tests once a physical printer is
  available.

### API

- health;
- printer status;
- print;
- idempotent duplicate submission;
- job status;
- retry.

## Frontend tests

`print-service.test.ts` should cover:

```text
preview:
  → opens preview

silent:
  → POSTs to local agent
  → does not call window.print()

agent unavailable:
  → does not open preview
  → reports print failure

duplicate submission:
  → preserves job identity
```

## E2E

Implemented as `tests/e2e/silent-print.spec.ts`, which boots the real Go print
agent (`file` transport) and drives a POS sale in silent mode:

```text
Complete sale (silent mode)
    ↓
No browser print dialog (window.print trapped + asserted 0 calls)
    ↓
Print request reaches agent (POST /print)
    ↓
Print job completed (ESC/POS .bin written)
```

For CI, use the `file` transport rather than physical hardware (the agent is
started with `go run ./cmd/print-agent` on a dedicated port).

---

# Deployment

The Go print agent should run as a local service.

### Linux

```text
systemd
   ↓
pos-print-agent
```

### Windows

```text
Windows Service
   ↓
pos-print-agent.exe
```

The POS browser communicates with:

```text
http://127.0.0.1:<port>
```

The agent should start automatically with the operating system.

---

# Migration from Current Implementation

Current:

```text
Svelte POS
    ↓
print-service
    ↓
POST /print
    ↓
Node print-agent
    ├── file
    ├── pdf
    └── thermal/serial
```

Target:

```text
Svelte POS
    ↓
print-service
    ↓
POST /print
    ↓
Go print-agent
    ↓
Print Queue
    ↓
ESC/POS Renderer
    ↓
Transport
    ├── USB
    ├── TCP
    └── File (test)
```

Migration should be incremental:

1. Freeze the frontend `print-service` contract.
2. Define the new print-agent API contract.
3. Implement Go agent HTTP API.
4. Implement receipt model.
5. Implement the 58 mm ESC/POS receipt profile.
6. Implement `FileTransport`.
7. Implement ESC/POS byte-level tests using generated `.bin` output.
8. Implement queue and idempotency.
9. Switch the frontend from the Node agent to the Go agent. This is **not** a
   drop-in swap: remove the silent-mode fallback to `window.print()`, adopt the
   `202 Accepted` + job-status polling model, and add the `Retry` / `Dismiss` UI
   for agent-unavailable and failed jobs (see Error Handling).
10. Add silent-print E2E coverage using `FileTransport`.
11. Implement TCP transport and test it with a local TCP capture server.
12. Acquire a compatible 58 mm thermal printer.
13. Validate the receipt output on the physical printer.
14. Implement and validate USB transport.
15. Add printer health/status.
16. Add retry/reprint handling.
17. Remove the Node agent after the Go agent is validated.
18. Move printer configuration from browser `localStorage` to per-register backend configuration when register management is ready.

---

# Out of Scope

The following are not part of the initial production implementation:

- WebUSB direct printing from the browser;
- browser kiosk/silent-print hacks;
- remote printer management;
- cloud printing;
- printer fleet management;
- PDF as the production thermal-print path.

WebUSB remains a possible alternative for tightly controlled Chromium-only
deployments, but it introduces browser compatibility, secure-context, device
authorization, and deployment-policy considerations. It should not be the
primary architecture when a local print agent is acceptable.

---

# Final Recommendation

The production architecture should be:

```text
Go Backend
    ↓
Svelte POS
    ↓
print-service.ts
    ↓
localhost HTTP
    ↓
Go Print Agent
    ↓
Idempotent Print Queue
    ↓
ESC/POS Renderer
    ↓
Printer Transport
    ├── USB
    └── TCP/IP
    ↓
Thermal Printer
```

The critical UX requirement is:

```text
PAY
 ↓
TRANSACTION COMMITTED
 ↓
PRINT JOB QUEUED
 ↓
POS READY FOR NEXT CUSTOMER
 ↓
RECEIPT PRINTS IN BACKGROUND
```

No browser print preview.

No cashier confirmation.

No rollback of a successful sale when the printer is unavailable.

The browser preview remains available as an explicit/manual mode rather than
being the default production path.
