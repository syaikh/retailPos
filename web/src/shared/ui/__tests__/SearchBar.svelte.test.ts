import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'SearchBar.svelte'), 'utf-8');
}

describe('SearchBar.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Input from shared/ui', () => {
    expect(src).toContain("import { Input } from '$shared/ui'");
  });

  it('uses $bindable for value prop', () => {
    expect(src).toContain('value = $bindable');
  });

  it('has loading state handling', () => {
    expect(src).toContain('Loader2');
  });

  it('has clear button functionality', () => {
    expect(src).toContain('handleClear');
  });

  it('has Escape key handling', () => {
    expect(src).toContain('Escape');
  });
});