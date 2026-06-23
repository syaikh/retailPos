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

  it('imports auth store and getAuthToken', () => {
    expect(src).toContain("import { useAuthStore, getAuthToken } from '$modules/auth'");
  });

  it('imports Jakarta time utilities', () => {
    expect(src).toContain("import { getTodayInJakarta, getDateNDaysAgoInJakarta, JAKARTA_OFFSET_MS, formatDateInJakarta, formatTimeInJakarta, formatDateTimeInJakarta, formatJakartaDateStr } from '$shared/utils/jakartaTime'");
  });

  it('imports ActionBadge shared UI component', () => {
    expect(src).toContain("import { ActionBadge, Button, Input, Pagination, SearchBar, Skeleton } from '$shared/ui'");
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

  it('has fetchLogs, exportToCsv, exportToExcel functions', () => {
    expect(src).toContain('async function fetchLogs()');
    expect(src).toContain('function exportToCsv');
    expect(src).toContain('function exportToExcel');
  });

  it('renders Pagination component', () => {
    expect(src).toContain('<Pagination');
  });
});
