<script>
	let { state: periodState } = $props();
	let open = $state(false);
	let ignoreNextWindowClick = $state(false);
	let hoveredOptionId = $state(null);

	const periodOptions = [
		{ id: 'realtime', label: 'Real-time' },
		{ id: 'yesterday', label: 'Kemarin' },
		{ id: '7days', label: '7 Hari Sebelumnya' },
		{ id: '30days', label: '30 Hari Sebelumnya' },
		{ id: 'weekly', label: 'Per Minggu' },
		{ id: 'monthly', label: 'Per Bulan' },
		{ id: 'yearly', label: 'Berdasarkan Tahun' }
	];

	function handlePeriodSelect(periodId) {
		periodState.updatePeriodType(periodId);
		open = false;
	}

	function toggleDropdown(e) {
		e.stopPropagation();
		ignoreNextWindowClick = true;
		open = !open;
	}

	function onWindowClick() {
		if (ignoreNextWindowClick) {
			ignoreNextWindowClick = false;
			return;
		}
		open = false;
	}

	function getOptionDescription(id) {
		const today = new Date();
		const yesterday = new Date(today);
		yesterday.setDate(yesterday.getDate() - 1);

		switch (id) {
			case 'realtime': {
				const currentHour = String(today.getHours()).padStart(2, '0');
				return `Hari ini (00:00) - ${currentHour}:00`;
			}
			case 'yesterday': {
				const y = String(yesterday.getDate()).padStart(2, '0');
				const m = String(yesterday.getMonth() + 1).padStart(2, '0');
				const d = String(yesterday.getFullYear());
				return `${y}-${m}-${d}`;
			}
			default:
				return periodState.getPeriodDescription();
		}
	}

	function getRightPanelText() {
		if (hoveredOptionId) {
			return getOptionDescription(hoveredOptionId);
		}
		return periodState.getRightPanelContent();
	}
</script>

<div class="relative inline-block">
	<button
		type="button"
		class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-text-primary bg-surface-subtle border border-border rounded-lg hover:bg-surface-hover transition-colors"
		onclick={toggleDropdown}
		aria-haspopup="dialog"
		aria-expanded={open}
	>
		<span>{periodState.getButtonLabel()}</span>
		<span class="text-text-muted">▼</span>
	</button>

	{#if open}
		<div
			class="absolute left-0 top-full mt-2 w-[500px] bg-slate-900 border border-slate-800 rounded-xl shadow-xl z-50 flex"
			role="dialog"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
		>
			<div class="w-48 p-3 border-r border-slate-800">
				<div class="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-2 px-2">Pilihan Cepat</div>
				{#each periodOptions as option}
					<button
						class="w-full px-3 py-2 text-left text-sm rounded-lg transition-all duration-200 {periodState.selectedPeriodType === option.id ? 'bg-purple-600 text-white' : 'text-slate-100 hover:bg-slate-800'}"
						onclick={(e) => { e.stopPropagation(); handlePeriodSelect(option.id); }}
						onmouseenter={() => hoveredOptionId = option.id}
						onmouseleave={() => hoveredOptionId = null}
					>
						{option.label}
					</button>
				{/each}
			</div>
			<div class="flex-1 p-3 flex items-center text-sm text-slate-100">
				{getRightPanelText()}
			</div>
		</div>
	{/if}
</div>

<svelte:window onclick={onWindowClick} onkeydown={(e) => { if (e.key === "Escape") { open = false; } }} />