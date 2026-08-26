// Receipt rendering + routing for the local print agent.
//
// Pure-ish functions with no external dependencies so the agent runs on a bare
// Node install. The HTTP server (index.js) calls into these.
//
// Supported targets (selected via the PRINT_TARGET env var):
//   - "file"    : write a self-contained 58mm HTML receipt to disk. Open it in a
//                 browser to eyeball layout. Primary no-hardware test path.
//   - "pdf"     : same as "file" (into a pdf/ subdir) and, if PRINT_PDF_PRINTER
//                 is set, additionally spools the HTML to that CUPS printer.
//   - "thermal" : build ESC/POS byte stream. Written to a .bin file by default;
//                 if PRINT_SERIAL_PORT is set (e.g. /dev/ttyUSB0) it is written
//                 straight to the serial device instead.

import { mkdirSync, writeFileSync, createWriteStream } from 'node:fs';
import { join } from 'node:path';
import { spawn } from 'node:child_process';

const CURRENCY = 'Rp';

export function formatMoney(n) {
  if (n == null || isNaN(n)) return '0';
  return Number(n).toLocaleString('id-ID');
}

function esc(s) {
  return String(s ?? '').replace(/[&<>]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' }[c]));
}

// ────────────────────────────────────────────────────────────────────────
// HTML (58mm) receipt — mirrors the Svelte ReceiptPrintOverlay layout.
// ────────────────────────────────────────────────────────────────────────
export function renderHtml(data, branding = {}) {
  const items = (data.items || [])
    .map(
      (it) => `
      <div class="item">
        <div class="name">${esc(it.name)} x${esc(it.quantity)}</div>
        <div class="price">${esc(formatMoney(it.unit_price * it.quantity))}</div>
      </div>`
    )
    .join('');

  const payments = (data.payments && data.payments.length > 0)
    ? data.payments
        .map((p) => `<div class="row"><span>${esc(p.method)}</span><span>${esc(formatMoney(p.amount))}</span></div>`)
        .join('')
    : `<div class="row"><span>${esc(data.paymentMethod || '')}</span></div>`;

  const taxRows =
    data.subtotal_dpp != null && data.tax != null && data.tax > 0
      ? `<div class="row"><span>DPP</span><span>${esc(formatMoney(data.subtotal_dpp))}</span></div>
         <div class="row"><span>PPN</span><span>${esc(formatMoney(data.tax))}</span></div>`
      : '';

  const changeRow =
    data.changeDue > 0
      ? `<div class="row total"><span>Kembali</span><span>${esc(formatMoney(data.changeDue))}</span></div>`
      : '';

  const footerLines = (branding.receiptFooter || 'Terima kasih atas kunjungan Anda!\nBarang yang sudah dibeli tidak dapat dikembalikan.')
    .split('\n')
    .map((l) => `<p>${esc(l)}</p>`)
    .join('');

  return `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8" />
<title>Struk ${esc(data.invoice_number)}</title>
<style>
  @page { size: 58mm auto; margin: 0; }
  * { box-sizing: border-box; }
  body { width: 58mm; margin: 0; padding: 4mm; font-family: monospace; font-size: 11px; color: #000; }
  .center { text-align: center; }
  .name { font-weight: bold; }
  .divider { border-top: 1px dashed #000; margin: 4px 0; }
  .row { display: flex; justify-content: space-between; }
  .item { display: flex; justify-content: space-between; }
  .total { font-weight: bold; }
  .footer { text-align: center; margin-top: 6px; }
</style>
</head>
<body>
  <div class="center name">${esc(branding.storeName || 'RetailPOS')}</div>
  ${branding.storeAddress ? `<div class="center">${esc(branding.storeAddress)}</div>` : ''}
  ${branding.storePhone ? `<div class="center">Telp: ${esc(branding.storePhone)}</div>` : ''}
  ${branding.receiptHeader ? `<div class="center">${esc(branding.receiptHeader)}</div>` : ''}
  <div class="divider"></div>
  <div class="row"><span>Inv:</span><span>${esc(data.invoice_number)}</span></div>
  <div class="row"><span>Waktu:</span><span>${esc(data.created_at || '')}</span></div>
  ${data.customer_name ? `<div class="row"><span>Cust:</span><span>${esc(data.customer_name)}</span></div>` : ''}
  <div class="divider"></div>
  ${items}
  <div class="divider"></div>
  ${taxRows}
  <div class="row total"><span>TOTAL</span><span>${esc(formatMoney(data.total_amount))}</span></div>
  <div class="divider"></div>
  ${payments}
  ${changeRow}
  <div class="divider"></div>
  <div class="footer">${footerLines}</div>
</body>
</html>`;
}

// ────────────────────────────────────────────────────────────────────────
// ESC/POS byte stream for real thermal printers.
// ────────────────────────────────────────────────────────────────────────
const ESC = 0x1b;
const GS = 0x1d;

export function buildEscPos(data, branding = {}) {
  const chunks = [];
  const push = (buf) => chunks.push(buf);
  const text = (s) => push(Buffer.from(`${s ?? ''}\n`, 'utf8'));
  const textRaw = (s) => push(Buffer.from(`${s ?? ''}`, 'utf8'));

  push(Buffer.from([ESC, 0x40])); // init
  const align = (a) => push(Buffer.from([ESC, 0x61, a])); // 0 left, 1 center, 2 right
  const bold = (on) => push(Buffer.from([ESC, 0x45, on ? 1 : 0]));

  align(1);
  bold(true); text(branding.storeName || 'RetailPOS'); bold(false);
  if (branding.storeAddress) text(branding.storeAddress);
  if (branding.storePhone) text(`Telp: ${branding.storePhone}`);
  if (branding.receiptHeader) text(branding.receiptHeader);
  text('');

  align(0);
  text(`Inv: ${data.invoice_number}`);
  text(`Waktu: ${data.created_at || ''}`);
  if (data.customer_name) text(`Cust: ${data.customer_name}`);
  text('------------------------');

  for (const it of data.items || []) {
    text(`${it.name} x${it.quantity}`);
    text(`  ${formatMoney(it.unit_price * it.quantity)}`);
  }
  text('------------------------');

  if (data.subtotal_dpp != null && data.tax != null && data.tax > 0) {
    text(`DPP  ${formatMoney(data.subtotal_dpp)}`);
    text(`PPN  ${formatMoney(data.tax)}`);
  }
  bold(true); text(`TOTAL ${formatMoney(data.total_amount)}`); bold(false);
  text('------------------------');

  const payments = (data.payments && data.payments.length > 0)
    ? data.payments
    : [{ method: data.paymentMethod, amount: data.total_amount }];
  for (const p of payments) text(`${p.method}  ${formatMoney(p.amount)}`);
  if (data.changeDue > 0) text(`Kembali ${formatMoney(data.changeDue)}`);
  text('------------------------');

  align(1);
  for (const l of (branding.receiptFooter || 'Terima kasih atas kunjungan Anda!').split('\n')) text(l);
  textRaw('');
  push(Buffer.from([GS, 0x56, 0x00])); // cut

  return Buffer.concat(chunks);
}

// ────────────────────────────────────────────────────────────────────────
// Routing
// ────────────────────────────────────────────────────────────────────────
function safeInvoice(invoice) {
  return String(invoice || 'receipt').replace(/[^a-zA-Z0-9_-]/g, '_');
}

export function writeReceipt({ target = 'file', dir = '/tmp', invoice = 'receipt', data, branding, serialPort } = {}) {
  mkdirSync(dir, { recursive: true });
  const name = safeInvoice(invoice);

  if (target === 'thermal') {
    const buf = buildEscPos(data, branding);
    if (serialPort) {
      return new Promise((resolve, reject) => {
        const stream = createWriteStream(serialPort);
        stream.on('error', reject);
        stream.end(buf, () => resolve({ target, serialPort }));
      });
    }
    const path = join(dir, `escpos-${name}.bin`);
    writeFileSync(path, buf);
    return { target, path };
  }

  // file + pdf both render HTML; pdf additionally tries to spool via CUPS.
  const html = renderHtml(data, branding);
  const outDir = target === 'pdf' ? join(dir, 'pdf') : dir;
  mkdirSync(outDir, { recursive: true });
  const path = join(outDir, `receipt-${name}.html`);
  writeFileSync(path, html);

  let spooled = false;
  const pdfPrinter = process.env.PRINT_PDF_PRINTER;
  if (target === 'pdf' && pdfPrinter) {
    try {
      const tmp = join(outDir, `receipt-${name}.html`);
      const child = spawn('lp', ['-d', pdfPrinter, tmp]);
      spooled = true;
      child.on('error', () => { spooled = false; });
    } catch {
      spooled = false;
    }
  }

  return { target, path, spooled };
}
