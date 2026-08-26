// Agent unit tests — run with `npm test` (node --test). No hardware required.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, readFileSync, existsSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { renderHtml, buildEscPos, writeReceipt, formatMoney } from './printer.js';

const data = {
  invoice_number: 'INV-1',
  created_at: '2026-08-26 10:00',
  items: [{ name: 'Kopi', quantity: 2, unit_price: 5000 }],
  total_amount: 10000,
  subtotal_dpp: 10000,
  tax: 0,
  paymentMethod: 'CASH',
  payments: [{ method: 'CASH', amount: 10000 }],
  cashReceived: 10000,
  changeDue: 0,
  customer_name: 'Budi',
};
const branding = { storeName: 'Toko Test', storeAddress: 'Jl. Test', storePhone: '021', receiptFooter: 'Terima kasih' };

test('formatMoney uses id-ID grouping', () => {
  assert.equal(formatMoney(10000), '10.000');
});

test('renderHtml produces a 58mm document with store + items', () => {
  const html = renderHtml(data, branding);
  assert.match(html, /<title>Struk INV-1/);
  assert.match(html, /Toko Test/);
  assert.match(html, /Kopi x2/);
  assert.match(html, /10\.000/);
  assert.match(html, /size: 58mm/);
});

test('buildEscPos emits init + cut control bytes', () => {
  const buf = buildEscPos(data, branding);
  assert.ok(Buffer.isBuffer(buf));
  // ESC @  init
  assert.deepEqual([buf[0], buf[1], buf[2]], [0x1b, 0x40, 0x1b]);
  // GS V 0 cut near the end
  const cut = buf.slice(buf.length - 3);
  assert.deepEqual([...cut], [0x1d, 0x56, 0x00]);
});

test('writeReceipt file target writes an HTML file', () => {
  const dir = mkdtempSync(join(tmpdir(), 'pa-'));
  try {
    const r = writeReceipt({ target: 'file', dir, invoice: 'INV-1', data, branding });
    assert.equal(r.target, 'file');
    assert.ok(existsSync(r.path));
    assert.match(readFileSync(r.path, 'utf8'), /Toko Test/);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test('writeReceipt thermal target writes an ESC/POS bin file', () => {
  const dir = mkdtempSync(join(tmpdir(), 'pa-'));
  try {
    const r = writeReceipt({ target: 'thermal', dir, invoice: 'INV-1', data, branding });
    assert.equal(r.target, 'thermal');
    assert.ok(existsSync(r.path));
    assert.ok(Buffer.isBuffer(readFileSync(r.path)));
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
