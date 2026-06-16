/**
 * State management composable for period selection in ReportsPage
 * Uses Svelte 5 Runes syntax
 */
import { getTodayInJakarta, getDateNDaysAgoInJakarta, getFirstOfMonthNAgoInJakarta, getCurrentJakartaHour } from '$lib/utils/jakartaTime';

export function usePeriodState() {
	// State using Svelte 5 Runes
	let selectedPeriodType = $state('realtime');
	let dateRange = $state({ start: '', end: '' });
	let chartType = $state('line');
	let userOverriddenTab = $state(false);
	let activeTab = $state('daily');

	// Derived values
	const totalDays = $derived.by(() => {
		if (!dateRange.start || !dateRange.end) return 0;
		const start = new Date(dateRange.start);
		const end = new Date(dateRange.end);
		const diffTime = Math.abs(end - start);
		return Math.ceil(diffTime / (1000 * 60 * 60 * 24)) + 1;
	});

	const autoTab = $derived.by(() => {
		if (userOverriddenTab) return activeTab;
		const days = totalDays;
		if (days <= 30) return 'daily';
		if (days <= 180) return 'weekly';
		return 'monthly';
	});

	const autoChartType = $derived.by(() => {
		// PRD 3.4: hourly for realtime/yesterday, daily for 7days/30days, high-unit for weekly/monthly/yearly
		if (selectedPeriodType === 'realtime' || selectedPeriodType === 'yesterday') return 'hourly';
		if (['7days', '30days', 'daily'].includes(selectedPeriodType)) return 'daily';
		return 'high-unit'; // weekly, monthly, yearly
	});

	// Live time state for realtime updates
	let currentTimeHour = $state(`${String(getCurrentJakartaHour()).padStart(2, '0')}:00`);

	// Effect to update time every minute when in realtime mode
	$effect(() => {
		if (selectedPeriodType !== 'realtime') return;

		function updateTime() {
			currentTimeHour = `${String(getCurrentJakartaHour()).padStart(2, '0')}:00`;
		}

		updateTime();
		const interval = setInterval(updateTime, 60000);
		return () => clearInterval(interval);
	});

	// Effect for auto-updating
	$effect(() => {
		dateRange;
		if (!userOverriddenTab) {
			activeTab = autoTab;
		}
	});

	// Helper to update period type
	function updatePeriodType(newType) {
		selectedPeriodType = newType;
		userOverriddenTab = true;

		const today = getTodayInJakarta();
		const daysAgo = getDateNDaysAgoInJakarta;

		switch (newType) {
			case 'realtime':
				dateRange = { start: today, end: today };
				chartType = 'hourly';
				break;
			case 'yesterday': {
				const yesterday = daysAgo(1);
				dateRange = { start: yesterday, end: yesterday };
				chartType = 'hourly';
				break;
			}
			case '7days':
				dateRange = { start: daysAgo(8), end: daysAgo(1) };
				break;
			case '30days':
				dateRange = { start: daysAgo(31), end: daysAgo(1) };
				break;
			case 'daily':
				dateRange = { start: daysAgo(7), end: today };
				break;
			case 'weekly':
				dateRange = { start: daysAgo(89), end: today };
				break;
			case 'monthly':
				dateRange = { start: getFirstOfMonthNAgoInJakarta(11), end: today };
				break;
			case 'yearly':
				dateRange = { start: getFirstOfMonthNAgoInJakarta(11), end: today };
				break;
		}
	}

	// Helper to handle tab change
	function handleTabChange(newTab) {
		activeTab = newTab;
		userOverriddenTab = true;

		const today = getTodayInJakarta();

		if (newTab === 'daily') {
			dateRange = { start: getDateNDaysAgoInJakarta(7), end: today };
		} else if (newTab === 'weekly') {
			dateRange = { start: getDateNDaysAgoInJakarta(89), end: today };
		} else if (newTab === 'monthly') {
			dateRange = { start: getFirstOfMonthNAgoInJakarta(11), end: today };
		}
	}

	// Format date for display (Jakarta timezone)
	function formatDateDisplay(dateString) {
		if (!dateString) return '';
		// Parse as UTC to avoid timezone shift, format in Jakarta
		const date = new Date(dateString + 'T00:00:00Z');
		const day = String(date.getUTCDate()).padStart(2, '0');
		const month = String(date.getUTCMonth() + 1).padStart(2, '0');
		const year = date.getUTCFullYear();
		return `${day}-${month}-${year}`;
	}

	// Get label for Card 4 based on period type
	function getAvgRevenueLabel() {
		switch (selectedPeriodType) {
			case 'realtime':
			case 'yesterday':
				return 'Peak Revenue Hour';
			case 'weekly':
				return 'Avg. Revenue / Week';
			case 'monthly':
				return 'Avg. Revenue / Week';
			case 'yearly':
				return 'Avg. Revenue / Month';
			default:
				return 'Avg. Revenue / Day';
		}
	}

	// Get timezone display string (client's local timezone)
	function getTimezoneDisplay() {
		const offset = -new Date().getTimezoneOffset() / 60;
		const sign = offset >= 0 ? '+' : '-';
		return `(GMT${sign}${String(Math.abs(offset)).padStart(2, '0')})`;
	}

	// Get period description for trigger
	function getPeriodDescription() {
		const start = formatDateDisplay(dateRange.start);
		const end = formatDateDisplay(dateRange.end);
		const tz = getTimezoneDisplay();

		switch (selectedPeriodType) {
			case 'realtime':
				return `Real-time. ${formatDateDisplay(getTodayInJakarta())} ${tz}`;
			case 'yesterday':
				return `Kemarin. ${start} ${tz}`;
			case '7days':
				return `7 hari sebelumnya. ${start} - ${end} ${tz}`;
			case '30days':
				return `30 hari sebelumnya. ${start} - ${end} ${tz}`;
			case 'weekly':
				return `Per minggu. ${start} - ${end} ${tz}`;
			case 'monthly':
				return `Per bulan. ${start} - ${end} ${tz}`;
			case 'yearly':
				return `Berdasarkan tahun. ${start} - ${end} ${tz}`;
			default:
				return `${start} - ${end} ${tz}`;
		}
	}

	function getButtonLabel() {
		if (selectedPeriodType === 'realtime') {
			return `Real-time (00:00 - ${currentTimeHour})`;
		}
		return getPeriodDescription();
	}

	function getRightPanelContent() {
		if (selectedPeriodType === 'realtime') {
			return `Hari ini (00:00) - ${currentTimeHour}`;
		}
		return getPeriodDescription();
	}

	return {
		// State
		get selectedPeriodType() { return selectedPeriodType; },
		set selectedPeriodType(v) { selectedPeriodType = v; },
		get dateRange() { return dateRange; },
		set dateRange(v) { dateRange = v; },
		get chartType() { return chartType; },
		set chartType(v) { chartType = v; },
		get userOverriddenTab() { return userOverriddenTab; },
		set userOverriddenTab(v) { userOverriddenTab = v; },
		get activeTab() { return activeTab; },
		set activeTab(v) { activeTab = v; },

		// Derived
		get totalDays() { return totalDays; },
		get autoTab() { return autoTab; },
		get autoChartType() { return autoChartType; },
		get currentTimeHour() { return currentTimeHour; },

		// Methods
		updatePeriodType,
		handleTabChange,
		formatDateDisplay,
		getAvgRevenueLabel,
		getTimezoneDisplay,
		getPeriodDescription,
		getButtonLabel,
		getRightPanelContent
	};
}