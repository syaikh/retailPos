package sale

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
)

func TestWriteCSV(t *testing.T) {
	tests := []struct {
		name     string
		rows     []ExportRow
		wantRows int
	}{
		{
			name:     "empty rows",
			rows:     []ExportRow{},
			wantRows: 1,
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
			wantRows: 2,
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
			wantRows: 3,
		},
		{
			name: "special characters preserved",
			rows: []ExportRow{
				{
					InvoiceNumber: "INV-003",
					CreatedAt:     "2026-01-15",
					CustomerName:  "O'Brien & Co.",
					ItemCount:     2,
					PaymentMethod: "e-wallet",
					TotalAmount:   250000,
				},
			},
			wantRows: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteCSV(tt.rows, &buf)
			assert.NoError(t, err)

			r := csv.NewReader(&buf)
			records, err := r.ReadAll()
			assert.NoError(t, err)
			assert.Equal(t, tt.wantRows, len(records))

			if len(tt.rows) > 0 {
				assert.Equal(t, "Invoice Number", records[0][0])
				assert.Equal(t, "Date", records[0][1])
				assert.Equal(t, "Customer", records[0][2])
				assert.Equal(t, "Items", records[0][3])
				assert.Equal(t, "Payment Method", records[0][4])
				assert.Equal(t, "Total Amount", records[0][5])

				assert.Equal(t, tt.rows[0].InvoiceNumber, records[1][0])
				assert.Equal(t, tt.rows[0].CreatedAt, records[1][1])
				assert.Equal(t, tt.rows[0].CustomerName, records[1][2])
				assert.Equal(t, fmt.Sprintf("%d", tt.rows[0].ItemCount), records[1][3])
				assert.Equal(t, tt.rows[0].PaymentMethod, records[1][4])
				assert.Equal(t, fmt.Sprintf("%d", tt.rows[0].TotalAmount), records[1][5])
			}
		})
	}
}

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

func TestWriteCSV_SpecialCharactersPreserved(t *testing.T) {
	rows := []ExportRow{
		{
			InvoiceNumber: "=SUM(A1:A2)",
			CreatedAt:     "2026-01-15",
			CustomerName:  "+tricky prefix",
			ItemCount:     1,
			PaymentMethod: "-discount",
			TotalAmount:   10000,
		},
	}
	var buf bytes.Buffer
	err := WriteCSV(rows, &buf)
	assert.NoError(t, err)

	r := csv.NewReader(&buf)
	records, err := r.ReadAll()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(records))

	_, _ = io.ReadAll(&buf)
	assert.Contains(t, records[1][0], "SUM")
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
