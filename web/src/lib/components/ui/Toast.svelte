<script lang="ts">
	import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-svelte';
	import { toasts, ui } from '$lib/stores/ui';
	import { fade, slide } from 'svelte/transition';

	function getIcon(type: string) {
		switch (type) {
			case 'success': return CheckCircle;
			case 'error': return AlertCircle;
			case 'warning': return AlertTriangle;
			default: return Info;
		}
	}

	function getTypeClass(type: string) {
		switch (type) {
			case 'success': return 'toast-success';
			case 'error': return 'toast-error';
			case 'warning': return 'toast-warning';
			default: return 'toast-info';
		}
	}
</script>

<div class="toast-container">
	{#each $toasts as toast (toast.id)}
		<div
			class="toast {getTypeClass(toast.type)}"
			transition:slide={{ duration: 200 }}
		>
			<svelte:component this={getIcon(toast.type)} size={20} />
			<span class="toast-message">{toast.message}</span>
			<button class="toast-close" onclick={() => ui.removeToast(toast.id)}>
				<X size={16} />
			</button>
		</div>
	{/each}
</div>

<style>
	.toast-container {
		position: fixed;
		top: 20px;
		right: 20px;
		z-index: 9999;
		display: flex;
		flex-direction: column;
		gap: 8px;
		max-width: 400px;
	}

	.toast {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 16px;
		border-radius: 8px;
		background: var(--color-bg-elevated);
		border: 1px solid var(--color-border);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
		font-size: 0.9rem;
	}

	.toast-success {
		border-left: 3px solid var(--color-success);
		color: var(--color-success);
	}

	.toast-error {
		border-left: 3px solid var(--color-danger);
		color: var(--color-danger);
	}

	.toast-warning {
		border-left: 3px solid var(--color-warning);
		color: var(--color-warning);
	}

	.toast-info {
		border-left: 3px solid var(--color-primary);
		color: var(--color-primary);
	}

	.toast-message {
		flex: 1;
		color: var(--color-text-primary);
	}

	.toast-close {
		background: transparent;
		border: none;
		color: var(--color-text-secondary);
		cursor: pointer;
		padding: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 4px;
		transition: all 0.2s;
	}

	.toast-close:hover {
		background: rgba(255, 255, 255, 0.1);
		color: var(--color-text-primary);
	}
</style>
