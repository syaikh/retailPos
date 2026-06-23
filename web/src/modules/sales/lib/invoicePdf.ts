interface InvoiceTransaction {
  invoice_number: string;
  created_at: string;
  payment_method?: string;
  customer_name?: string;
  total_amount?: number;
  tax?: number;
  items?: Array<{
    name: string;
    quantity: number;
    unit_price: number;
  }>;
}

export async function downloadInvoice(
  transaction: InvoiceTransaction,
  formatDate: (date: Date) => string,
): Promise<boolean> {
  if (!transaction) return false;
  try {
    const { jsPDF } = await import('jspdf');
    const { default: autoTable } = await import('jspdf-autotable');

    const doc = new jsPDF();
    doc.setFontSize(18);
    doc.text('INVOICE', 20, 20);
    doc.setFontSize(10);
    doc.text(`Invoice #: ${transaction.invoice_number}`, 20, 30);
    doc.text(`Date: ${formatDate(new Date(transaction.created_at))}`, 20, 36);
    doc.text(`Payment: ${transaction.payment_method || '—'}`, 20, 42);
    if (transaction.customer_name) {
      doc.text(`Customer: ${transaction.customer_name}`, 20, 48);
    }

    const itemRows = (transaction.items || []).map((item) => [
      item.name,
      item.quantity.toString(),
      `Rp ${(item.unit_price || 0).toLocaleString('id-ID')}`,
      `Rp ${(item.unit_price * item.quantity).toLocaleString('id-ID')}`,
    ]);

    autoTable(doc, {
      startY: 58,
      head: [['Item', 'Qty', 'Price', 'Subtotal']],
      body: itemRows,
      theme: 'grid',
      styles: { fontSize: 9 },
      headStyles: { fillColor: [124, 58, 237] },
    });

    const finalY = doc.lastAutoTable.finalY + 10;
    const taxAmt = transaction.tax || 0;
    if (taxAmt > 0) {
      doc.setFontSize(10);
      doc.text(`Subtotal (DPP): Rp ${((transaction.total_amount || 0) - taxAmt).toLocaleString('id-ID')}`, 20, finalY);
      doc.text(`PPN 11%: Rp ${taxAmt.toLocaleString('id-ID')}`, 20, finalY + 6);
      doc.setFontSize(12);
      doc.text(`Total: Rp ${(transaction.total_amount || 0).toLocaleString('id-ID')}`, 20, finalY + 14);
    } else {
      doc.setFontSize(12);
      doc.text(`Total: Rp ${(transaction.total_amount || 0).toLocaleString('id-ID')}`, 20, finalY);
    }

    doc.save(`invoice-${transaction.invoice_number}.pdf`);
    return true;
  } catch {
    return false;
  }
}
