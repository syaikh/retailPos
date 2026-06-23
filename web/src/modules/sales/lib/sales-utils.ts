export function statusVariant(s: string): string {
  return s === 'completed' ? 'success' : s === 'refunded' ? 'danger' : 'warning';
}

export function getPaymentMethodVariant(method = ''): string {
  if (!method) return 'muted';
  const m = method.toLowerCase();
  if (m === 'cash') return 'success';
  if (m === 'qris' || m === 'e_wallet') return 'default';
  if (m === 'card') return 'primary';
  if (m === 'transfer') return 'muted';
  return 'muted';
}

export function sanitizeSearch(q: string): string {
  let s = q.trim();
  if (/^INV-/i.test(s)) s = s.slice(4).trim();
  return s;
}
