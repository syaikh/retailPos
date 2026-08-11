import { describe, it, expect, vi } from 'vitest';
import { useSortable } from '../useSortable.svelte';

describe('useSortable', () => {
  it('exports a function and returns the expected API shape', () => {
    expect(typeof useSortable).toBe('function');
    const s = useSortable();
    expect(s.sortState.sortBy).toBe('name');
    expect(s.sortState.sortDir).toBe('asc');
    expect(typeof s.handleSort).toBe('function');
  });

  it('honors custom initial column and direction', () => {
    const s = useSortable('created_at', 'desc');
    expect(s.sortState.sortBy).toBe('created_at');
    expect(s.sortState.sortDir).toBe('desc');
  });

  it('toggles direction when the same column is re-clicked', () => {
    const s = useSortable('name', 'asc');
    s.handleSort('name');
    expect(s.sortState.sortBy).toBe('name');
    expect(s.sortState.sortDir).toBe('desc');
    s.handleSort('name');
    expect(s.sortState.sortDir).toBe('asc');
  });

  it('resets direction to asc when a new column is clicked', () => {
    const s = useSortable('name', 'desc');
    s.handleSort('created_at');
    expect(s.sortState.sortBy).toBe('created_at');
    expect(s.sortState.sortDir).toBe('asc');
  });

  it('invokes the onChange callback after each sort toggle', () => {
    const onChange = vi.fn();
    const s = useSortable('name', 'asc', onChange);
    s.handleSort('name');
    expect(onChange).toHaveBeenCalledTimes(1);
    s.handleSort('created_at');
    expect(onChange).toHaveBeenCalledTimes(2);
  });

  it('updates sortBy/sortDir when mutated externally', () => {
    const s = useSortable('name', 'asc');
    s.sortState.sortBy = 'status';
    s.sortState.sortDir = 'desc';
    expect(s.sortState.sortBy).toBe('status');
    expect(s.sortState.sortDir).toBe('desc');
  });
});
