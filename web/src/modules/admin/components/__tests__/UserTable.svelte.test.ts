import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'UserTable.svelte'), 'utf-8');
}

describe('UserTable.svelte source-structure guards', () => {
  const src = getSource();

  it('imports Badge, Button, Skeleton from shared/ui', () => {
    expect(src).toContain("import { Badge, Button, Skeleton, SortableHeader } from '$shared/ui'");
  });

  it('imports jakartaTime formatters', () => {
    expect(src).toContain("import { formatDateInJakarta, formatTimeInJakarta } from '$shared/utils/jakartaTime'");
  });

  it('uses $bindable for sortBy and sortDir', () => {
    expect(src).toContain('sortBy = $bindable');
    expect(src).toContain('sortDir = $bindable');
  });

  it('has event callbacks (onsort, onedit, ondelete)', () => {
    expect(src).toContain('onsort');
    expect(src).toContain('onedit');
    expect(src).toContain('ondelete');
  });

  it('has roleVariant helper function', () => {
    expect(src).toContain('function roleVariant');
  });

  it('handles loading state with Skeleton', () => {
    expect(src).toContain('{#if loading}');
  });

  it('handles empty state', () => {
    expect(src).toContain('No users found');
  });

  it('disables delete for current user and superadmin', () => {
    expect(src).toContain('user.id === currentUserID || user.role_id === 1');
  });
});
