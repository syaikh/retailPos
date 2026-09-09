import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'SettlementPage.svelte'), 'utf-8');
}

describe('SettlementPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports FormattedNumberInput instead of NumberInput for payout amount', () => {
    expect(src).toContain("FormattedNumberInput");
    expect(src).not.toMatch(/import.*\bNumberInput\b.*from.*'\$shared\/ui'/);
  });

  it('extracts nested error message from API response (e?.response?.data?.error)', () => {
    expect(src).toContain("e?.response?.data?.error");
  });

  it('does not use the old flat error pattern for create settlement', () => {
    const lines = src.split('\n');
    const createSettlementLine = lines.findIndex(l => l.includes('consignmentCreateSettlementError'));
    if (createSettlementLine >= 0) {
      const context = lines.slice(Math.max(0, createSettlementLine - 5), createSettlementLine + 1).join('\n');
      expect(context).toContain('response?.data?.error');
    }
  });

  it('does not use the old flat error pattern for record payout', () => {
    const lines = src.split('\n');
    const payoutLine = lines.findIndex(l => l.includes('consignmentRecordPayoutError'));
    if (payoutLine >= 0) {
      const context = lines.slice(Math.max(0, payoutLine - 5), payoutLine + 1).join('\n');
      expect(context).toContain('response?.data?.error');
    }
  });

  it('uses FormattedNumberInput for payout amount field', () => {
    expect(src).toContain('<FormattedNumberInput');
    expect(src).toContain('bind:value={payoutForm.amount}');
  });
});
