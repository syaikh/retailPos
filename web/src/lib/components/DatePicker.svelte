<script lang="ts">
  import { DateInput } from 'date-picker-svelte';
  
  let { 
    value = $bindable(null), 
    min = null, 
    max = null, 
    format = 'yyyy-MM-dd', 
    locale = {}, 
    placeholder = 'Pilih tanggal',
    onchange
  } = $props();
  

  
  // Working value that gets updated as user interacts with calendar
  let workingValue: Date | null | undefined = $state(value);
  // Flag to track if we're showing the picker
  let showPicker = $state(false);
  
  // Convert null to undefined for min/max to prevent library crashes
  let safeMin = $derived(min === null ? undefined : min);
  let safeMax = $derived(max === null ? undefined : max);

  // Reset working value when picker is closed without committing
  $effect(() => {
    if (!showPicker) {
      if (value && value instanceof Date) {
        workingValue = new Date(value.getTime());
      } else {
        workingValue = undefined;
      }
    }
  });

  // Initialize working value when external value changes
  $effect(() => {
    if (value && value instanceof Date) {
      // Create a new Date object to avoid reference issues
      workingValue = new Date(value.getTime());
    } else {
      workingValue = undefined;
    }
  });
  
  function handleDateSelect(e: any) {
    // Update working value when user selects a date
    const selectedDate = e.detail;
    if (selectedDate && selectedDate instanceof Date) {
      workingValue = new Date(selectedDate.getTime());
    }
  }
  
  function handleOkClick(e: MouseEvent) {
    e.stopPropagation();
    // Only update the bound value when OK is clicked
    if (workingValue && workingValue instanceof Date) {
      value = new Date(workingValue.getTime());
      onchange?.(value);
    }
    // Hide picker
    showPicker = false;
  }
  
  function handleCancelClick(e: MouseEvent) {
    e.stopPropagation();
    // Reset working value to match bound value
    if (value && value instanceof Date) {
      workingValue = new Date(value.getTime());
    } else {
      workingValue = undefined;
    }
    // Hide picker
    showPicker = false;
  }
  
  // Hide empty week rows in the date picker
  let observer: MutationObserver | null = null;
  
  function hideEmptyWeeks() {
    if (typeof document === 'undefined') return;
    
    const picker = document.querySelector('.date-input-wrapper .picker');
    if (!picker) return;
    
    const weeks = picker.querySelectorAll('.week');
    weeks.forEach((week) => {
      const cells = week.querySelectorAll('.cell');
      let allOtherMonth = true;
      
      cells.forEach((cell) => {
        if (!cell.classList.contains('other-month')) {
          allOtherMonth = false;
        }
      });
      
      // Hide week if all cells are from other months
      if (allOtherMonth && cells.length > 0) {
        (week as HTMLElement).style.display = 'none';
      } else {
        (week as HTMLElement).style.display = 'flex';
      }
    });
  }
  
  function setupObserver() {
    if (typeof document === 'undefined') return;
    
    // Disconnect previous observer
    if (observer) {
      observer.disconnect();
    }
    
    // Use setTimeout to ensure picker is rendered
    setTimeout(() => {
      const picker = document.querySelector('.date-input-wrapper .picker');
      if (!picker) return;
      
      // Hide empty weeks initially
      hideEmptyWeeks();
      
      // Observe changes to the picker (e.g., month navigation)
      observer = new MutationObserver(() => {
        hideEmptyWeeks();
      });
      
      observer.observe(picker, {
        childList: true,
        subtree: true
      });
    }, 50);
  }
  
  // Watch for showPicker changes to hide empty weeks and setup observer
  $effect(() => {
    if (showPicker) {
      setupObserver();
    } else {
      // Cleanup observer when picker is closed
      if (observer) {
        observer.disconnect();
        observer = null;
      }
    }
    
    // Cleanup on component destroy
    return () => {
      if (observer) {
        observer.disconnect();
        observer = null;
      }
    };
  });
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="date-input-wrapper" onclick={() => showPicker = true} role="none">
  <DateInput 
    bind:value={workingValue}
    min={safeMin}
    max={safeMax}
    {format}
    {locale}
    {placeholder}
    closeOnSelection={false}
    on:select={handleDateSelect}
    bind:visible={showPicker}
  >
    <div class="picker-footer" 
         role="presentation"
         onclick={(e) => e.stopPropagation()} 
         onkeydown={(e) => e.stopPropagation()}>
      <button type="button" class="cancel-btn" 
              onclick={handleCancelClick}>Cancel</button>
      <button type="button" class="ok-btn" 
              onclick={handleOkClick}>OK</button>
    </div>
  </DateInput>
</div>

<style>
  .date-input-wrapper {
    position: relative;
    display: inline-block;
  }
  
  :global(.date-input-wrapper .picker) {
    overflow: visible !important;
    z-index: 1000;
  }
  
  .picker-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 8px 12px;
    border-top: 1px solid var(--border, rgba(103, 113, 137, 0.3));
    background: var(--bg-surface, #1e293b);
    border-radius: 0 0 4px 4px;
  }
  
  .picker-footer button {
    padding: 4px 12px;
    border-radius: 4px;
    font-size: 0.75rem;
    cursor: pointer;
    border: none;
    font-family: inherit;
    transition: background-color 0.15s ease;
  }
  
  .ok-btn {
    background: #22c55e;
    color: white;
  }
  
  .ok-btn:hover {
    background: #16a34a;
  }
  
  .cancel-btn {
    background: transparent;
    color: var(--text-secondary, #9ca3af);
    border: 1px solid var(--border, #4b5563);
  }
  
  .cancel-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: white;
  }
  
  /* Hide week rows that only contain other-month days */
  :global(.picker .week) {
    display: flex;
  }
  
  :global(.picker .week:empty) {
    display: none;
  }
</style>