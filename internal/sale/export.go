package sale

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/xuri/excelize/v2"

	"retail-pos-system/internal/shared"
)

func WriteCSV(rows []SaleExportRow, w io.Writer) error {
	cw := csv.NewWriter(w)
	shared.WriteCSVRow(cw, []string{"Invoice Number", "Date", "Customer", "Items", "Payment Method", "Total Amount"})
	for _, row := range rows {
		shared.WriteCSVRow(cw, []string{
			row.InvoiceNumber,
			row.CreatedAt,
			row.CustomerName,
			strconv.Itoa(row.ItemCount),
			row.PaymentMethod,
			fmt.Sprintf("%d", row.TotalAmount),
		})
	}
	cw.Flush()
	return cw.Error()
}

func WriteXLSX(rows []SaleExportRow, w io.Writer) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := f.GetSheetName(0)
	f.SetCellValue(sheet, "A1", "Invoice Number")
	f.SetCellValue(sheet, "B1", "Date")
	f.SetCellValue(sheet, "C1", "Customer")
	f.SetCellValue(sheet, "D1", "Items")
	f.SetCellValue(sheet, "E1", "Payment Method")
	f.SetCellValue(sheet, "F1", "Total Amount")

	for i, row := range rows {
		r := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", r), row.InvoiceNumber)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", r), row.CreatedAt)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", r), row.CustomerName)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", r), row.ItemCount)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", r), row.PaymentMethod)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", r), row.TotalAmount)
	}

	return f.Write(w)
}
