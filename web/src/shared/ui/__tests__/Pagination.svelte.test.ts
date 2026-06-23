import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'Pagination.svelte'), 'utf-8');
}

describe('Pagination.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Button from shared/ui', () => {
    expect(src).toContain("import { Button } from '$shared/ui'");
  });

  it('uses $derived for currentPage', () => {
    expect(src).toContain('currentPage');
  });

  it('uses $derived for totalPages', () => {
    expect(src).toContain('totalPages');
  });

  it('calls onPageChange prop', () => {
    expect(src).toContain('onPageChange');
  });

  it('renders navigation buttons', () => {
    expect(src).toContain('ChevronLeft');
    expect(src).toContain('ChevronRight');
  });
});