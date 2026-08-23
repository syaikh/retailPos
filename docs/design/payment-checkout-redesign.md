# Payment & Split-Payment — Redesign Plan

**Status:** Phase 1 (frontend quick wins) specified with exact edits. Phases 2–3 deferred (require backend decision on cash change).
**Scope:** `web/src/modules/pos/components/CheckoutModal.svelte` + `web/src/shared/i18n/{en,id}.ts`.
**Goal:** Remove the highest-frequency cashier errors in the payment/split-payment flow with low-risk frontend changes, without waiting on the backend cash-change workstream (C1).

---

## 1. Design Mockups (target state)

### 1.1 CheckoutModal — main view (fixes U1, U3, U4)

```
┌───────────────────────────────────────────────────────────────────────┐
│ Payment                                                                  │ [X]
├──────────────────────────────────────┬──────────────────────────────────┤
│ ITEMS (col-span-7, scroll)           │  TOTAL        Rp 85.000          │
│  Item A            1 × 10.000  10.000│  ───────────────────────────────  │
│  Item B (disc)     2 × 40.000  80.000│  [ CASH ][ CARD ][ QRIS ]         │
│                                       │  [ E-WALLET ][ GOPAY ][ BCA ]    │
│                                       │  👤 Customer (optional)        > │
│                                       │  ───────────────────────────────  │
│                                       │  PAYMENT ALLOCATION      [reset] │
│                                       │  ┌─ CASH ──────────────── [🗑] ┐ │
│                                       │  │ Amount   [ 85.000      ]    │ │
│                                       │  │ [5K][10K][20K][50K][100K]   │ │
│                                       │  │ [ Exact ][ Reset ]          │ │
│                                       │  └─────────────────────────────┘ │
│                                       │  ┌─ CARD ──────────────── [🗑] ┐ │
│                                       │  │ Amount   [ 0           ]    │ │
│                                       │  │ Ref *    [ auto EDC/..  ]    │ │
│                                       │  └─────────────────────────────┘ │
├──────────────────────────────────────┴──────────────────────────────────┤
│  TOTAL 85.000 │ PAID 85.000 │ REMAINING 0 ✓       [ Cancel ][ Done Enter ]│
└───────────────────────────────────────────────────────────────────────┘
```
- **Sticky summary bar** (new, U1): `TOTAL | PAID | REMAINING` always visible at the bottom.
- **CASH row defaults to `totalAmount`** (U3) → pure-cash sale = press Enter.
- REMAINING turns green ✓ at 0; red ⚠ when ≠ 0.

### 1.2 Summary-bar state variants (fixes U4)

```
EXACT (normal):   TOTAL 85.000 │ PAID 85.000 │ REMAINING    0 ✓
SHORT (under):    TOTAL 85.000 │ PAID 70.000 │ REMAINING 15.000 ⚠   (red)
OVER (overshot):  TOTAL 85.000 │ PAID 90.000 │ OVERPAID  5.000 ⚠   (red, press Reset)
OVER-TENDER (C1): TOTAL 85.000 │ PAID 100.000 │ CHANGE 15.000 💵   (green, Phase 2)
```
- Replaces the silent disabled-Done dead-end with explicit, colored feedback.

### 1.3 Confirm step (fixes U2 — Phase 2, shown for context)

```
┌──────────────────────────────────────┐
│  Confirm Payment                      │
│  ───────────────────────────────────  │
│  Total        Rp 85.000               │
│  Cash         Rp 100.000              │
│  Change       Rp 15.000               │
│  ───────────────────────────────────  │
│  [ Cancel ]      [ Confirm ✓ Enter ]  │
└──────────────────────────────────────┘
```

### 1.4 Receipt with Change (fixes K3 — Phase 2, depends on C1)

```
         TOKO SEJAHTERA
   ─────────────────────────
   TOTAL              85.000
   CASH              100.000
   CHANGE             15.000   ← NEW line (already in type, never rendered)
   ─────────────────────────
```

---

## 2. Phase 1 — Exact Code-Level Edit Instructions (frontend only)

All paths relative to `web/src`. No backend change required for Phase 1.

### Edit 1 — Default opening CASH amount to total (U3)
File: `modules/pos/components/CheckoutModal.svelte`

