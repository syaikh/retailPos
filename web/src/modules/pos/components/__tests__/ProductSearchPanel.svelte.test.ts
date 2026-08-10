import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'ProductSearchPanel.svelte'), 'utf-8');
}

describe('ProductSearchPanel.svelte source-structure guards', () => {
  const src = getSource();

  it('imports SearchBar from shared/ui', () => {
    expect(src).toContain("import { SearchBar } from '$shared/ui'");
  });

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels } from '$shared/i18n'");
  });

  it('uses $bindable for searchQuery', () => {
    expect(src).toContain('searchQuery = $bindable(');
  });

  it('renders F2 kbd shortcut', () => {
    expect(src).toContain('F2</kbd>');
  });

  it('renders arrow and Enter navigation hints', () => {
    expect(src).toContain('↑↓');
    expect(src).toContain('Enter</kbd>');
    expect(src).toContain('posSelectProductHint');
    expect(src).toContain('posAddToCartHint');
  });

  it('renders SearchBar with pos-search-input id', () => {
    expect(src).toContain('id="pos-search-input"');
  });
});
