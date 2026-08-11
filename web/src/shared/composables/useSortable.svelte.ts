export function useSortable(
  initialColumn = 'name',
  initialDirection: 'asc' | 'desc' = 'asc',
  onChange?: () => void
) {
  const sortState = $state({ sortBy: initialColumn, sortDir: initialDirection });

  function handleSort(column: string) {
    if (sortState.sortBy === column) {
      sortState.sortDir = sortState.sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortState.sortBy = column;
      sortState.sortDir = 'asc';
    }
    onChange?.();
  }

  return { sortState, handleSort };
}
