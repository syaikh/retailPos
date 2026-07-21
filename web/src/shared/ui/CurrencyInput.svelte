<script lang="ts">
  import { cn } from '$shared/utils/cn';

  let {
    value = $bindable(0),
    placeholder = '0',
    class: className = '',
    disabled = false,
    required = false,
    id = '',
  }: {
    value?: number;
    placeholder?: string;
    class?: string;
    disabled?: boolean;
    required?: boolean;
    id?: string;
  } = $props();

  let displayValue = $state('');

  function formatCurrency(num: number): string {
    return num ? num.toLocaleString('id-ID') : '';
  }

  function handleInput(e: Event) {
    const raw = (e.target as HTMLInputElement).value.replace(/[^0-9]/g, '');
    value = raw ? parseInt(raw, 10) : 0;
    displayValue = formatCurrency(value);
    const input = e.target as HTMLInputElement;
    if (input.value !== displayValue) {
      input.value = displayValue;
      input.setSelectionRange(displayValue.length, displayValue.length);
    }
  }

  $effect(() => {
    displayValue = formatCurrency(value);
  });
</script>

<div class={cn(
  'flex items-center gap-1.5 bg-bg-secondary border border-border-default rounded-xl px-3 h-[42px] w-full transition-colors duration-200',
  value > 0 ? 'border-primary-default' : '',
  disabled ? 'opacity-40 cursor-not-allowed' : '',
  className
)}>
  <span class="text-xs text-text-muted font-medium shrink-0 select-none">Rp</span>
  <input
    {id}
    type="text"
    inputmode="numeric"
    value={displayValue}
    {placeholder}
    {disabled}
    {required}
    class="w-full bg-transparent text-sm text-right text-text-primary outline-none placeholder:text-text-muted disabled:cursor-not-allowed"
    oninput={handleInput}
  />
</div>
