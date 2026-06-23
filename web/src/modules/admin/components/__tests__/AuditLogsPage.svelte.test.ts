import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'AuditLogsPage.svelte'), 'utf-8');
}

describe('AuditLogsPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports apiClient for HTTP calls', () => {
    expect(src).toContain("import apiClient from '$shared/api/http-client'");
  });

  it('imports auth store', () => {
    expect(src).toContain("import { useAuthStore } from '$modules/auth'");
  });

  it('imports Jakarta time utilities (remaining)', () => {
    expect(src).toContain("import { getTodayInJakarta, getDateNDaysAgoInJakarta, JAKARTA_OFFSET_MS } from '$shared/utils/jakartaTime'");
  });

  it('uses $state for items, pagination, filters, and request tracking', () => {
    expect(src).toContain('let items = $state([])');
    expect(src).toContain('let total = $state(0)');
    expect(src).toContain('let searchQuery = $state');
    expect(src).toContain('let currentRequestId = $state(0)');
    expect(src).toContain('let abortController = $state');
  });

  it('has RBAC — only superadmin can view', () => {
    expect(src).toContain("let canView = $derived(userRole === 'superadmin')");
  });

  it('has drawer open/close functions', () => {
    expect(src).toContain('function openDrawer(log)');
    expect(src).toContain('function closeDrawer()');
  });

  it('has fetchLogs function', () => {
    expect(src).toContain('async function fetchLogs()');
  });

  it('has handlePageChange function', () => {
    expect(src).toContain('function handlePageChange(newOffset, newLimit)');
  });

  it('imports the three extracted child components', () => {
    expect(src).toContain("import AuditLogsFilterToolbar from './AuditLogsFilterToolbar.svelte'");
    expect(src).toContain("import AuditLogsTable from './AuditLogsTable.svelte'");
    expect(src).toContain("import AuditLogDetailsDrawer from './AuditLogDetailsDrawer.svelte'");
  });

  it('renders AuditLogsFilterToolbar', () => {
    expect(src).toContain('<AuditLogsFilterToolbar');
  });

  it('renders AuditLogsTable', () => {
    expect(src).toContain('<AuditLogsTable');
  });

  it('renders AuditLogDetailsDrawer', () => {
    expect(src).toContain('<AuditLogDetailsDrawer');
  });

  it('does NOT contain extracted filter constants', () => {
    expect(src).not.toContain('const actionsMap');
  });

  it('does NOT contain extracted table template', () => {
    expect(src).not.toContain('class="empty-state-icon"');
  });

  it('does NOT contain extracted drawer template', () => {
    expect(src).not.toContain('getDiffDescription');
  });

  it('does NOT contain extracted export functions', () => {
    expect(src).not.toContain('function exportToCsv');
    expect(src).not.toContain('function exportToExcel');
    expect(src).not.toContain('function buildExportUrl');
  });
});
