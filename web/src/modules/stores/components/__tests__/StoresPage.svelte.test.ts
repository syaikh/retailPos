import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'StoresPage.svelte'), 'utf-8');
}

describe('StoresPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Jakarta time utility', () => {
    expect(src).toContain("import { formatDateInJakarta } from '$shared/utils/jakartaTime'");
  });

  it('imports store service functions', () => {
    expect(src).toContain("import { getStores, createStore, updateStore, deleteStore } from '../services/stores-service'");
  });

  it('imports shared UI components', () => {
    expect(src).toContain("import { Button, Input, Modal, Skeleton, BulkActionDropdown, ImportWizard, SearchBar, ToggleSwitch, ConfirmDeleteModal, Pagination, SortableHeader } from '$shared/ui'");
  });

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels, t } from '$shared/i18n'");
  });

  it('uses $state for stores, loading, pagination', () => {
    expect(src).toContain('let loading = $state(true)');
    expect(src).toContain('let stores = $state([])');
    expect(src).toContain('let total = $state(0)');
    expect(src).toContain('let searchQuery = $state');
    expect(src).toContain('let statusFilter = $state');
  });

  it('has RBAC derived from the shared composable', () => {
    expect(src).toContain('const rbac = useRBAC()');
    expect(src).toContain('let canCreate = $derived(rbac.can(Permissions.store.create))');
    expect(src).toContain('let canEdit = $derived(rbac.can(Permissions.store.update))');
    expect(src).toContain('let canDelete = $derived(rbac.can(Permissions.store.delete))');
  });

  it('has sort state and handleSort function', () => {
    expect(src).toContain("const { sortState, handleSort } = useSortable('name', 'asc')");
    expect(src).toContain('sortState.sortBy');
    expect(src).toContain('sortState.sortDir');
  });

  it('has fetchStores, openAdd, openEdit, saveStore functions', () => {
    expect(src).toContain('async function fetchStores');
    expect(src).toContain('function openAdd');
    expect(src).toContain('function openEdit');
    expect(src).toContain('async function saveStore');
  });

  it('has status filter chips', () => {
    expect(src).toContain('labels.all');
    expect(src).toContain('labels.active');
    expect(src).toContain('labels.inactive');
    expect(src).toContain('function setStatusFilter');
  });

  it('renders BulkActionDropdown with module stores', () => {
    expect(src).toContain('module="stores"');
    expect(src).toContain('<BulkActionDropdown');
  });

  it('renders ImportWizard with module stores', () => {
    expect(src).toContain('<ImportWizard');
    expect(src).toContain('module="stores"');
    expect(src).toContain('displayName={labels.stores}');
  });

  it('renders Pagination component', () => {
    expect(src).toContain('<Pagination');
  });
});
