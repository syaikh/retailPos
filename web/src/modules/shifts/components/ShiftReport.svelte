<script lang="ts">
  import { onMount } from 'svelte';
  import { Button } from '$shared/ui';
  import { getShiftReport } from '../services/shift-service';
  import type { ShiftReportData } from '../types';

  interface Props {
    shiftId: number;
    onClose: () => void;
  }

  let { shiftId, onClose }: Props = $props();
  let report = $state<ShiftReportData | null>(null);
  let loading = $state(true);
  let error = $state('');

  function formatMoney(amount: number) {
    return new Intl.NumberFormat('id-ID', {
      style: 'currency',
      currency: 'IDR',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(amount);
  }

  onMount(async () => {
    try {
      report = await getShiftReport(shiftId);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load report';
    } finally {
      loading = false;
    }
  });

  function handlePrint() {
    window.print();
  }

  function formatDuration(minutes: number): string {
    const h = Math.floor(minutes / 60);
    const m = minutes % 60;
    if (h === 0) return `${m}m`;
    if (m === 0) return `${h}h`;
    return `${h}h ${m}m`;
  }

  function formatDate(iso: string): string {
    if (!iso) return '-';
    return new Date(iso).toLocaleString('id-ID', {
      dateStyle: 'medium',
      timeStyle: 'short',
    });
  }

  function getMethodName(code: string): string {
    const names: Record<string, string> = {
      cash: 'Tunai',
      qris: 'QRIS',
      debit: 'Debit',
      credit: 'Kredit',
      ewallet: 'E-Wallet',
      transfer: 'Transfer',
    };
    return names[code] || code.toUpperCase();
  }
</script>

<div class="shift-report">
  {#if loading}
    <div class="flex items-center justify-center py-12">
      <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
    </div>
  {:else if error}
    <div class="py-12 text-center text-danger">{error}</div>
  {:else if report}
    <!-- Header -->
    <div class="mb-6 border-b pb-4">
      <h2 class="text-xl font-bold text-text-primary">Laporan Sif / Shift Report</h2>
      <p class="text-sm text-text-muted">{report.username} &middot; {report.store_name || '-'}</p>
      <p class="text-xs text-text-muted">Dibuat: {formatDate(new Date().toISOString())}</p>
    </div>

    <!-- Summary -->
    <div class="mb-6 grid grid-cols-2 gap-4">
      <div>
        <p class="text-xs text-text-muted">Status</p>
        <p class="font-medium text-text-primary">{report.status === 'open' ? 'Aktif' : 'Tutup'}</p>
      </div>
      <div>
        <p class="text-xs text-text-muted">Durasi</p>
        <p class="font-medium text-text-primary">{formatDuration(report.duration_minutes)}</p>
      </div>
      <div>
        <p class="text-xs text-text-muted">Dibuka</p>
        <p class="font-medium text-text-primary">{formatDate(report.opened_at)}</p>
      </div>
      <div>
        <p class="text-xs text-text-muted">Ditutup</p>
        <p class="font-medium text-text-primary">{report.closed_at ? formatDate(report.closed_at) : '-'}</p>
      </div>
    </div>

    <!-- Sales Summary -->
    <div class="mb-6">
      <h3 class="mb-2 text-sm font-semibold text-text-primary">Ringkasan Penjualan</h3>
      <div class="grid grid-cols-2 gap-3 text-sm">
        <div class="flex justify-between">
          <span class="text-text-muted">Kas Awal</span>
          <span class="font-medium text-text-primary">{formatMoney(report.opening_balance)}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-text-muted">Penjualan Tunai</span>
          <span class="font-medium text-text-primary">{formatMoney(report.cash_sales)}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-text-muted">Penjualan Non-Tunai</span>
          <span class="font-medium text-text-primary">{formatMoney(report.non_cash_sales)}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-text-muted">Total Penjualan</span>
          <span class="font-medium text-text-primary">{formatMoney(report.total_sales)}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-text-muted">Jumlah Transaksi</span>
          <span class="font-medium text-text-primary">{report.transaction_count}</span>
        </div>
      </div>
    </div>

    <!-- Payment Breakdown -->
    {#if report.payment_breakdown && report.payment_breakdown.length > 0}
      <div class="mb-6">
        <h3 class="mb-2 text-sm font-semibold text-text-primary">Rincian Pembayaran</h3>
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b text-left text-text-muted">
              <th class="pb-1 font-medium">Metode</th>
              <th class="pb-1 text-right font-medium">Jumlah</th>
              <th class="pb-1 text-right font-medium">Transaksi</th>
            </tr>
          </thead>
          <tbody>
            {#each report.payment_breakdown as pm}
              <tr class="border-b border-dashed">
                <td class="py-1">{getMethodName(pm.method)}</td>
                <td class="py-1 text-right">{formatMoney(pm.amount)}</td>
                <td class="py-1 text-right">{pm.count}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}

    <!-- Cash Reconciliation -->
    <div class="mb-6">
      <h3 class="mb-2 text-sm font-semibold text-text-primary">Rekonsiliasi Kas</h3>
      <div class="space-y-1 text-sm">
        <div class="flex justify-between">
          <span class="text-text-muted">Kas Awal</span>
          <span class="text-text-primary">{formatMoney(report.opening_balance)}</span>
        </div>
        <div class="flex justify-between">
          <span class="text-text-muted">Penjualan Tunai</span>
          <span class="text-text-primary">{formatMoney(report.cash_sales)}</span>
        </div>
        {#if report.cash_movement_summary}
          <div class="flex justify-between">
            <span class="text-text-muted">Setor Kas</span>
            <span class="text-danger">-{formatMoney(report.cash_movement_summary.cash_drops)}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-text-muted">Uang Masuk</span>
            <span class="text-success">+{formatMoney(report.cash_movement_summary.paid_ins)}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-text-muted">Uang Keluar</span>
            <span class="text-danger">-{formatMoney(report.cash_movement_summary.paid_outs)}</span>
          </div>
        {/if}
        <div class="flex justify-between border-t pt-1 font-medium">
          <span class="text-text-primary">Kas yang Diharapkan</span>
          <span class="text-primary">{formatMoney(report.opening_balance + report.cash_sales - (report.cash_movement_summary?.cash_drops || 0) + (report.cash_movement_summary?.paid_ins || 0) - (report.cash_movement_summary?.paid_outs || 0))}</span>
        </div>
        {#if report.closing_balance !== null}
          <div class="flex justify-between">
            <span class="text-text-primary">Kas Aktual</span>
            <span class="text-text-primary">{formatMoney(report.closing_balance)}</span>
          </div>
          <div class="flex justify-between font-medium">
            <span class="text-text-primary">Selisih</span>
            <span class={report.discrepancy === 0 ? 'text-success' : 'text-danger'}>
              {report.discrepancy === 0 ? 'Seimbang' : formatMoney(report.discrepancy || 0)}
            </span>
          </div>
        {/if}
      </div>
    </div>

    <!-- Actions -->
    <div class="flex justify-end gap-2 print:hidden">
      <Button variant="secondary" onclick={onClose}>Tutup</Button>
      <Button variant="primary" onclick={handlePrint}>Cetak Laporan</Button>
    </div>
  {/if}
</div>

<style>
  @media print {
    .shift-report :global(.print\\:hidden) {
      display: none !important;
    }
    .shift-report {
      padding: 0;
    }
  }
</style>
