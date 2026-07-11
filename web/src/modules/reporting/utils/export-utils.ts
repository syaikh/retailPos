import { apiFetch } from '$shared/api/http-client';
import { toast } from '$shared/stores/toast.svelte';
import { getTodayInJakarta } from '$shared/utils/jakartaTime';
import { getPeriodLabel } from '$modules/reporting/lib/reporting-utils';

interface ExportToExcelParams {
  exportPeriod?: string;
  exportMode?: string;
  exportDate?: string;
  chartCanvas?: HTMLCanvasElement;
}

interface ExportToPDFParams {
  startDate?: string;
  endDate?: string;
  selectedPeriodType: string;
  currentTimeHour?: string;
  chartType: string;
  comparisonDateRange?: string;
  statCardLabels: { comparisonLabel: string };
  kpiData: {
    totalRevenue: number;
    previousRevenue: number;
    totalOrders: number;
    previousOrders: number;
    avgOrderValue: number;
    previousAvgOrderValue: number;
    revenuePerDay: number;
    previousRevenuePerDay: number;
    peakRevenueHour: number;
    previousPeakRevenue: number;
    peakRevenueMonth: number;
    previousPeakRevenueMonth: number;
  };
  chartCanvas?: HTMLCanvasElement;
  chartData: Array<{ date?: string; total: number }>;
  bestPeriod?: { total: number; [key: string]: unknown } | null;
  worstPeriod?: { total: number; [key: string]: unknown } | null;
  bestWorstHeading: string;
  sortedRows: Array<{
    period: string;
    revenue: number;
    prevRevenue: number | null;
    orderCount: number | null;
  }>;
}

/**
 * Exports the current dashboard view to Excel.
 */
export async function exportToExcel({
  exportPeriod,
  exportMode,
  exportDate,
  chartCanvas,
}: ExportToExcelParams) {
  try {
    const formData = new FormData();
    if (exportPeriod) formData.set('period', exportPeriod);
    if (exportMode) formData.set('mode', exportMode);
    if (exportDate) formData.set('date', exportDate);
    if (chartCanvas) {
      const temp = document.createElement('canvas');
      temp.width = chartCanvas.width;
      temp.height = chartCanvas.height;
      const tCtx = temp.getContext('2d');
      tCtx.fillStyle = '#111827';
      tCtx.fillRect(0, 0, temp.width, temp.height);
      tCtx.drawImage(chartCanvas, 0, 0);
      formData.set('chartData', temp.toDataURL('image/png'));
    }

    const res = await apiFetch('/api/dashboard/export', {
      method: 'POST',
      body: formData,
    });
    if (!res.ok) {
      const err = await res.text();
      throw new Error(err || 'Export failed');
    }

    const blob = await res.blob();
    const fileName = `dashboard-${getTodayInJakarta()}.xlsx`;
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = fileName;
    a.click();
    URL.revokeObjectURL(url);

    toast.success('Excel export completed');
  } catch (error) {
    console.error('Excel export error:', error);
    toast.error('Export Excel gagal: ' + (error.message || 'unknown error'));
  }
}

/**
 * Exports the current dashboard view to PDF.
 */
