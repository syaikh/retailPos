package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"

	"retailPos/internal/repo"
)

type InventoryService struct {
	productRepo *repo.ProductRepo
}

func NewInventoryService(productRepo *repo.ProductRepo) *InventoryService {
	return &InventoryService{
		productRepo: productRepo,
	}
}

func (s *InventoryService) ExportInventoryCSV(ctx context.Context, w io.Writer) error {
	products, err := s.productRepo.GetAllForExport(ctx)
	if err != nil {
		return err
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"product_id", "nama produk", "sku", "barcode", "stock", "harga", "tanggal dibuat"})

	for p := range products {
		barcode := ""
		if p.Barcode != nil {
			barcode = *p.Barcode
		}
		record := []string{
			fmt.Sprintf("%d", p.ID),
			p.Name,
			p.SKU,
			barcode,
			fmt.Sprintf("%d", p.Stock),
			fmt.Sprintf("%d", p.Price),
			p.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
		// Flush each row to ensure CSV is written
		writer.Flush()
	}

	return nil
}

func (s *InventoryService) ExportInventoryExcel(ctx context.Context) (io.Reader, error) {
	products, err := s.productRepo.GetAllForExport(ctx)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheet := "Inventory"
	f.NewSheet(sheet)
	// Delete default Sheet1
	f.DeleteSheet("Sheet1")
	// Set Inventory as active sheet (index 0)
	// f.SetActiveSheet(0)

	headers := []string{"product_id", "nama produk", "sku", "barcode", "stock", "harga", "tanggal dibuat"}
	for i, h := range headers {
		f.SetCellValue(sheet, cellAt(i+1, 1), h)
	}

	rowNum := 2
	for p := range products {
		barcode := ""
		if p.Barcode != nil {
			barcode = *p.Barcode
		}
		f.SetCellValue(sheet, cellAt(1, rowNum), p.ID)
		f.SetCellValue(sheet, cellAt(2, rowNum), p.Name)
		f.SetCellValue(sheet, cellAt(3, rowNum), p.SKU)
		f.SetCellValue(sheet, cellAt(4, rowNum), barcode)
		f.SetCellValue(sheet, cellAt(5, rowNum), p.Stock)
		f.SetCellValue(sheet, cellAt(6, rowNum), p.Price)
		f.SetCellValue(sheet, cellAt(7, rowNum), p.CreatedAt.Format("2006-01-02 15:04:05"))
		rowNum++
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}

	return &buf, nil
}

func cellAt(col, row int) string {
	return fmt.Sprintf("%s%d", string(rune('A'+col-1)), row)
}
