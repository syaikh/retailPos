import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'AuditLogDetailsDrawer.svelte'), 'utf-8');
}

describe('AuditLogDetailsDrawer.svelte source-structure guards', () => {
  const src = getSource();

  it('imports lucide icons', () => {
    expect(src).toContain("import { Plus, Minus, ArrowRight, Clock, Globe, Monitor } from 'lucide-svelte'");
  });

  it('imports ActionBadge from shared/ui', () => {
    expect(src).toContain("import { ActionBadge, Drawer } from '$shared/ui'");
  });

  it('imports Jakarta time utilities', () => {
    expect(src).toContain("import { formatDateInJakarta, formatTimeInJakarta, formatDateTimeInJakarta, JAKARTA_OFFSET_MS } from '$shared/utils/jakartaTime'");
  });

  it('imports i18n labels and t', () => {
    expect(src).toContain("import { labels, t } from '$shared/i18n'");
  });

  it('uses $props and $bindable', () => {
    expect(src).toContain('= $props()');
    expect(src).toContain('drawerOpen = $bindable');
  });

  it('has diff computation functions', () => {
    expect(src).toContain('function getChanges');
    expect(src).toContain('function getDiffDescription');
    expect(src).toContain('function formatValue');
    expect(src).toContain('function getFieldLabel');
  });

  it('has action/resource label helpers', () => {
    expect(src).toContain('function getActionVerb');
    expect(src).toContain('function getResourceLabel');
  });

  it('has timestamp and date formatting', () => {
    expect(src).toContain('function formatTimestamp');
    expect(src).toContain('function formatDateHuman');
  });

  it('has fieldLabels map', () => {
    expect(src).toContain('const fieldLabels');
    expect(src).toContain('name: labels.name');
    expect(src).toContain('email: labels.email');
  });

  it('renders drawer with header and body', () => {
    expect(src).toContain('labels.auditLogs');
    expect(src).toContain('labels.whatChanged');
  });

  it('shows changes section', () => {
    expect(src).toContain('No specific data changes captured');
  });
});