```svelte
// OLD (line ~146, inside $effect when showCheckoutModal)
      allocations = [{ id: 'a1', methodCode: 'CASH', amount: 0, referenceNumber: '' }];

// NEW
      allocations = [{ id: 'a1', methodCode: 'CASH', amount: totalAmount, referenceNumber: '' }];
```
**Why:** For the common pure-cash sale, the cashier can now press Enter immediately instead of typing the total or pressing F7.

### Edit 2 — Gate completion on required references (S2)
File: `modules/pos/components/CheckoutModal.svelte`

```svelte
// OLD (line ~51)
  const canComplete = $derived(remainingBalance === 0 && allocations.length > 0);

// NEW
  const canComplete = $derived(
    remainingBalance === 0 &&
    allocations.length > 0 &&
    allocations.every(a => {
      const opt = paymentOptions.find(o => o.id === a.methodCode);
      return !(opt?.requiresReference && !a.referenceNumber?.trim());
    })
  );
```
**Why:** Prevents a failed submit when a `requiresReference` method (CARD/EDC/E_WALLET) has an emptied reference. The "Done" button is already disabled by `!canComplete`.

### Edit 3 — Add sticky summary bar (U1) + over/short messaging (U4)
File: `modules/pos/components/CheckoutModal.svelte`
Replace the existing "Fixed bottom: Actions" block (the `<div class="shrink-0">` containing the Cancel/Done `<Button>`s) with:

```svelte
            <!-- Fixed bottom: Summary + Actions -->
            <div class="shrink-0">
              <!-- Summary bar: Total | Paid | Remaining (U1) -->
              <div class="grid grid-cols-3 gap-2 px-1 pb-2 mb-2 border-b border-border/30 text-center">
                <div>
                  <p class="text-[10px] uppercase tracking-wider text-text-muted">{labels.total}</p>
                  <p class="text-sm font-semibold text-text-primary tabular-nums">{totalAmount.toLocaleString('id-ID')}</p>
                </div>
                <div>
                  <p class="text-[10px] uppercase tracking-wider text-text-muted">{labels.paid}</p>
                  <p class="text-sm font-semibold text-text-primary tabular-nums">{totalAllocated.toLocaleString('id-ID')}</p>
                </div>
                <div>
                  <p class="text-[10px] uppercase tracking-wider text-text-muted">{labels.remaining}</p>
                  {#if remainingBalance > 0}
                    <p class="text-sm font-bold text-danger tabular-nums">{remainingBalance.toLocaleString('id-ID')} &#9888;</p>
                  {:else if remainingBalance < 0}
                    <p class="text-sm font-bold text-danger tabular-nums">{labels.overpaid} {Math.abs(remainingBalance).toLocaleString('id-ID')}</p>
                  {:else}
                    <p class="text-sm font-bold text-success tabular-nums">&#10003;</p>
                  {/if}
                </div>
              </div>
              <!-- Actions -->
              <div class="flex gap-2 pt-2">
                <Button
                  variant="secondary"
                  class="flex-1 py-2"
                  onclick={close}
                >
                  {labels.cancelEsc}
                </Button>
                <Button
                  variant="success"
                  class="flex-1 py-2"
                  disabled={cart.length === 0 || !canComplete}
                  onclick={handleFinalize}
                >
                  <Check size={14} />
                  {labels.doneEnter}
                </Button>
              </div>
            </div>
```
**Why:** The cashier gets constant, glanceable feedback on paid vs. remaining; over/short states are explicit instead of a silent disabled button.

### Edit 4 — Required-reference visual cue (S2)
File: `modules/pos/components/CheckoutModal.svelte`
Replace the reference input block (the `{#if !isCash && opt?.requiresReference}` block) with:

```svelte
                  {#if !isCash && opt?.requiresReference}
                    <div>
                      <label for="alloc-ref-{alloc.id}" class="text-[10px] text-text-muted mb-1 block">
                        {labels.referenceNumber} <span class="text-danger">*</span>
                      </label>
                      <input
                        id="alloc-ref-{alloc.id}"
                        type="text"
                        bind:value={alloc.referenceNumber}
                        placeholder={labels.referenceNumberPlaceholder}
                        class="w-full px-2 py-1.5 rounded-lg border text-xs text-text-primary placeholder:text-text-muted outline-none focus:border-primary-light transition-colors {alloc.referenceNumber?.trim() ? 'border-border bg-surface' : 'border-danger bg-danger-subtle/20'}"
                      />
                    </div>
                  {/if}
```
**Why:** Makes the required field obvious and shows an error state before submit.

