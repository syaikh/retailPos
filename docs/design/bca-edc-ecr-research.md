# BCA EDC / ECR as a Supported Payment — Research Note

**Purpose:** Gather what BCA EDC/ECR is and how it can become a supported payment method in this POS app.
**Status:** Research / pre-design. Some BCA specifics are proprietary (NDA) and must be obtained from BCA merchant onboarding.
**Verified vs. assumed:** Items marked ✅ are confirmed from public sources; ⚠️ require BCA's NDA spec / merchant agreement.

---

## 1. Terminology (BCA context)

- **EDC (Electronic Data Capture):** the physical BCA card terminal at the counter (debit/credit, often Ingenico/Verifone/PAX-branded, running BCA's acquiring firmware).
- **ECR (Electronic Cash Register):** the merchant's POS / cash register — i.e., **this app**.
- **BCA ECR Interface:** the automated data interface between a merchant ECR and a BCA EDC terminal. It processes a transaction via a **request/response message format** to enable automated reconciliation, reduce double-data-entry, improve speed and security. ✅ (confirmed via the BCA ECR interface reference — a 7-page spec describing interface flow, accepted card types, transactions, **serial-port messaging**, and implementation).

## 2. Two integration tiers

### Tier A — Manual EDC reference capture (quick win, ✅ supported today)
The cashier runs the card on the BCA EDC terminal manually, then keys the **approval/reference number** (printed on the EDC receipt) into the POS.
- This app **already supports it**: the `CARD` payment method requires a reference and auto-generates an `EDC/...` reference number (`CheckoutModal.generateRefNumber`). The `reference_number` on `PaymentAllocation` is exactly this.
- No terminal connection needed. Fully viable now.
- Limitation: no automatic amount-send or approval verification; relies on cashier accuracy.

### Tier B — Full ECR auto-integration (⚠️ requires BCA NDA spec + middleware)
The POS sends the sale amount to the BCA EDC terminal over a link; the terminal captures the card and returns approval (authorization code, card type, reference, masked PAN, receipt text), which the POS records and prints.
- **Transport (industry-standard, ✅):** serial **RS232** or **Ethernet TCP/IP** between ECR and EFT-POS. Communication is **always initiated by the ECR (POS)**, request→response, synchronous.
- **Protocol (⚠️ bank/vendor-specific):** typically a packet format with `STX`/`ETX` + `LRC` checksum (as in Nexi/ZVT/Comgate ECR-EFT specs). BCA's exact byte-level command set is **proprietary and provided under merchant/NDA** — not publicly published.
- **Flow:** `Terminal Status` (link check) → `Payment` (send amount) → terminal processes → `Result` (approval code, card BIN, type Debit/Credit, reference) → optional receipt data returned to ECR for printing.
- **Architecture implication:** a browser/Svelte frontend **cannot** speak RS232/LAN to a terminal directly. Needs a **local bridge** (desktop shell / native helper / Go service on the store machine) that the POS backend talks to, which in turn drives the terminal. The Go backend (`internal/sale`) would expose an endpoint (e.g., `POST /pos/edc/authorize`) that proxies to the bridge.

## 3. BCA API / SNAP vs EDC — do not confuse

| Capability | BCA API / SNAP (`developer.bca.co.id`) | BCA EDC + ECR |
|------------|----------------------------------------|---------------|
| Card-present (counter swipe/tap) | ❌ | ✅ |
| Virtual Account / QRIS / Transfer | ✅ | ❌ (QRIS is separate SNAP channel) |
| Integration | Server-to-server REST, OAuth/SNAP | Terminal link (RS232/LAN) |
| Use case | E-commerce / online | In-store retail POS |

If the goal is also QRIS/VA, that is a **different** integration (SNAP-based) and can be a parallel payment method — but EDC is the card-terminal path.

## 4. Mapping to this app's payment model

- Payment methods are dynamic (`/payment-methods`): `{ code, label, requiresReference }`.
- `PaymentAllocation { payment_method_code, amount, reference_number }` already carries what an EDC txn needs.
- **Tier A mapping:** add a payment method (e.g., code `BCA_EDC` or reuse `CARD`) with `requiresReference = true`; cashier enters the terminal's approval reference. Already works.
- **Tier B mapping:** extend the CARD/EDC allocation row to (optionally) trigger `POST /pos/edc/authorize`, display terminal status, and auto-fill `reference_number` + `approval_code` from the terminal response.

## 5. What we need from BCA (⚠️)

1. **Merchant acquiring account** + a BCA EDC terminal.
2. **BCA ECR Interface specification** (command set, packet format, baud/parity or IP/port) — obtained via BCA merchant/onboarding or the terminal vendor.
3. Terminal **IP/port or COM port** config and "ECR Integration" mode enabled (Nexi terms: "ECR Integration and standalone" vs "ECR Integration only").
4. Test cards / sandbox terminal for development.

## 6. Recommended phased approach

- **Phase 0 (now):** Add `BCA_EDC` (or use `CARD`) payment method with required reference — Tier A. Zero terminal work; closes the "card payment with reference" need immediately.
- **Phase 1 (later):** Build a local EDC bridge + backend `authorize` endpoint for Tier B auto-integration once BCA's ECR spec is in hand.
- **Parallel:** If QRIS/VA also wanted, scope a separate SNAP-based method.

## 7. Open questions for the product owner

- Manual reference entry (Tier A) vs. full terminal integration (Tier B)? Tier B needs BCA's NDA spec.
- One combined `CARD` method, or a distinct `BCA_EDC` method (also possible: `BCA_DEBIT` / `BCA_CREDIT`)?
- Is a local desktop/native bridge acceptable (required for Tier B), or is this a pure web/PWA deployment (Tier A only)?

## 8. Sources

- BCA ECR Interface reference (Scribd, 7pp): ECR↔BCA EDC Terminal, serial messaging, request/response.
- Nexi / Comgate ECR-EFT integration guides: RS232/TCP, ECR-initiated request/response, STX/ETX/LRC, Terminal Status / Payment / Result flow (industry-standard pattern BCA EDC follows).
- `developer.bca.co.id` — BCA Open Banking / SNAP APIs (VA, QRIS, transfer) — distinct from EDC.
- App source: `CheckoutModal.svelte` (`generateRefNumber` → `EDC/...`), `PaymentAllocation` type, `internal/sale/service.go` validation.
