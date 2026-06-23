import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'CustomersPage.svelte'), 'utf-8');
}

describe('CustomersPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports apiClient for HTTP calls', () => {
    expect(src).toContain("import apiClient from '$shared/api/http-client'");
  });

  it('imports auth store', () => {
    expect(src).toContain("import { useAuthStore } from '$modules/auth'");
  });

  it('imports shared UI components', () => {
    expect(src).toContain("import { Badge, Button, SearchBar, Skeleton } from '$shared/ui'");
    expect(src).toContain("import { Input, Modal, Pagination } from '$shared/ui'");
  });

  it('uses $state for customers, loading, pagination', () => {
    expect(src).toContain('let customers = $state');
    expect(src).toContain('let loading = $state');
    expect(src).toContain('let total = $state(0)');
    expect(src).toContain('let searchQuery = $state');
  });

  it('has permission-based RBAC (canCreate, canUpdate, canDelete, canRead)', () => {
    expect(src).toContain("const canCreate = $derived(userPermissions.includes('customer:create'))");
    expect(src).toContain("const canUpdate = $derived(userPermissions.includes('customer:update'))");
    expect(src).toContain("const canDelete = $derived(userPermissions.includes('customer:delete'))");
    expect(src).toContain("const canRead = $derived(userPermissions.includes('customer:read'))");
  });

  it('has bulk selection and operations (toggleSelectAll, handleBulkStatusUpdate, handleBulkDelete)', () => {
    expect(src).toContain('function toggleSelectAll');
    expect(src).toContain('async function handleBulkStatusUpdate');
    expect(src).toContain('async function handleBulkDelete');
  });

  it('has load, createCustomer, saveEdit functions', () => {
    expect(src).toContain('async function load');
    expect(src).toContain('async function createCustomer');
    expect(src).toContain('async function saveEdit');
  });

  it('has inline editing (startEdit, cancelEdit)', () => {
    expect(src).toContain('function startEdit');
    expect(src).toContain('function cancelEdit');
  });

  it('renders Pagination component', () => {
    expect(src).toContain('<Pagination');
  });
});
