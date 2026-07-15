import Chart, { type ChartConfiguration } from 'chart.js/auto';

export function chart(node: HTMLCanvasElement, config: ChartConfiguration) {
  let chartInstance: Chart | null = null;

  // Create chart immediately
  try {
    chartInstance = new Chart(node, config);
  } catch (e) {
    console.error('Failed to create chart:', e);
  }

  return {
    update(newConfig: ChartConfiguration) {
      if (chartInstance && newConfig) {
        if (newConfig.data?.labels) {
          chartInstance.data.labels = [...newConfig.data.labels];
        }
        
        if (newConfig.data?.datasets) {
          chartInstance.data.datasets = newConfig.data.datasets.map((ds) => ({...ds} as any));
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
