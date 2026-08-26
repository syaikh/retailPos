// Receipt printing service.
//
// Single entry point used by the POS and sales screens. It dispatches a
// ReceiptData payload to the appropriate sink:
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

import { printReceipt as printReceiptStore, type ReceiptData } from '$shared/stores/printReceipt.svelte';
import { printConfig } from '$shared/stores/printConfig.svelte';
import { settingsStore } from '$shared/stores/settings.svelte';
import { toast } from '$shared/stores/toast.svelte';
import { labels } from '$shared/i18n';

export interface ReceiptBranding {
  storeName: string;
  storeAddress?: string;
  storePhone?: string;
  receiptHeader?: string;
  receiptFooter?: string;
}

export interface PrintAgentPayload {
  invoice: string;
  data: ReceiptData;
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

function buildAgentPayload(data: ReceiptData): PrintAgentPayload {
  return {
    invoice: data.invoice_number,
    data,
    branding: brandingFromSettings(),
  };
}

async function sendToAgent(payload: PrintAgentPayload): Promise<void> {
  const base = printConfig.agentUrl.replace(/\/+$/, '');
  const res = await fetch(`${base}/print`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    throw new Error(`print agent responded ${res.status}`);
  }
}

function previewPrint(data: ReceiptData) {
  printReceiptStore.set(data);
  setTimeout(() => {
    window.print();
    setTimeout(() => printReceiptStore.set(null), 1000);
  }, 300);
}

export interface PrintResult {
  ok: boolean;
  mode: 'preview' | 'silent';
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
export async function printReceipt(data: ReceiptData): Promise<PrintResult> {
  if (printConfig.mode !== 'silent') {
    previewPrint(data);
    return { ok: true, mode: 'preview' };
  }
  try {
    await sendToAgent(buildAgentPayload(data));
    return { ok: true, mode: 'silent' };
  } catch (err) {
    console.warn('[print] silent print failed', err);
    return { ok: false, mode: 'silent', error: 'printAgentUnavailable' };
  }
}

/**
 * Print a receipt and surface a non-blocking Retry/Dismiss notification when the
 * silent print agent is unreachable. This centralizes the toast-on-failure
 * behaviour shared by the POS screen and the transaction drawer so callers do
 * not each re-implement the same `!res.ok` check.
 */
export async function printReceiptWithToast(data: ReceiptData): Promise<PrintResult> {
  const res = await printReceipt(data);
  if (!res.ok) {
    toast.error(labels.printAgentUnavailable);
  }
  return res;
}
