# BCA EDC/ECR Payment Integration — Implementation Plan (Tier B)

**Status:** Build-ready design. Phase 1 is fully implementable without BCA's NDA spec (uses a Mock adapter); only the final protocol decoding (Phase 2) waits on BCA.
**Related docs:** `docs/design/bca-edc-ecr-research.md` (background/research).

---

## 1. Architecture (recommended)

- **No browser-native bridge needed.** The SPA uses `baseURL: '/api'` (relative) and the Go backend serves both the API and the SPA. The **Go backend talks to the BCA EDC terminal directly** (TCP/IP or RS232 via `go.bug.st/serial`), keeping all card logic server-side and testable.
- **No native shell required (minimalist, Option A).** Deployment = the Go `server` runs on the counter PC (wired to the BCA EDC terminal) and the Svelte SPA is opened in a normal browser at `http://localhost:<port>/`. Since all terminal comms live in the Go backend, a desktop wrapper adds nothing to the integration.
- **Kiosk mode = optional, later.** Locking the device to a single full-screen app (no URL bar / no desktop access, auto-start on boot) is a pure packaging concern. It can be added afterwards (Tauri/Electron or OS kiosk settings) **without changing any payment code**, so it is out of scope for this plan.
- **Payment method:** add a single `BCA_EDC` method (`requires_reference=true`), keeping existing `CARD` for manual/generic cards.

---

## 2. Data model (migrations)

- `payment_methods`: insert a `BCA_EDC` row (mirrors `CARD`, `requires_reference=true`, sort after CARD).
  - Columns (from `database/migrations/000_squash.sql:143`): `id, code, name, is_active, requires_reference, sort_order, created_at, updated_at`.
- `sale_payments` (`database/migrations/000_squash.sql:736`): add nullable
  - `card_type varchar(20)`
  - `approval_code varchar(50)`
  - `masked_pan varchar(20)`
- `app_settings` (seeded by migration `005_app_settings.sql`): keys
  - `edc_enabled`, `edc_mode` (`tcp`|`serial`), `edc_host`/`edc_port` or `edc_com`/`edc_baud`, `edc_terminal_id`, `edc_merchant_id`, `edc_timeout_ms`.

---

## 3. Backend — new `internal/edc` package

- `types.go`: `Adapter` interface — `Connect`, `Status`, `Authorize(req{Amount int}) → resp{ApprovalCode, CardType, Reference, MaskedPAN, ReceiptLines []string}`, `Cancel`, `Settle`.
- `transport.go`: connection management (TCP `net` or serial) + `STX/ETX/LRC` framing skeleton + timeout/retry.
- `bcaecr.go`: BCA ECR protocol implementation — transport + **placeholder command set (TODO: fill from BCA NDA spec)**.
- `mock.go`: `MockAdapter` returning canned approvals — lets the whole flow be built/tested **without hardware**.
- `service.go` + `handler.go`: routes
  - `GET /api/edc/status`
  - `POST /api/edc/authorize`
  - `POST /api/edc/cancel`
  - (cashier auth). Adapter selected by config (`mock` | `bcaecr`).

---

## 4. Backend — `internal/sale` extensions

- `internal/sale/domain.go` `Payment`: add `CardType`, `ApprovalCode`, `MaskedPAN`.
- `internal/sale/repository.go` `CreateSalePayments` (line ~165): persist new columns.
- `internal/sale/presenter.go` (`presentSale` ~line 22): return new fields in the payment payload.
- `validatePayments` (`internal/sale/service.go:186`) already enforces `reference_number` for `requires_reference` methods → `BCA_EDC` is covered (approval code becomes the reference).

---

## 5. Frontend

- `web/src/modules/pos/types/index.ts` `PaymentAllocation`: add optional `card_type`, `approval_code`, `masked_pan`.
- `web/src/modules/pos/components/CheckoutModal.svelte`:
  - For `BCA_EDC`, show a **"Charge"** button → calls `POST /api/edc/authorize` → **status modal** ("Insert / tap card…", spinner).
  - On success: auto-fill `reference_number = approvalCode`, set `card_type`/`masked_pan`, show success (card type + last4).
  - On failure: terminal error + **manual-entry fallback** (Tier A) when the terminal is offline.
- New `web/src/modules/pos/services/edc-service.ts` API client for `/api/edc/*`.
- Receipt (`web/src/app/components/ReceiptPrintOverlay.svelte` / `web/src/shared/stores/printReceipt.svelte.ts`): print approval code, card type, last4.
- i18n `web/src/shared/i18n/en.ts` & `id.ts`: `edcCharge`, `edcInsertCard`, `edcError`, `approvalCode`, `cardType`, etc.

---

## 6. Testing

- `MockAdapter` → unit tests for the authorize→checkout flow; integration test adding a `BCA_EDC` payment end-to-end without hardware.
- E2E with EDC in mock mode.

---

## 7. Phasing

- **Phase 1 (now):** migrations + `internal/edc` skeleton + MockAdapter + routes + frontend charge/status modal + receipt + i18n. Fully buildable/testable, no BCA spec required.
- **Phase 2 (later):** real BCA ECR command set once the NDA spec is obtained; swap `mock`→`bcaecr`; configure TCP/serial.
- **Phase 3 (separate, later):** QRIS / Virtual Account via BCA SNAP API (server-to-server, different integration).

---

## 8. Decisions / assumptions

- **No native shell required (Option A, minimalist).** Pure localhost-browser deployment; kiosk mode is an optional future packaging step, explicitly out of scope for this plan.
- Terminal transport TCP vs RS232 is **config-driven** (Go supports both).
- Go server serves the built SPA in prod (so the browser just hits localhost) — verify in `cmd/server/main.go` static serving; the dev model already proxies `/api` to the Go backend regardless.
- Real BCA ECR byte-level protocol is **NDA** → placeholder + Mock until provided.
