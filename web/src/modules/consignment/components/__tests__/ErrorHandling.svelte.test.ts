import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(component: string): string {
  return readFileSync(path.join(path.dirname(__filename), '..', component), 'utf-8');
}

describe('Consignment components error-handling pattern', () => {
  it('ArrangementsPage uses nested error?.response?.data?.error pattern', () => {
    const src = getSource('ArrangementsPage.svelte');
    expect(src).toContain("e?.response?.data?.error");
  });

  it('PendingReturnPage uses nested error?.response?.data?.error pattern', () => {
    const src = getSource('PendingReturnPage.svelte');
    expect(src).toContain("e?.response?.data?.error");
  });

  it('ReceiptEntry uses nested error?.response?.data?.error pattern', () => {
    const src = getSource('ReceiptEntry.svelte');
    expect(src).toContain("e?.response?.data?.error");
  });

  it('ReturnPage uses nested error?.response?.data?.error pattern', () => {
    const src = getSource('ReturnPage.svelte');
    expect(src).toContain("e?.response?.data?.error");
  });

  it('TermsEditor uses nested error?.response?.data?.error pattern', () => {
    const src = getSource('TermsEditor.svelte');
    expect(src).toContain("e?.response?.data?.error");
  });
});
