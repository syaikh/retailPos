import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'PurchaseOrderDetail.svelte'), 'utf-8');
}

describe('PurchaseOrderDetail.svelte source-structure guards', () => {
  const src = getSource();

  it('imports getReceipts for GR lookup', () => {
    expect(src).toContain("getReceipts } from '../services/po-service'");
  });

  it('imports GoodsReceipt type', () => {
    expect(src).toContain('GoodsReceipt');
  });

  it('has action buttons props (onedit, onconfirm, oncancel, onreceive)', () => {
    expect(src).toContain('onedit?:');
    expect(src).toContain('onconfirm?:');
    expect(src).toContain('oncancel?:');
    expect(src).toContain('onreceive?:');
  });

  it('has permission props (canEdit, canConfirm, canCancel, canReceive)', () => {
    expect(src).toContain('canEdit = false');
    expect(src).toContain('canConfirm = false');
    expect(src).toContain('canCancel = false');
    expect(src).toContain('canReceive = false');
  });

  it('renders edit button for draft POs', () => {
    expect(src).toContain("canEdit && po.status === 'draft'");
    expect(src).toContain('Pencil');
    expect(src).toContain('Edit');
  });

  it('renders confirm button for draft POs', () => {
    expect(src).toContain("canConfirm && po.status === 'draft'");
    expect(src).toContain('Check');
    expect(src).toContain('Confirm');
  });

  it('renders receive button for confirmed or partial-received POs', () => {
    expect(src).toContain("canReceive && (po.status === 'confirmed' || po.status === 'partial_received')");
    expect(src).toContain('Package');
    expect(src).toContain('Receive');
  });

  it('renders cancel button for draft or confirmed POs', () => {
    expect(src).toContain("canCancel && (po.status === 'draft' || po.status === 'confirmed')");
    expect(src).toContain('XCircle');
    expect(src).toContain('Cancel PO');
  });

  it('displays DO numbers from receipts', () => {
    expect(src).toContain('Truck');
    expect(src).toContain('DO#');
    expect(src).toContain('delivery_order_number');
  });

  it('has copy PO number button with clipboard API', () => {
    expect(src).toContain('navigator.clipboard.writeText(po!.po_number!)');
    expect(src).toContain('aria-label="Salin nomor PO"');
  });

  it('has reloadKey prop tracking', () => {
    expect(src).toContain('reloadKey');
  });
});
