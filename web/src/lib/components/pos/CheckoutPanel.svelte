<script lang="ts">
import Button from '$lib/components/ui/Button.svelte';
import { CreditCard, Banknote } from 'lucide-svelte';

	let {
		total = 0,
		paymentMethod = $bindable('cash'),
		isLoading = false,
		onCheckout
	}: {
		total: number;
		paymentMethod: 'cash' | 'card';
		isLoading: boolean;
		onCheckout: () => void;
	} = $props();
</script>

<div class="checkout-panel">
	<div class="payment-methods">
		<button
			class="method-btn"
			class:active={paymentMethod === 'cash'}
			onclick={() => paymentMethod = 'cash'}
		>
			<Banknote size={16} />
			Tunai
		</button>
		<button
			class="method-btn"
			class:active={paymentMethod === 'card'}
			onclick={() => paymentMethod = 'card'}
		>
			<CreditCard size={16} />
			Kartu
		</button>
	</div>

	<div class="total-section">
		<div class="total-label">Total</div>
		<div class="total-amount">Rp {total.toLocaleString()}</div>
	</div>

	<Button
		class="checkout-btn"
		variant="primary"
		size="lg"
		loading={isLoading}
		disabled={isLoading || total <= 0}
		onclick={onCheckout}
	>
		{isLoading ? 'Memproses...' : 'BAYAR SEKARANG'}
	</Button>
</div>

<style>
	.checkout-panel {
		display: flex;
		flex-direction: column;
		gap: 20px;
		padding-top: 16px;
		border-top: 2px solid var(--border);
		flex-shrink: 0;
	}

	.payment-methods {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 8px;
	}

	.method-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		padding: 10px 12px;
		background: var(--bg-main);
		color: var(--text-secondary);
		border: 2px solid transparent;
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.2s;
		font-size: 0.875rem;
	}

	.method-btn:hover {
		border-color: var(--border);
		color: var(--text-primary);
	}

	.method-btn.active {
		background: var(--primary);
		color: white;
		border-color: var(--primary);
	}

	.total-section {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 12px;
		background: rgba(99, 102, 241, 0.1);
		border-radius: 8px;
		border: 1px solid var(--border);
	}

	.total-label {
		font-size: 0.875rem;
		color: var(--text-secondary);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.total-amount {
		font-size: 1.25rem;
		font-weight: 800;
		color: var(--accent);
	}

	.checkout-btn {
		font-weight: 700;
		letter-spacing: 0.05em;
		text-transform: uppercase;
		box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
		transition: all 0.2s ease;
	}

	.checkout-btn:hover:not(:disabled) {
		box-shadow: 0 6px 16px rgba(99, 102, 241, 0.4);
		transform: translateY(-1px);
	}
</style>
