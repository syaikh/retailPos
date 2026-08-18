package sale

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
)

func TestWriteXLSX(t *testing.T) {
	tests := []struct {
		name string
		rows []ExportRow
	}{
		{
			name: "empty rows",
			rows: []ExportRow{},
		},
		{
			name: "single row",
			rows: []ExportRow{
				{
					InvoiceNumber: "INV-001",
					CreatedAt:     "2026-01-15 10:30:00",
					CustomerName:  "John",
					ItemCount:     3,
					PaymentMethod: "cash",
					TotalAmount:   150000,
				},
			},
		},
		{
			name: "multiple rows",
			rows: []ExportRow{
				{
					InvoiceNumber: "INV-001",
					CreatedAt:     "2026-01-15 10:30:00",
					CustomerName:  "John",
					ItemCount:     3,
					PaymentMethod: "cash",
					TotalAmount:   150000,
				},
				{
					InvoiceNumber: "INV-002",
					CreatedAt:     "2026-01-15 11:00:00",
					CustomerName:  "Jane",
					ItemCount:     1,
					PaymentMethod: "card",
					TotalAmount:   50000,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteXLSX(tt.rows, &buf)
			assert.NoError(t, err)
			assert.Greater(t, buf.Len(), 0)

			f, err := excelize.OpenReader(&buf)
			assert.NoError(t, err)
			defer func() { _ = f.Close() }()

			sheet := f.GetSheetName(0)
			assert.NotEmpty(t, sheet)

			header, err := f.GetCellValue(sheet, "A1")
			assert.NoError(t, err)
			assert.Equal(t, "Invoice Number", header)

			if len(tt.rows) > 0 {
				val, err := f.GetCellValue(sheet, "A2")
				assert.NoError(t, err)
				assert.Equal(t, tt.rows[0].InvoiceNumber, val)
			}
		})
	}
}

func TestWriteXLSX_CellValues(t *testing.T) {
	rows := []ExportRow{
		{
			InvoiceNumber: "INV-010",
			CreatedAt:     "2026-07-13 14:00:00",
			CustomerName:  "Bob",
			ItemCount:     5,
			PaymentMethod: "transfer",
			TotalAmount:   500000,
		},
	}
	var buf bytes.Buffer
	err := WriteXLSX(rows, &buf)
	assert.NoError(t, err)

	f, err := excelize.OpenReader(&buf)
	assert.NoError(t, err)
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(0)

	type xlsxCase struct {
		cell string
		want string
	}
	cases := []xlsxCase{
		{"A1", "Invoice Number"},
		{"B1", "Date"},
		{"C1", "Customer"},
		{"E1", "Payment Method"},
		{"A2", "INV-010"},
		{"B2", "2026-07-13 14:00:00"},
		{"C2", "Bob"},
		{"E2", "transfer"},
	}
	for _, c := range cases {
		val, err := f.GetCellValue(sheet, c.cell)
		assert.NoError(t, err)
		assert.Equal(t, c.want, val)
	}

	dVal, err := f.GetCellValue(sheet, "D2")
	assert.NoError(t, err)
	assert.Equal(t, "5", dVal)

	fVal, err := f.GetCellValue(sheet, "F2")
	assert.NoError(t, err)
	assert.Equal(t, "500000", fVal)
}