export async function exportToPDF({
  startDate,
  endDate,
  selectedPeriodType,
  currentTimeHour,
  chartType,
  comparisonDateRange,
  statCardLabels,
  kpiData,
  chartCanvas,
  chartData,
  bestPeriod,
  worstPeriod,
  bestWorstHeading,
  sortedRows,
}: ExportToPDFParams) {
  try {
    const { jsPDF } = await import('jspdf');
    const { default: autoTable } = await import('jspdf-autotable');

    const doc = new jsPDF('l', 'mm', 'a4');
    const pageWidth = doc.internal.pageSize.getWidth();
    const margin = 15;
    let yPos = 20;

    const rangeDateFormat = (ds) => {
      if (!ds) return '';
      const d = new Date(ds + 'T00:00:00Z');
      return `${String(d.getUTCDate()).padStart(2, '0')} ${d.toLocaleString('id-ID', { month: 'short', timeZone: 'UTC' })} ${d.getUTCFullYear()}`;
    };
    const pDesc = (() => {
      const s = rangeDateFormat(startDate);
      const e = rangeDateFormat(endDate);
      switch (selectedPeriodType) {
        case 'realtime': return `Real-time (00:00 - ${currentTimeHour})`;
        case 'yesterday': return `Yesterday · ${s}`;
        case '7days': return `7 Days · ${s} - ${e}`;
        case '30days': return `30 Days · ${s} - ${e}`;
        case 'daily': return `Daily · ${s}`;
        case 'weekly': return `Weekly · ${s} - ${e}`;
        case 'monthly': return `Monthly · ${s} - ${e}`;
        case 'yearly': return `Yearly · ${s} - ${e}`;
        default: return `${s} - ${e}`;
      }
    })();

    doc.setFontSize(16);
    doc.text('Revenue Report', margin, yPos);
    yPos += 8;
    doc.setFontSize(10);
    doc.text(`Period: ${pDesc}`, margin, yPos);
    yPos += 6;
    doc.text(`Granularity: ${chartType === 'hourly' ? 'Hourly' : chartType === 'daily' ? 'Daily' : 'Periodic'}`, margin, yPos);
    yPos += 6;

    if (comparisonDateRange) {
      doc.setFontSize(9);
      doc.text(`Comparison: ${statCardLabels.comparisonLabel} · ${comparisonDateRange}`, margin, yPos);
      yPos += 6;
    }

    const fmt = (v) => `Rp ${v.toLocaleString('id-ID')}`;
    const fmtChg = (cur, prev) => {
      if (prev === 0) return cur > 0 ? '+100%' : '0%';
      const chg = (cur - prev) / prev;
      return `${chg >= 0 ? '+' : ''}${(chg * 100).toFixed(1)}%`;
    };
    let summaryBody;
    if (chartType === 'hourly') {
      summaryBody = [
        ['Revenue (RP)', fmt(kpiData.totalRevenue), fmt(kpiData.previousRevenue), fmtChg(kpiData.totalRevenue, kpiData.previousRevenue)],
        ['Orders', kpiData.totalOrders.toLocaleString('id-ID'), kpiData.previousOrders.toLocaleString('id-ID'), fmtChg(kpiData.totalOrders, kpiData.previousOrders)],
        ['Avg Order Value (RP)', fmt(kpiData.avgOrderValue), fmt(kpiData.previousAvgOrderValue), fmtChg(kpiData.avgOrderValue, kpiData.previousAvgOrderValue)],
        ['Peak Revenue Hour (RP)', fmt(kpiData.peakRevenueHour), fmt(kpiData.previousPeakRevenue), fmtChg(kpiData.peakRevenueHour, kpiData.previousPeakRevenue)],
      ];
    } else if (chartType === 'yearly') {
      summaryBody = [
        ['Revenue (RP)', fmt(kpiData.totalRevenue), fmt(kpiData.previousRevenue), fmtChg(kpiData.totalRevenue, kpiData.previousRevenue)],
        ['Orders', kpiData.totalOrders.toLocaleString('id-ID'), kpiData.previousOrders.toLocaleString('id-ID'), fmtChg(kpiData.totalOrders, kpiData.previousOrders)],
        ['Avg Order Value (RP)', fmt(kpiData.avgOrderValue), fmt(kpiData.previousAvgOrderValue), fmtChg(kpiData.avgOrderValue, kpiData.previousAvgOrderValue)],
        ['Peak Revenue Month (RP)', fmt(kpiData.peakRevenueMonth), fmt(kpiData.previousPeakRevenueMonth), fmtChg(kpiData.peakRevenueMonth, kpiData.previousPeakRevenueMonth)],
        ['Avg. Revenue / Month (RP)', fmt(kpiData.revenuePerDay * 30), fmt(kpiData.previousRevenuePerDay * 30), fmtChg(kpiData.revenuePerDay * 30, kpiData.previousRevenuePerDay * 30)],
      ];
    } else {
      summaryBody = [
        ['Revenue (RP)', fmt(kpiData.totalRevenue), fmt(kpiData.previousRevenue), fmtChg(kpiData.totalRevenue, kpiData.previousRevenue)],
        ['Orders', kpiData.totalOrders.toLocaleString('id-ID'), kpiData.previousOrders.toLocaleString('id-ID'), fmtChg(kpiData.totalOrders, kpiData.previousOrders)],
        ['Avg Order Value (RP)', fmt(kpiData.avgOrderValue), fmt(kpiData.previousAvgOrderValue), fmtChg(kpiData.avgOrderValue, kpiData.previousAvgOrderValue)],
        ['Revenue per Day (RP)', fmt(kpiData.revenuePerDay), fmt(kpiData.previousRevenuePerDay), fmtChg(kpiData.revenuePerDay, kpiData.previousRevenuePerDay)],
      ];
    }

    autoTable(doc, {
      startY: yPos + 2,
      head: [['Metric', 'Current Period', 'Previous Period', 'Change']],
      body: summaryBody,
      theme: 'grid',
      styles: { fontSize: 9 },
    });
    yPos = doc.lastAutoTable.finalY + 8;

    if (chartCanvas && chartData.length > 0) {
      const temp = document.createElement('canvas');
      temp.width = chartCanvas.width;
      temp.height = chartCanvas.height;
      const tCtx = temp.getContext('2d');
      tCtx.fillStyle = '#111827';
      tCtx.fillRect(0, 0, temp.width, temp.height);
      tCtx.drawImage(chartCanvas, 0, 0);
      const imgData = temp.toDataURL('image/png');
      const imgWidth = pageWidth - margin * 2;
      const imgHeight = (chartCanvas.height / chartCanvas.width) * imgWidth;
      const maxImgHeight = 90;
      const finalImgHeight = Math.min(imgHeight, maxImgHeight);
      doc.addImage(imgData, 'PNG', margin, yPos, imgWidth, finalImgHeight);
      yPos += finalImgHeight + 8;
    }

    if (bestPeriod) {
      doc.setFontSize(9);
      doc.text(`Best ${bestWorstHeading}: ${getPeriodLabel(bestPeriod)} — Rp ${(bestPeriod.total || 0).toLocaleString('id-ID')}`, margin, yPos);
      yPos += 5;
    }
    if (worstPeriod && worstPeriod.total !== bestPeriod?.total) {
      doc.setFontSize(9);
      doc.text(`Worst ${bestWorstHeading}: ${getPeriodLabel(worstPeriod)} — Rp ${(worstPeriod.total || 0).toLocaleString('id-ID')}`, margin, yPos);
      if (chartType === 'hourly') {
        doc.setFontSize(8);
        doc.setFont('Helvetica', 'italic');
        doc.text('(zero-revenue hours excluded)', margin, yPos + 3);
        doc.setFont('Helvetica', 'normal');
        yPos += 8;
      } else {
        yPos += 5;
      }
    }

    doc.addPage();
    yPos = 20;

    if (sortedRows.length > 0) {
      const hasOrders = sortedRows.some(r => r.orderCount !== null);
      const headers = ['Period', 'Revenue (Rp)', 'Prev Period (Rp)', 'Change %'];
      if (hasOrders) headers.push('Orders');

      const body = sortedRows.map(row => {
        const change = row.prevRevenue > 0 ? (((row.revenue - row.prevRevenue) / row.prevRevenue) * 100) : null;
        const rowData = [
          row.period,
          row.revenue.toLocaleString('id-ID'),
          row.prevRevenue !== null ? row.prevRevenue.toLocaleString('id-ID') : '—',
          change !== null ? `${change >= 0 ? '+' : ''}${change.toFixed(1)}%` : '—',
        ];
        if (hasOrders) rowData.push(row.orderCount !== null ? row.orderCount.toString() : '—');
        return rowData;
      });

      const tRev = sortedRows.reduce((s, r) => s + (r.revenue || 0), 0);
      const tPrev = sortedRows.reduce((s, r) => s + (r.prevRevenue || 0), 0);
      const tChg = tPrev > 0 ? ((tRev - tPrev) / tPrev * 100) : null;
      const totalRow = [
        'TOTAL',
        tRev.toLocaleString('id-ID'),
        tPrev > 0 ? tPrev.toLocaleString('id-ID') : '—',
        tChg !== null ? `${tChg >= 0 ? '+' : ''}${tChg.toFixed(1)}%` : '—',
      ];
      if (hasOrders) {
        totalRow.push(sortedRows.reduce((s, r) => s + (r.orderCount || 0), 0).toString());
      }
      body.push(totalRow);

      autoTable(doc, {
        startY: yPos + 2,
        head: [headers],
        body,
        theme: 'grid',
        styles: { fontSize: 7 },
        headStyles: { fillColor: [124, 58, 237] },
        footStyles: { fillColor: [30, 41, 59] },
      });
    }

    const fileName = `revenue-report-${selectedPeriodType}-${startDate || 'N/A'}-${endDate || 'N/A'}.pdf`;
    doc.save(fileName);

    toast.success('PDF export completed');
  } catch (error) {
    console.error('PDF export error:', error);
    toast.error('Failed to export to PDF');
  }
}
