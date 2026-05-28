<script>
	let {
		label,
		value,
		sub = '',
		trend = null,
		trendLabel = 'vs kemarin',
		displayTrend = false,
		icon: Icon,
		iconBg = 'bg-primary-subtle',
		iconColor = 'text-primary-light',
		loading = false,
	} = $props();

	const trendUp = $derived(trend !== null && trend > 0);
	const trendDown = $derived(trend !== null && trend < 0);
</script>

<div class="flex items-center gap-4">
	<div class="w-14 h-14 rounded-xl {iconBg} flex items-center justify-center shrink-0">
		<Icon size={28} class={iconColor} />
	</div>

	<div class="flex-1 min-w-0 text-left">
		{#if loading}
			<div class="skeleton h-10 w-48 mb-2"></div>
			<div class="skeleton h-6 w-32"></div>
			<div class="skeleton h-4 w-24"></div>
		{:else}
			<p class="text-4xl font-bold text-text-primary leading-none truncate">{value}</p>
			{#if displayTrend && trend !== null}
				<p class="flex items-center gap-1.5 text-sm font-medium mt-1 {trendUp ? 'text-success-light' : trendDown ? 'text-danger-light' : 'text-text-muted'}">
					{trendUp ? '↑' : trendDown ? '↓' : '—'} {trend ? Math.abs(trend) : 0}% {trendLabel}
				</p>
			{:else if sub}
				<p class="text-base text-text-muted mt-1 truncate">{sub}</p>
			{/if}
			<p class="text-sm text-text-muted mt-3 font-medium uppercase tracking-wide">{label}</p>
		{/if}

		{#if trend !== null && !displayTrend}
			<span class="inline-block text-sm font-semibold px-2.5 py-1 rounded-full mt-2 {trendUp ? 'badge-success' : trendDown ? 'badge-danger' : 'badge-muted'}">
				{trendUp ? '↑' : trendDown ? '↓' : '—'} {trend ? Math.abs(trend) : 0}%
			</span>
		{/if}
	</div>
</div>