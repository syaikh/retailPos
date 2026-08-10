import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'AuditLogsTable.svelte'), 'utf-8');
}

describe('AuditLogsTable.svelte source-structure guards', () => {
  const src = getSource();

  it('imports shared UI components', () => {
    expect(src).toContain("import { Button, Pagination, Skeleton, ActionBadge } from '$shared/ui'");
  });

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels } from '$shared/i18n'");
  });

  it('uses $props', () => {
    expect(src).toContain('= $props()');
  });

  it('accepts items and pagination props', () => {
    expect(src).toContain('items =');
    expect(src).toContain('total =');
    expect(src).toContain('limit =');
    expect(src).toContain('offset =');
  });

  it('has callback props', () => {
    expect(src).toContain('onpagechange');
    expect(src).toContain('onrowclick');
  });

  it('has loading skeleton section', () => {
    expect(src).toContain('<Skeleton');
  });

  it('has empty state section', () => {
    expect(src).toContain('labels.noAuditLogsFound');
  });

  it('renders table with all columns', () => {
    expect(src).toContain('labels.timestamp');
    expect(src).toContain('labels.actor');
    expect(src).toContain('labels.resource');
    expect(src).toContain('labels.action');
    expect(src).toContain('labels.description');
    expect(src).toContain('labels.ipAddress');
  });

  it('has Pagination component', () => {
    expect(src).toContain('<Pagination');
  });

  it('has formatTimestamp helper', () => {
    expect(src).toContain('function formatTimestamp');
  });

  it('imports Jakarta time utils', () => {
    expect(src).toContain("import { formatDateInJakarta, formatTimeInJakarta } from '$shared/utils/jakartaTime'");
  });
});
