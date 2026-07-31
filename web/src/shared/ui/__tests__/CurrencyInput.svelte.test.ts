import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'CurrencyInput.svelte'), 'utf-8');
}

describe('CurrencyInput.svelte source-structure guards', () => {
  const src = getSource();

  it('selects all text on focus for fast replacement', () => {
    expect(src).toContain('onfocus={(e) => (e.target as HTMLInputElement).select()}');
  });

  it('uses inputmode numeric for mobile keyboard', () => {
    expect(src).toContain('inputmode="numeric"');
  });

  it('formats value with id-ID locale on input', () => {
    expect(src).toContain("toLocaleString('id-ID')");
  });

  it('displays Rp prefix', () => {
    expect(src).toContain('Rp');
  });
});
