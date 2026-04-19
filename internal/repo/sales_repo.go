package repo

import (
	"database/sql"
	"fmt"
	model "retailPos/internal/model"
	"strconv"
	"strings"
)

type SalesRepo struct {
	db *sql.DB
}

func NewSalesRepo(db *sql.DB) *SalesRepo {
	return &SalesRepo{db: db}
}

func (r *SalesRepo) GetAll(limit, offset int, search string, sortBy, sortDir string, startDate, endDate string) ([]model.Sale, int, error) {
	fmt.Printf("SalesRepo.GetAll: startDate=%s, endDate=%s, limit=%d, offset=%d\n", startDate, endDate, limit, offset)

	// Base query with join only if searching by item name
	query := `SELECT DISTINCT s.id, s.total_amount, s.payment_method, s.cashier_id, s.created_at 
	          FROM sales s 
	          LEFT JOIN sale_items si ON s.id = si.sale_id 
	          WHERE 1=1`
	countQuery := `SELECT COUNT(DISTINCT s.id) 
	               FROM sales s 
	               LEFT JOIN sale_items si ON s.id = si.sale_id 
	               WHERE 1=1`
	args := []any{}
	placeholderIdx := 1

	// Add date filtering if provided
	if startDate != "" && endDate != "" {
		dateFilter := fmt.Sprintf(" AND s.created_at::date >= $%d AND s.created_at::date <= $%d", placeholderIdx, placeholderIdx+1)
		query += dateFilter
		countQuery += dateFilter
		args = append(args, startDate, endDate)
		placeholderIdx += 2
	}

	if search != "" {
		// handle #TRX-, TRX-, and padding
		cleanSearch := strings.TrimPrefix(strings.TrimPrefix(strings.ToUpper(search), "#TRX-"), "TRX-")

		filter := " AND (si.product_name ILIKE $" + strconv.Itoa(placeholderIdx) + " OR s.id::text ILIKE $" + strconv.Itoa(placeholderIdx)

		searchInt, err := strconv.Atoi(cleanSearch)
		if err == nil {
			// If numeric, add exact ID match to handle cases like "0001" searching for ID 1
			filter += " OR s.id = $" + strconv.Itoa(placeholderIdx+1) + ")"
			args = append(args, "%"+cleanSearch+"%", searchInt)
			placeholderIdx += 2
		} else {
			filter += ")"
			args = append(args, "%"+cleanSearch+"%")
			placeholderIdx++
		}
		query += filter
		countQuery += filter
	}

	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	validSortFields := map[string]string{
		"id":         "s.id",
		"created_at": "s.created_at",
		"total":      "s.total_amount",
	}
	sortField, ok := validSortFields[sortBy]
	if !ok {
		sortField = "s.created_at"
	}
	if strings.ToLower(sortDir) != "asc" {
		sortDir = "DESC"
	} else {
		sortDir = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortField, sortDir)
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sales []model.Sale
	var saleIds []int
	for rows.Next() {
		var s model.Sale
		if err := rows.Scan(&s.ID, &s.TotalAmount, &s.PaymentMethod, &s.CashierID, &s.CreatedAt); err != nil {
			return nil, 0, err
		}
		sales = append(sales, s)
		saleIds = append(saleIds, s.ID)
	}

	if len(sales) == 0 {
		return sales, total, nil
	}

	// Fetch items ONLY for the sales on current page (Optimization)
	idsStr := []string{}
	for _, id := range saleIds {
		idsStr = append(idsStr, strconv.Itoa(id))
	}
	queryItems := fmt.Sprintf(`SELECT id, sale_id, product_id, product_name, quantity, price_at_sale FROM sale_items WHERE sale_id IN (%s) ORDER BY id ASC`, strings.Join(idsStr, ","))

	itemRows, err := r.db.Query(queryItems)
	if err != nil {
		return nil, 0, err
	}
	defer itemRows.Close()

	itemMap := make(map[int][]model.SaleItem)
	for itemRows.Next() {
		var i model.SaleItem
		if err := itemRows.Scan(&i.ID, &i.SaleID, &i.ProductID, &i.ProductName, &i.Quantity, &i.PriceAtSale); err != nil {
			return nil, 0, err
		}
		itemMap[i.SaleID] = append(itemMap[i.SaleID], i)
	}

	for i := range sales {
		sales[i].Items = itemMap[sales[i].ID]
		if sales[i].Items == nil {
			sales[i].Items = []model.SaleItem{}
		}
	}

	return sales, total, nil
}
