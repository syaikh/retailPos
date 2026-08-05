package purchase

import "context"

type ProductInfo struct {
	Name string
	SKU  string
}

type ProductLookup interface {
	GetProductNamesByIDs(ctx context.Context, ids []int) (map[int]ProductInfo, error)
}

type SupplierInfo struct {
	Name string
}

type SupplierLookup interface {
	GetSupplierNamesByIDs(ctx context.Context, ids []int) (map[int]SupplierInfo, error)
	GetSupplierIDsByName(ctx context.Context, name string) ([]int, error)
}
