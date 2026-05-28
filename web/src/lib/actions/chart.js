import Chart from 'chart.js/auto';

export function chart(node, options) {
  let chartInstance = new Chart(node, options);

  return {
    update(newOptions) {
      // Ensure proper reactivity - Chart.js needs explicit data updates
      chartInstance.data.labels = newOptions.data.labels;
      
      // Replace datasets entirely to ensure reactivity
      chartInstance.data.datasets = newOptions.data.datasets.map(ds => ({...ds}));
      
      chartInstance.options = newOptions.options;
      chartInstance.update('active'); // Use 'active' for smooth animation
    },
    destroy() {
      chartInstance.destroy();
    }
  };
}