### Edit 5 — Add `paid` i18n label (K1)
File: `shared/i18n/en.ts` (near `remaining`, line 1094)

```ts
// OLD
  remaining: 'Remaining',

// NEW
  remaining: 'Remaining',
  paid: 'Paid',
```
File: `shared/i18n/id.ts` (near `remaining`, line 1094)

```ts
// OLD
  remaining: 'Sisa',

// NEW
  remaining: 'Sisa',
  paid: 'Dibayar',
```
**Why:** `labels.paid` is referenced by the new summary bar (Edit 3). `remaining`, `overpaid`, `outstanding`, `changeDue` already exist.

---

## 2b. Phase 1b — Split Payment Selection Improvements

Extends Phase 1 with refinements to **how a cashier picks and adds payment methods** inside `CheckoutModal.svelte` (the split-payment surface). Phase 1 covers the amount/summary bar; Phase 1b covers method *selection* ergonomics.

### 2b.0 Assessment (current selection behavior)

Good: click-to-add is fast; re-clicking an added method focuses its amount (no duplicate); used chips are highlighted; reference numbers auto-generate.
Weak: no keyboard method hotkeys; the new row's amount input isn't auto-focused; the "prefill new row with `remainingBalance`" only works cleanly for a 2-method split; you can add redundant 0-amount methods once fully allocated; no "split equally / fixed amount" helper; the "already added" affordance is subtle. Fine for 1–2 methods, frictionful for 3+ and for power users.

### 2b.1 Mockup — integrated redesigned modal (Phase 1 + 1b)

Shows the full end state: summary bar (U1/U3/U4) **and** the selection improvements (S-A…S-F) on a real 2-method split.

```
┌───────────────────────────────────────────────────────────────────────────┐
│ Payment — Split                                                              │ [X]
├────────────────────────────────────────┬────────────────────────────────────┤
│ ITEMS (col-span-7, scroll)             │  TOTAL        Rp 85.000            │
│  Item A            1 × 10.000  10.000  │  ─────────────────────────────────  │
│  Item B (disc)     2 × 40.000  80.000  │  [1 CASH ][2 CARD ][3 QRIS ]  ←S-B  │
│                                        │  [4 E-WALLET][5 GOPAY][6 BCA]       │
│                                        │         [ ⟳ Split Equally ]  ←S-C   │
│                                        │  👤 Customer (optional)          >   │
│                                        │  ─────────────────────────────────  │
│                                        │  PAYMENT ALLOCATION        [reset]  │
│                                        │  ┌─ CASH  ✓ Added ────── [🗑] ┐ ←S-E │
│                                        │  │ Amount [ 50.000      ]          │ │
│                                        │  │ [5K][10K][20K][50K][100K]        │ │
│                                        │  │ [ Exact ][ Reset ]              │ │
│                                        │  └─────────────────────────────────┘ │
│                                        │  ┌─ CARD  ✓ Added ────── [🗑] ┐ ←S-E │
│                                        │  │ Amount [ 35.000      ]          │ │
│                                        │  │ Ref *  [ auto EDC/..  ] ←S2       │ │
│                                        │  └─────────────────────────────────┘ │
│                                        │  (other chips greyed: fully paid)←S-D│
├────────────────────────────────────────┴────────────────────────────────────┤
│  TOTAL 85.000 │ PAID 85.000 │ REMAINING 0 ✓         [ Cancel ][ Done Enter ] │ ←U1
└───────────────────────────────────────────────────────────────────────────┘
```

**Selection flow annotations**
- **S-B** Number hotkeys `1–9` on chips → add the Nth method without the mouse.
- **S-C** `⟳ Split Equally` distributes the total across current methods (remainder on first row).
- **S-E** `✓ Added` on each row makes the chip↔row link explicit.
- **S-D** Once `remaining == 0`, not-yet-added chips grey out; reduce an existing amount to re-enable.
- **S-A** Adding/focusing a method auto-focuses its amount input (not shown; behavioral).
- **U3** CASH opened at `total`; here edited to 50.000 to start the split.

