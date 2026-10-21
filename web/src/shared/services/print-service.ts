// Receipt printing service.
//
// Single entry point used by the POS and sales screens. It dispatches a
// PrintDocument payload (a sale receipt or a return receipt, told apart by
// document_type) to the appropriate sink:
//   - "preview" mode: show the on-screen 58mm overlay and trigger the browser
//     print dialog (window.print) — the explicit/manual preview path.
//   - "silent"  mode: POST the payload to the local print agent. No overlay, no
//     dialog — the agent prints to a thermal / virtual printer (or, in dev, a
//     file).
//
// Silent mode NEVER falls back to the browser print dialog. If the agent is
// unreachable the call resolves with { ok: false } so the caller can surface a
// Retry/Dismiss notification. This matches the design doc: printing is a side
// effect of a completed sale, not part of transaction success.

import {
  printReceipt as printReceiptStore,
  type PrintDocument,
  type ReceiptData,
  type ReturnReceiptData,
} from "$shared/stores/printReceipt.svelte";
import { printConfig } from "$shared/stores/printConfig.svelte";
import { settingsStore } from "$shared/stores/settings.svelte";
import { toast } from "$shared/stores/toast.svelte";
import { labels } from "$shared/i18n";

export interface ReceiptBranding {
  storeName: string;
  storeAddress?: string;
  storePhone?: string;
  receiptHeader?: string;
  receiptFooter?: string;
}

export interface PrintAgentPayload {
  invoice: string;
  /** Discriminator the agent dispatches its renderer on. Absent = "sale":
   * an old frontend payload without it still renders as a sale receipt. */
  type: "sale" | "return";
  /** Sale payload (type "sale"). */
  data?: ReceiptData;
  /** Return payload (type "return") — a separate key because the agent decodes
   * JSON into concrete Go structs and cannot map one `data` key onto two
   * shapes: unknown fields would be silently dropped, so an agent that did not
   * know the shape would print a wrong receipt rather than refuse. */
  return_data?: ReturnReceiptData;
  branding: ReceiptBranding;
}

function brandingFromSettings(): ReceiptBranding {
  return {
    storeName: settingsStore.storeName,
    storeAddress: settingsStore.storeAddress || undefined,
    storePhone: settingsStore.storePhone || undefined,
    receiptHeader: settingsStore.receiptHeader || undefined,
    receiptFooter: settingsStore.receiptFooter || undefined,
  };
}

function buildAgentPayload(data: PrintDocument): PrintAgentPayload {
  const branding = brandingFromSettings();
  if (data.document_type === "return") {
    return {
      invoice: data.invoice_number,
      type: "return",
      return_data: data,
      branding,
    };
  }
  return {
    invoice: data.invoice_number,
    type: "sale",
    data,
    branding,
  };
}

function agentBase(): string {
  return printConfig.agentUrl.replace(/\/+$/, "");
}

async function sendToAgent(payload: PrintAgentPayload): Promise<void> {
  const res = await fetch(`${agentBase()}/print`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    throw new Error(`print agent responded ${res.status}`);
  }
}

function previewPrint(data: PrintDocument) {
  printReceiptStore.set(data);
  setTimeout(() => {
    window.print();
    setTimeout(() => printReceiptStore.set(null), 1000);
  }, 300);
}

export interface PrintResult {
  ok: boolean;
  mode: "preview" | "silent";
  /** i18n key describing the failure, when ok is false. */
  error?: string;
}

/**
 * Print a receipt.
 *
 * - preview mode: opens the on-screen receipt overlay and the browser print
 *   dialog.
 * - silent mode: submits the job to the local print agent and resolves once the
 *   job is accepted. If the agent cannot be reached it resolves with
 *   `{ ok: false }` and does NOT fall back to the browser dialog.
 *
 * Callers should check `result.ok` and, on failure, surface a non-blocking
 * Retry/Dismiss notification. The manual "Print Receipt" action is the retry
 * path; the notification itself is the Dismiss path.
 */
export async function printReceipt(data: PrintDocument): Promise<PrintResult> {
  if (printConfig.mode !== "silent") {
    previewPrint(data);
    return { ok: true, mode: "preview" };
  }
  try {
    await sendToAgent(buildAgentPayload(data));
    return { ok: true, mode: "silent" };
  } catch (err) {
    console.warn("[print] silent print failed", err);
    return { ok: false, mode: "silent", error: "printAgentUnavailable" };
  }
}

/**
 * Print a receipt and surface a non-blocking Retry/Dismiss notification when the
 * silent print agent is unreachable. This centralizes the toast-on-failure
 * behaviour shared by the POS screen and the transaction drawer so callers do
 * not each re-implement the same `!res.ok` check.
 */
export async function printReceiptWithToast(
  data: PrintDocument,
): Promise<PrintResult> {
  const res = await printReceipt(data);
  if (!res.ok) {
    toast.error(labels.printAgentUnavailable);
  }
  return res;
}

/**
 * Pulse the cash drawer wired to this till's printer.
 *
 * The drawer is not a second device on the PC: Indonesian cash drawers are
 * RJ11 and printer-driven, so they hang off the printer's drawer port and open
 * on an ESC/POS pulse. That makes this a print job with no receipt, which is
 * why it goes to the same endpoint and inherits the same origin allowlist and
 * job-id idempotency rather than opening a second route.
 *
 * Unlike a receipt it ignores `printConfig.mode`. Preview mode is a choice
 * about how a receipt is presented; it is not a claim that this till owns no
 * drawer, and the drawer is the one thing that cannot be shown on screen
 * instead.
 *
 * Failure resolves `false` and stays quiet. When no agent is reachable the
 * receipt that follows already raises its own notification, and a second toast
 * on every cash sale would be noise. Callers that need to report the outcome —
 * the manual test button during setup — check the returned value.
 */
export async function openCashDrawer(): Promise<boolean> {
  try {
    const res = await fetch(`${agentBase()}/print`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ type: "drawer" }),
    });
    return res.ok;
  } catch (err) {
    console.warn("[print] cash drawer pulse failed", err);
    return false;
  }
}
