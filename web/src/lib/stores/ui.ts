import { writable, derived } from 'svelte/store';

interface Toast {
	id: number;
	message: string;
	type: 'success' | 'error' | 'info' | 'warning';
	duration?: number;
}

interface UIState {
	toasts: Toast[];
	isLoading: boolean;
	error: string | null;
}

let toastId = 0;

function createUIStore() {
	const { subscribe, set, update } = writable<UIState>({
		toasts: [],
		isLoading: false,
		error: null
	});

	// Helper to add toast
	function addToast(message: string, type: Toast['type'] = 'info', duration = 5000): number {
		const id = ++toastId;
		update(state => ({
			...state,
			toasts: [...state.toasts, { id, message, type, duration }]
		}));

		if (duration > 0) {
			setTimeout(() => {
				update(state => ({
					...state,
					toasts: state.toasts.filter(t => t.id !== id)
				}));
			}, duration);
		}
		return id;
	}

	return {
		subscribe,

		addToast,
		removeToast: (id: number) => {
			update(state => ({
				...state,
				toasts: state.toasts.filter(t => t.id !== id)
			}));
		},

		success: (message: string, duration?: number): number => {
			return addToast(message, 'success', duration);
		},

		error: (message: string, duration?: number): number => {
			return addToast(message, 'error', duration);
		},

		info: (message: string, duration?: number): number => {
			return addToast(message, 'info', duration);
		},

		warning: (message: string, duration?: number): number => {
			return addToast(message, 'warning', duration);
		},

		setLoading: (loading: boolean) => {
			update(state => ({ ...state, isLoading: loading }));
		},

		setError: (error: string | null) => {
			update(state => ({ ...state, error }));
		},

		clearError: () => {
			update(state => ({ ...state, error: null }));
		}
	};
}

export const ui = createUIStore();

// Derived stores for convenience
export const toasts = derived(ui, $ui => $ui.toasts);
export const isLoading = derived(ui, $ui => $ui.isLoading);
export const error = derived(ui, $ui => $ui.error);
