import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import { setLocale } from '$shared/i18n';

const calls: any[] = [];
const mockGetPurchaseOrders = vi.fn();

vi.mock('$modules/purchase-orders/services/po-service', async (importOriginal) => {
  const orig = await importOriginal<object>();
  return {
    ...orig,
    getPurchaseOrders: (...args: unknown[]) => mockGetPurchaseOrders(...args),
    getSuppliers: vi.fn().mockResolvedValue({ data: [], total: 0 }),
  };
});

vi.mock('$modules/auth', async (importOriginal) => {
  const orig = await importOriginal<object>();
  return {
    ...orig,
    useAuthStore: () => ({ user: { permissions: ['purchase_order.view'] } }),
  };
});

vi.mock('$shared/api/websocket', () => ({
  useWebSocket: () => ({ on: () => () => {} }),
}));

vi.mock('$modules/purchase-orders/components/PurchaseOrderForm.svelte', async () => {
  const m = await import('./stubs/Empty.svelte');
  return { default: m.default };
});
vi.mock('$modules/purchase-orders/components/PurchaseOrderDetail.svelte', async () => {
  const m = await import('./stubs/Empty.svelte');
  return { default: m.default };
});
vi.mock('$modules/purchase-orders/components/GoodsReceiptModal.svelte', async () => {
  const m = await import('./stubs/Empty.svelte');
  return { default: m.default };
});

import PurchaseOrdersPage from '$modules/purchase-orders/components/PurchaseOrdersPage.svelte';

function makePOs(page: number, pageSize: number) {
  return Array.from({ length: pageSize }, (_, i) => {
    const n = page * pageSize + i + 1;
    return {
      id: n, po_number: `PO-${n}`, supplier_id: 1, store_id: 1, status: 'draft',
      grand_total: 1000, created_at: new Date().toISOString(), updated_at: new Date().toISOString(), items: [],
    };
  });
}

const nextBtn = (c: HTMLElement) => c.querySelector('button[aria-label="Next page"]') as HTMLButtonElement;
const pageLabel = (c: HTMLElement) => {
  const btns = Array.from(c.querySelectorAll('button'));
  const b = btns.find((x) => /Page \d+ of \d+/.test(x.textContent || ''));
  return b?.textContent?.match(/Page (\d+) of (\d+)/);
};

describe('PO pagination sequence', () => {
  beforeEach(() => {
    setLocale('en');
    calls.length = 0;
    mockGetPurchaseOrders.mockReset();
    mockGetPurchaseOrders.mockImplementation(async (filters: any) => {
      calls.push({
        page: filters.page ?? 0,
        pageSize: filters.pageSize ?? 20,
        search: filters.search ?? null,
        status: filters.status ?? null,
        supplier: filters.supplier_id ?? null,
        start: filters.startDate ?? null,
        end: filters.endDate ?? null,
      });
      return { data: makePOs(filters.page ?? 0, filters.pageSize ?? 20), total: 45 };
    });
  });

  it('clicking next advances to page 2 and does not snap back to page 1', async () => {
    const { container } = render(PurchaseOrdersPage);

    await waitFor(() => {
      expect(nextBtn(container)).toBeTruthy();
      expect(pageLabel(container)?.[1]).toBe('1');
    });

    await fireEvent.click(nextBtn(container));
    await new Promise((r) => setTimeout(r, 600));

    const afterClick1 = calls.map((c) => c.page);
    const label1 = pageLabel(container);

    await fireEvent.click(nextBtn(container));
    await new Promise((r) => setTimeout(r, 600));

    const afterClick2 = calls.map((c) => c.page);
    const label2 = pageLabel(container);

    expect({
      afterClick1,
      callsDetail: calls,
      labelAfterClick1: label1?.[1],
      afterClick2,
      labelAfterClick2: label2?.[1],
    }).toEqual({
      afterClick1: [0, 1],
      callsDetail: calls,
      labelAfterClick1: '2',
      afterClick2: [0, 1, 2],
      labelAfterClick2: '3',
    });
  });
});
