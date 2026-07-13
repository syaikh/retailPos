import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'RoleDetailDrawer.svelte'), 'utf-8');
}

describe('RoleDetailDrawer.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Badge, Button and Drawer from shared/ui', () => {
    expect(src).toContain("import { Badge, Button, Drawer } from '$shared/ui'");
  });

  it('imports lucide icons', () => {
    expect(src).toContain("import { Shield, Users, Copy, Pencil, Trash2 } from 'lucide-svelte'");
  });

  it('uses $props for component props', () => {
    expect(src).toContain('= $props()');
  });

  it('has selectedRole prop', () => {
    expect(src).toContain('selectedRole = null');
  });

  it('has permissions prop', () => {
    expect(src).toContain('permissions = []');
  });

  it('has canEdit and canDelete boolean props', () => {
    expect(src).toContain('canEdit = false');
    expect(src).toContain('canDelete = false');
  });

  it('has event callbacks (onclose, onedit, onduplicate, ondeleterequest)', () => {
    expect(src).toContain('onclose = () => {}');
    expect(src).toContain('onedit = () => {}');
    expect(src).toContain('onduplicate = () => {}');
    expect(src).toContain('ondeleterequest = () => {}');
  });

  it('has getRolePermissions helper function', () => {
    expect(src).toContain('function getRolePermissions');
  });

  it('has getGroupedPermissions helper function', () => {
    expect(src).toContain('function getGroupedPermissions');
  });

  it('has groupMeta constant', () => {
    expect(src).toContain('const groupMeta');
  });

  it('has permission list with grouped display', () => {
    expect(src).toContain('getRolePermissions(selectedRole)');
    expect(src).toContain('getGroupedPermissions(rolePerms)');
  });

  it('has edit/duplicate/delete action buttons', () => {
    expect(src).toContain('<Copy size={15}');
    expect(src).toContain('<Pencil size={15}');
    expect(src).toContain('<Trash2 size={15}');
  });

  it('handles empty permissions state', () => {
    expect(src).toContain('No permissions assigned');
  });
});
