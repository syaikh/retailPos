import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'Input.svelte'), 'utf-8');
}

describe('Input.svelte source-structure guards', () => {
  const src = getSource();

  it('imports cn utility', () => {
    expect(src).toContain("import { cn } from '$shared/utils/cn'");
  });

  it('has error prop', () => {
    expect(src).toContain('error = ');
  });

  it('binds aria-invalid to !!error', () => {
    expect(src).toContain('aria-invalid={!!error}');
  });

  it('binds aria-describedby when error is present', () => {
    expect(src).toContain('aria-describedby={error ? `${inputId}-error` : undefined}');
  });

  it('renders error message with role="alert"', () => {
    expect(src).toContain('role="alert"');
  });

  it('renders error message with id matching aria-describedby', () => {
    expect(src).toContain('id="{inputId}-error"');
  });
});