**Summary-bar variants** (bottom bar, U1/U4 + C1 preview)

```
EXACT (split above): TOTAL 85.000 │ PAID 85.000 │ REMAINING   0 ✓
SHORT:              TOTAL 85.000 │ PAID 70.000 │ REMAINING 15.000 ⚠  (red, Done disabled)
OVER (no changeyet):TOTAL 85.000 │ PAID 90.000 │ OVERPAID  5.000 ⚠  (red, press Reset)
OVER-TENDER (C1):   TOTAL 85.000 │ PAID 100.000 │ CHANGE 15.000 💵  (green)
```
When `remaining == 0` the not-yet-added chips disable (S-D); when a method is added its amount field auto-focuses (S-A).

### 2b.2 Prioritized table

| # | Improvement | Priority | Effort |
|---|-------------|----------|--------|
| S-A | Auto-focus new allocation's amount input on add | Quick win | S |
| S-B | Number-key (1–9) shortcuts to add the Nth method | Should | S |
| S-D | Disable not-yet-added chips when fully allocated (no redundant 0 rows) | Should | S |
| S-C | "Split Equally" helper across current methods | Should (high value) | M |
| S-E | Clearer "✓ Added" affordance on used rows | Nice | S |
| S-F | Group/scroll the method grid if many methods | Conditional | S–M |

---

### Edit 6 — Auto-focus new allocation amount on add (S-A)
File: `modules/pos/components/CheckoutModal.svelte` (`addAllocation`, ~lines 82–97)

```svelte
// OLD
  function addAllocation(methodCode: string) {
    const existing = allocations.find(a => a.methodCode === methodCode);
    if (existing) {
      const input = document.getElementById(`alloc-amount-${existing.id}`);
      input?.focus();
      return;
    }
    const opt = paymentOptions.find(o => o.id === methodCode);
    const allocAmount = remainingBalance > 0 ? remainingBalance : 0;
    allocations = [...allocations, {
      id: `a${nextId++}`,
      methodCode,
      amount: allocAmount,
      referenceNumber: opt?.requiresReference ? generateRefNumber(methodCode) : '',
    }];
  }

// NEW
  function addAllocation(methodCode: string) {
    const existing = allocations.find(a => a.methodCode === methodCode);
    if (existing) {
      const input = document.getElementById(`alloc-amount-${existing.id}`);
      input?.focus();
      return;
    }
    // S-D: do not add a redundant method once the total is fully allocated.
    if (remainingBalance <= 0) return;
    const opt = paymentOptions.find(o => o.id === methodCode);
    const allocAmount = remainingBalance > 0 ? remainingBalance : 0;
    const newId = `a${nextId++}`;
    allocations = [...allocations, {
      id: newId,
      methodCode,
      amount: allocAmount,
      referenceNumber: opt?.requiresReference ? generateRefNumber(methodCode) : '',
    }];
    tick().then(() => document.getElementById(`alloc-amount-${newId}`)?.focus());
  }
```
**Why (S-A):** focus lands straight on the amount field → one less click per method. `tick` is already imported (`import { tick } from 'svelte'`).
**Why (S-D):** prevents adding a useless 0-amount row after the sale is fully paid (e.g., CASH defaults to total in Edit 1). The cashier frees up `remaining` by editing an existing amount, which re-enables the chips.

### Edit 7 — Number-key method shortcuts (S-B)
File: `modules/pos/components/CheckoutModal.svelte` (`handleKeydown`, add right after the `Escape` block ~line 108)

```svelte
// NEW — insert after the Escape handler, before the F7 handler
    // Number-key shortcuts: add the Nth payment method (ignored while typing in a field)
    if (/^[1-9]$/.test(e.key) && dialogEl && !(document.activeElement instanceof HTMLInputElement)) {
      const idx = parseInt(e.key, 10) - 1;
      const opt = paymentOptions[idx];
      if (opt) {
        e.preventDefault();
        addAllocation(opt.id);
      }
      return;
    }
```
**Why:** power-user speed — `1` adds CASH, `2` adds CARD, etc., without leaving the keyboard. Guard avoids hijacking digits typed into amount/reference inputs.

### Edit 8 — Grey out not-yet-added chips when fully allocated (S-D)
File: `modules/pos/components/CheckoutModal.svelte` (payment grid button, ~lines 267–272)

