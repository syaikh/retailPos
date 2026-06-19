import { describe, it, expect, beforeAll } from 'vitest';
import fs from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

function getSource(): string {
  const dir = path.dirname(fileURLToPath(import.meta.url));
  return fs.readFileSync(path.join(dir, 'ReportsPage.svelte'), 'utf-8');
}

describe('ReportsPage.svelte — export features', () => {
  let src: string;

  beforeAll(() => {
    src = getSource();
  });

  // ── Export function existence ───────────────────────────────────────────────
  it('declares exportToExcel function', () => {
    expect(src).toContain('async function exportToExcel()');
  });

  it('declares exportToPDF function', () => {
    expect(src).toContain('async function exportToPDF()');
  });

  // ── Dynamic imports ─────────────────────────────────────────────────────────
  it('dynamically imports xlsx in exportToExcel', () => {
    expect(src).toContain("await import('xlsx')");
  });

  it('dynamically imports jsPDF in exportToPDF', () => {
    expect(src).toContain("await import('jspdf')");
  });

  it('dynamically imports jspdf-autotable in exportToPDF', () => {
    expect(src).toContain("await import('jspdf-autotable')");
  });

  // ── Canvas ref (chart capture) ──────────────────────────────────────────────
  it('declares chartCanvas state for canvas ref', () => {
    expect(src).toContain('let chartCanvas = $state()');
  });

  it('binds chartCanvas ref to the canvas element', () => {
    expect(src).toContain('bind:this={chartCanvas} use:chart={chartConfig}');
  });

  it('uses toDataURL to capture chart image in PDF export', () => {
    expect(src).toContain("chartCanvas.toDataURL('image/png')");
  });

  it('uses doc.addImage to embed chart image in PDF', () => {
    expect(src).toContain("doc.addImage(imgData, 'PNG', margin, yPos, imgWidth, finalImgHeight)");
  });

  // ── Data table in exports (sortedRows based) ────────────────────────────────
  it('uses sortedRows for Excel data rows', () => {
    expect(src).toContain('sortedRows.map(row =>');
  });

  it('uses sortedRows for PDF data rows', () => {
    const matches = src.match(/sortedRows\.map\(row =>/g);
    expect(matches).not.toBeNull();
    expect(matches!.length).toBeGreaterThanOrEqual(2);
  });

  it('exports Period column header', () => {
    expect(src).toContain("'Period'");
  });

  it('exports Revenue column header in Excel', () => {
    expect(src).toContain("'Revenue'");
  });

  it('exports Revenue (Rp) column header in PDF', () => {
    expect(src).toContain("'Revenue (Rp)'");
  });

  it('exports Prev Period column headers', () => {
    expect(src).toContain("'Prev Period'");
    expect(src).toContain("'Prev Period (Rp)'");
  });

  it('exports Change % column header', () => {
    expect(src).toContain("'Change %'");
  });

  it('conditionally includes Orders column based on hasOrders', () => {
    expect(src).toContain('hasOrders) headers.push');
  });

  // ── Total row ───────────────────────────────────────────────────────────────
  it('computes total revenue via reduce for Excel total row', () => {
    expect(src).toContain("sortedRows.reduce((s, r) => s + (r.revenue || 0), 0)");
  });

  it('computes total revenue via reduce in both exports and footer', () => {
    const matches = src.match(/sortedRows\.reduce\(\(s, r\) => s \+ \(r\.revenue \|\| 0\), 0\)/g);
    expect(matches).not.toBeNull();
    expect(matches!.length).toBeGreaterThanOrEqual(2);
  });

  it('writes TOTAL row in Excel data', () => {
    expect(src).toContain("'TOTAL'");
  });

  // ── Best/worst period in exports ────────────────────────────────────────────
  it('includes bestPeriod info in Excel summary', () => {
    expect(src).toContain('Best Period');
  });

  it('includes worstPeriod info in Excel summary', () => {
    expect(src).toContain('Worst Period');
  });

  it('includes bestPeriod info in PDF', () => {
    expect(src).toContain('Best ${bestWorstHeading}');
  });

  it('includes worstPeriod info in PDF', () => {
    expect(src).toContain('Worst ${bestWorstHeading}');
  });

  // ── Summary in exports ──────────────────────────────────────────────────────
  it('includes period description in Excel summary', () => {
    expect(src).toContain('getPeriodDescription()');
  });

  it('includes comparison info in Excel summary', () => {
    expect(src).toContain('comparisonDateRange');
  });

  it('includes comparison info in PDF', () => {
    expect(src).toContain('comparisonDateRange');
  });

  it('includes granularity in both export summaries', () => {
    expect(src).toContain("'Granularity'");
  });

  // ── File naming ─────────────────────────────────────────────────────────────
  it('generates .xlsx file name with period type and dates', () => {
    expect(src).toContain('.xlsx');
  });

  it('generates .pdf file name with period type and dates', () => {
    expect(src).toContain('.pdf');
  });

  // ── Success / error reporting ───────────────────────────────────────────────
  it('shows success toast after Excel export', () => {
    expect(src).toContain("toast.success('Excel export completed')");
  });

  it('shows success toast after PDF export', () => {
    expect(src).toContain("toast.success('PDF export completed')");
  });

  it('shows error toast if Excel export fails', () => {
    expect(src).toContain("toast.error('Export Excel gagal:");
  });

  it('shows error toast if PDF export fails', () => {
    expect(src).toContain("toast.error('Failed to export to PDF')");
  });

  // ── UI: Export dropdown ─────────────────────────────────────────────────────
  it('has Export button in the UI', () => {
    expect(src).toContain('btn btn-primary');
    expect(src).toContain('Export\n');
  });

  it('has Export to Excel menu item', () => {
    expect(src).toContain('Export to Excel');
  });

  it('has Export to PDF menu item', () => {
    expect(src).toContain('Export to PDF');
  });

  it('uses showExportDropdown state', () => {
    expect(src).toContain('let showExportDropdown');
  });

  it('uses FileSpreadsheet icon for Excel', () => {
    expect(src).toContain('FileSpreadsheet');
  });

  it('has Export button with Download icon', () => {
    expect(src).toContain('<Download size={15} />');
  });
});
