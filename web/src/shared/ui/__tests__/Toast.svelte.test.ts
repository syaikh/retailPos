import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'Toast.svelte'), 'utf-8');
}

describe('Toast.svelte source-structure guards', () => {
  const src = getSource();

  it('imports toast store from shared/stores', () => {
    expect(src).toContain("import { toast } from '$shared/stores/toast.svelte'");
  });

  it('imports lucide icons for variants', () => {
    expect(src).toContain("import { CheckCircle, XCircle, AlertTriangle, Info, X } from 'lucide-svelte'");
  });

  it('defines icons map for success, error, warning, info', () => {
    expect(src).toContain('success: CheckCircle');
    expect(src).toContain('error:   XCircle');
    expect(src).toContain('warning: AlertTriangle');
    expect(src).toContain('info:    Info');
  });

  it('defines styles map for each variant', () => {
    expect(src).toContain("'border-success/30 bg-success-subtle text-success-light'");
    expect(src).toContain("'border-danger/30 bg-danger-subtle text-danger-light'");
    expect(src).toContain("'border-warning/30 bg-warning-subtle text-warning-light'");
    expect(src).toContain("'border-info/30 bg-info-subtle text-info-light'");
  });

  it('uses aria-live polite for accessibility', () => {
    expect(src).toContain('aria-live="polite"');
  });

  it('has role="alert" on each toast', () => {
    expect(src).toContain('role="alert"');
  });

  it('renders dismiss button with aria-label', () => {
    expect(src).toContain('aria-label="Dismiss"');
  });

  it('calls toast.remove on dismiss', () => {
    expect(src).toContain('toast.remove(t.id)');
  });

  it('uses #each over toast store', () => {
    expect(src).toContain('{#each $toast as t (t.id)}');
  });
});
