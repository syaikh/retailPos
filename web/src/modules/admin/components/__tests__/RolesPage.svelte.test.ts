import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'RolesPage.svelte'), 'utf-8');
}

describe('RolesPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports apiFetch for HTTP calls', () => {
    expect(src).toContain("import { apiFetch } from '$shared/api/http-client'");
  });

  it('imports auth store', () => {
    expect(src).toContain("import { useAuthStore } from '$modules/auth'");
  });

  it('imports shared UI components', () => {
    expect(src).toContain("import { Badge, Button, Dropdown, Input, Modal, Pagination, SearchBar, Skeleton, ConfirmDeleteModal, SortableHeader } from '$shared/ui'");
  });

  it('imports RoleDetailDrawer component', () => {
    expect(src).toContain("import RoleDetailDrawer from './RoleDetailDrawer.svelte'");
  });

  it('uses $state for roles, permissions, modals, pagination', () => {
    expect(src).toContain('let roles = $state([])');
    expect(src).toContain('let permissions = $state([])');
    expect(src).toContain('let showModal = $state(false)');
    expect(src).toContain('let saving = $state(false)');
    expect(src).toContain('let permissionSearch = $state');
  });

  it('has pagination state (pageLimit, pageOffset)', () => {
    expect(src).toContain('let pageLimit = $state(20)');
    expect(src).toContain('let pageOffset = $state(0)');
  });

  it('has modalStep for multi-step modal', () => {
    expect(src).toContain('let modalStep = $state(1)');
  });

  it('has expanded detail row', () => {
    expect(src).toContain('let expandedRoleId = $state(null)');
  });

  it('has action dropdown via Dropdown component', () => {
    expect(src).toContain('<Dropdown');
  });

  it('has fetchData, openAdd, openEdit, saveRole functions', () => {
    expect(src).toContain('async function fetchData()');
    expect(src).toContain('function openAdd');
    expect(src).toContain('function openEdit');
    expect(src).toContain('async function saveRole');
  });

  it('has role detail drawer state', () => {
    expect(src).toContain('let showRoleDrawer = $state');
  });

  it('renders Pagination component', () => {
    expect(src).toContain('<Pagination');
  });
});
