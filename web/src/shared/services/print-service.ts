// Receipt printing service.
//
// Single entry point used by the POS and sales screens. It dispatches a
// ReceiptData payload to the appropriate sink:
//   - "preview" mode: show the on-screen 58mm overlay and trigger the browser
//     print dialog (window.print) — the current behaviour.
//   - "silent"  mode: POST the payload to the local print agent. No overlay, no
//     dialog — the agent prints to a thermal / virtual printer (or, in dev, a
//     file). If the agent is unreachable the call falls back to preview mode so
//     the cashier always gets a receipt.

import { printReceipt as printReceiptStore, type ReceiptData } from '$shared/stores/printReceipt.svelte';
import { printConfig } from '$shared/stores/printConfig.svelte';
import { settingsStore } from '$shared/stores/settings.svelte';

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

/**
 * Print a receipt. Resolves once the receipt has been dispatched (to the agent
 * in silent mode, or after the preview dialog is requested). Failures in silent
 * mode fall back to the preview dialog.
 */
export async function printReceipt(data: ReceiptData): Promise<void> {
  if (printConfig.mode === 'silent') {
    try {
      await sendToAgent(buildAgentPayload(data));
      return;
    } catch (err) {
      console.warn('[print] silent print failed, falling back to preview', err);
    }
  }
  previewPrint(data);
}