```svelte
// OLD
                <button
                    class="py-2 rounded-xl border text-[11px] font-medium transition-all {isUsed ? 'border-primary bg-primary-subtle text-primary-light' : 'border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
                     onclick={() => addAllocation(opt.id)}
                   >
                     {paymentMethodLabel(opt.id, opt.label)}
                   </button>

// NEW
                <button
                    disabled={remainingBalance <= 0 && !isUsed}
                    class="py-2 rounded-xl border text-[11px] font-medium transition-all {isUsed ? 'border-primary bg-primary-subtle text-primary-light' : (remainingBalance <= 0 ? 'border-border text-text-muted/40 cursor-not-allowed' : 'border-border text-text-muted hover:border-border-strong hover:text-text-secondary')}"
                     onclick={() => addAllocation(opt.id)}
                   >
                     {paymentMethodLabel(opt.id, opt.label)}
                   </button>
```
**Why:** visually communicates "fully paid — reduce an amount to add another method" and pairs with the `addAllocation` guard in Edit 6.

### 2b.3 Design notes (S-C / S-E / S-F) — specify at implementation time

**S-C "Split Equally" helper (Should, high value).** Add a small `⟳ Split Equally` button near the payment grid (only when `allocations.length >= 2`, or always). On click, distribute `totalAmount` equally across the *current* methods, putting any rounding remainder on the first row:
```ts
function splitEqually() {
  if (allocations.length === 0) return;
  const base = Math.floor(totalAmount / allocations.length);
  const remainder = totalAmount - base * allocations.length;
  allocations = allocations.map((a, i) => ({
    ...a,
    amount: i === 0 ? base + remainder : base,
  }));
}
```
A "fixed amount" variant (assign X to the new method, auto-adjust an existing row) is a natural follow-up. Add i18n `splitEqually: 'Split Equally'` / `Bagi Rata`.

**S-E clearer "added" affordance (Nice).** In each allocation row header, append a subtle `✓ {labels.added}` (or "tap to edit") so the link between the top chip and its row is explicit. Add i18n `added: 'Added'` / `Ditambahkan`.

**S-F group/scroll grid (Conditional).** If `paymentOptions.length` is large, wrap the grid in a `max-h-40 overflow-y-auto` block or group chips by category (Cash / Card / E-Wallet) so the fixed-top region doesn't crowd the allocation list. Only needed if the method count grows.

### 2b.4 i18n additions (S-C / S-E)
File: `shared/i18n/en.ts` and `shared/i18n/id.ts` — add near the POS labels:
```ts
  splitEqually: 'Split Equally',   // id: 'Bagi Rata'
  added: 'Added',                  // id: 'Ditambahkan'
```

---

## 3. Phase 2 — Deferred (requires backend decision on C1)

- **C1 Cash change:** backend `change_due` column; relax `validatePayments` (`internal/sale/service.go:247`) from `totalPaid == totalAmount` to `totalPaid >= totalAmount`; derive `change = cashTendered − total`; return in `presentSale`; map into `ReceiptData.changeDue`.
- **K3 Receipt change line:** in `app/components/ReceiptPrintOverlay.svelte` render `changeDue` when `> 0`; remove hardcoded `0` in `PosPage.svelte:452` and `TransactionDrawer.svelte:103`.
- **U2 Confirm step:** lightweight summary modal consistent with the app's existing `ConfirmDeleteModal` pattern before `handleFinalize`.

## 4. Phase 3 — Later

- **S1 Multi same-method:** remove `ErrDuplicatePaymentMethod` for non-cash methods (`internal/sale/service.go:228`) to allow two cards / two e-wallets / two gift cards.
- **S3/S4 polish:** distinguish "Reset amounts" vs "Remove all"; label the dialog "Payment — Split" when `allocations.length > 1`.

---

## 5. Verification (after Phase 1 edits)

```bash
cd web && npm run build        # compile/type check
cd web && npx vitest run src/modules/pos   # any existing pos component tests
```
Manual: open POS → add items → Pay (F4) → confirm CASH row defaults to total, summary bar shows Paid=Remaining=0 ✓, Done enabled; reduce CASH amount → Remaining turns red ⚠ and Done disabled; add CARD → leave reference empty → Done disabled + red input; fill reference → Done enabled.
