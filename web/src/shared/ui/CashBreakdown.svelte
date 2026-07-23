<script lang="ts">
  let {
    total = $bindable(0),
    disabled = false,
  }: {
    total?: number;
    disabled?: boolean;
  } = $props();

  function fmt(n: number): string {
    return n.toLocaleString('id-ID');
  }

  const denominations = [
    { label: '100rb', value: 100000 },
    { label: '50rb', value: 50000 },
    { label: '20rb', value: 20000 },
    { label: '10rb', value: 10000 },
    { label: '5rb', value: 5000 },
    { label: '2rb', value: 2000 },
    { label: '1rb', value: 1000 },
    { label: '500', value: 500 },
    { label: '200', value: 200 },
    { label: '100', value: 100 },
  ];

  let counts: Record<number, number> = $state({});

  $effect(() => {
    total = Object.entries(counts).reduce((sum, [denom, count]) => sum + Number(denom) * count, 0);
  });

  function handleCountChange(denom: number, raw: string) {
    const v = raw === '' ? 0 : parseInt(raw, 10);
    counts = { ...counts, [denom]: isNaN(v) ? 0 : Math.max(0, v) };
  }

  function getCount(denom: number): string {
    const v = counts[denom];
    return v !== undefined && v > 0 ? String(v) : '';
  }
</script>

<div class="space-y-3">
  <div class="grid grid-cols-3 gap-2">
    {#each denominations as d}
      <div class="flex items-center gap-2 px-3 py-2 bg-surface-default rounded-lg border border-border/50">
        <span class="text-sm font-medium text-text-primary w-14 shrink-0">{d.label}</span>
        <input
          type="number"
          min="0"
          value={getCount(d.value)}
          {disabled}
          class="w-14 h-8 bg-bg-secondary border border-border-default rounded-lg px-1 text-sm text-text-primary text-center outline-none focus:border-primary-default transition-colors [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
          oninput={(e) => handleCountChange(d.value, (e.target as HTMLInputElement).value)}
        />
        <span class="text-xs text-text-muted shrink-0">×</span>
        <span class="text-xs text-text-primary font-medium ml-auto whitespace-nowrap">
          Rp{fmt((counts[d.value] || 0) * d.value)}
        </span>
      </div>
    {/each}
  </div>
  <div class="flex items-center justify-between px-3 py-2 border-t border-border">
    <span class="text-sm font-semibold text-text-primary">Total</span>
    <span class="text-lg font-bold text-primary">Rp{fmt(total)}</span>
  </div>
</div>
