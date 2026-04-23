<script lang="ts">
	import type { CartItem } from '$lib/domain/entities';
	import { Plus, Trash2, Minus } from 'lucide-svelte';

	let {
		items,
		onRemove,
		onIncrement,
		onDecrement,
		onSetQuantity,
		onClear
	}: {
		items: CartItem[];
		onRemove: (id: number) => void;
		onIncrement: (id: number) => void;
		onDecrement: (id: number) => void;
		onSetQuantity: (id: number, qty: number) => void;
		onClear: () => void;
	} = $props();
</script>

<div class="cart-panel">
	<div class="cart-header">
		<h2>Keranjang</h2>
		<button class="clear-btn" onclick={onClear}>Clear All</button>
	</div>

	<div class="cart-items">
		{#if items.length === 0}
			<div class="empty-state">
				<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>
				<p>Keranjang kosong</p>
			</div>
		{:else}
			{#each items as item (item.id)}
				<div class="cart-item">
					<div class="item-info">
						<div class="item-name">{item.name}</div>
						<div class="item-price">Rp {item.price.toLocaleString()}</div>
					</div>
					<div class="item-actions">
						<button
							class="qty-btn"
							onclick={() => onDecrement(item.id)}
							disabled={item.quantity <= 1}
						>
							<Minus size={14} />
						</button>
						<input
							type="number"
							class="qty-input"
							value={item.quantity}
							min="1"
							max={item.stock}
							onchange={(e) => onSetQuantity(item.id, parseInt(e.currentTarget.value))}
						/>
						<button
							class="qty-btn"
							onclick={() => onIncrement(item.id)}
							disabled={item.quantity >= item.stock}
						>
							<Plus size={14} />
						</button>
						<button class="remove-btn" onclick={() => onRemove(item.id)}>
							<Trash2 size={16} />
						</button>
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>

<style>
	.cart-panel {
		display: flex;
		flex-direction: column;
		height: 100%;
		gap: 16px;
	}

	.cart-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-bottom: 12px;
		border-bottom: 1px solid var(--border);
	}

	.cart-header h2 {
		margin: 0;
		font-size: 1.25rem;
		font-weight: 700;
	}

	.clear-btn {
		background: transparent;
		color: var(--text-secondary);
		font-size: 0.875rem;
		cursor: pointer;
		transition: color 0.2s;
	}

	.clear-btn:hover {
		color: var(--danger);
	}

	.cart-items {
		flex: 1;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		color: var(--text-secondary);
		gap: 12px;
	}

	.cart-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 12px;
		background: rgba(255, 255, 255, 0.03);
		border-radius: 8px;
		gap: 12px;
	}

	.item-info {
		flex: 1;
		min-width: 0;
	}

	.item-name {
		font-weight: 600;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.item-price {
		font-size: 0.875rem;
		color: var(--text-secondary);
		margin-top: 2px;
	}

	.item-actions {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.qty-btn {
		width: 28px;
		height: 28px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--bg-main);
		color: var(--text-primary);
		border: 1px solid var(--border);
		border-radius: 4px;
		cursor: pointer;
		transition: all 0.2s;
	}

	.qty-btn:hover:not(:disabled) {
		background: var(--primary);
		border-color: var(--primary);
	}

	.qty-btn:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	.qty-input {
		width: 48px;
		height: 28px;
		text-align: center;
		background: transparent;
		border: 1px solid var(--border);
		color: var(--text-primary);
		font-weight: 600;
		border-radius: 4px;
		appearance: textfield;
	}

	.qty-input::-webkit-outer-spin-button,
	.qty-input::-webkit-inner-spin-button {
		-webkit-appearance: none;
		margin: 0;
	}

	.qty-input[type=number] {
		-moz-appearance: textfield;
		appearance: textfield;
	}

	.remove-btn {
		background: transparent;
		color: var(--text-secondary);
		padding: 4px;
		border-radius: 4px;
		cursor: pointer;
		transition: color 0.2s;
	}

	.remove-btn:hover {
		color: var(--danger);
	}
</style>
