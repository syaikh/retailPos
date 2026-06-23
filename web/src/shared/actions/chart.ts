import Chart from 'chart.js/auto';

export function chart(node, config) {
  let chartInstance = null;

  // Create chart immediately
  try {
    chartInstance = new Chart(node, config);
  } catch (e) {
    console.error('Failed to create chart:', e);
  }

  return {
    update(newConfig) {
      if (chartInstance && newConfig) {
        if (newConfig.data?.labels) {
          chartInstance.data.labels = [...newConfig.data.labels];
        }
        
        if (newConfig.data?.datasets) {
          chartInstance.data.datasets = newConfig.data.datasets.map(ds => ({...ds}));
        }
        
        if (newConfig.options) {
          chartInstance.options = {...newConfig.options};
        }
        
        chartInstance.update('active');
      }
    },
    destroy() {
      if (chartInstance) {
        chartInstance.destroy();
      }
    }
  };
}
