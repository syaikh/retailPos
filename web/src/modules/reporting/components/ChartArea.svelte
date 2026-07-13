<script>
  import { chart } from '$shared/actions/chart';

  let {
    loading = true,
    chartData = [],
    chartConfig = {},
    chartCanvas = $bindable(),
  } = $props();
</script>

<div class="h-64 relative">
  {#if loading}
    <div class="absolute inset-0 flex items-center justify-center rounded-xl border border-dashed border-primary/30 bg-primary-subtle/10 shadow-glow-primary-sm overflow-hidden">
      <div class="absolute inset-0 bg-linear-to-r from-transparent via-primary-subtle/20 to-transparent animate-shimmer" style="background-size: 200% 100%;"></div>
    </div>
  {:else if chartData.length === 0}
    <div class="absolute inset-0 flex items-center justify-center text-text-muted">
      No data available for this period
    </div>
  {:else}
    <canvas bind:this={chartCanvas} use:chart={chartConfig} aria-label="Sales chart"></canvas>
  {/if}
</div>
