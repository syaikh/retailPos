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
		font-weight: 800;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		background: linear-gradient(135deg, var(--primary) 0%, #818cf8 100%);
		box-shadow: 0 4px 15px rgba(99, 102, 241, 0.4);
		border: none;
		padding: 14px 24px;
		font-size: 1rem;
		transition: all 0.3s ease;
		position: relative;
		overflow: hidden;
	}

	.checkout-btn::before {
		content: '';
		position: absolute;
		top: 0;
		left: -100%;
		width: 100%;
		height: 100%;
		background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.2), transparent);
		transition: left 0.5s ease;
	}

	.checkout-btn:hover:not(:disabled)::before {
		left: 100%;
	}

	.checkout-btn:hover:not(:disabled) {
		box-shadow: 0 6px 20px rgba(99, 102, 241, 0.6);
		transform: translateY(-2px);
	}

	.checkout-btn:active:not(:disabled) {
		transform: translateY(0px);
		box-shadow: 0 2px 10px rgba(99, 102, 241, 0.4);
	}
</style>
