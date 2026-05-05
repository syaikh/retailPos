import Chart from 'chart.js/auto';

export function chart(node, options) {
  let chartInstance = new Chart(node, options);

  return {
    update(newOptions) {
      chartInstance.data = newOptions.data;
      chartInstance.options = newOptions.options;
      chartInstance.update();
    },
    destroy() {
      chartInstance.destroy();
    }
  };
}
