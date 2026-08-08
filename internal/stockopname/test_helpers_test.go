package stockopname

import (
	"context"

	"retail-pos-system/internal/brand"
	"retail-pos-system/internal/category"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/store"
	"retail-pos-system/internal/storagelocation"
	"retail-pos-system/internal/supplier"
	"retail-pos-system/internal/uom"
	"retail-pos-system/internal/user"
)

// testScopeNameResolver mirrors the production scopeNameResolverAdapter from
// internal/wiring so stockopname tests exercise the same owner-provider reads.
type testScopeNameResolver struct{}

func (testScopeNameResolver) ScopeNames(ctx context.Context, db shared.DBPool, refs []ScopeRef) (map[ScopeRef]string, error) {
	names := make(map[ScopeRef]string, len(refs))
	if len(refs) == 0 {
		return names, nil
	}
	grouped := make(map[string][]int)
	for _, ref := range refs {
		if ref.ScopeID <= 0 {
			continue
		}
		grouped[ref.ScopeType] = append(grouped[ref.ScopeType], int(ref.ScopeID))
	}
	for scopeType, ids := range grouped {
		var byID map[int]string
		var err error
		switch scopeType {
		case "store":
			byID, err = (store.StoreNamesProvider{}).StoreNamesByIDs(ctx, db, ids)
		case "warehouse":
			byID, err = (store.WarehouseNamesProvider{}).WarehouseNamesByIDs(ctx, db, ids)
		case "category":
			byID, err = (category.CategoryNamesProvider{}).CategoryNamesByIDs(ctx, db, ids)
		case "brand":
			byID, err = (brand.BrandNamesProvider{}).BrandNamesByIDs(ctx, db, ids)
		case "supplier":
			byID, err = (supplier.SupplierNamesProvider{}).SupplierNamesByIDs(ctx, db, ids)
		case "product":
			byID, err = (product.ProductNameLookup{}).ProductNamesByIDs(ctx, db, ids)
		case "location":
			racks, rerr := (storagelocation.RackProvider{}).RacksByIDs(ctx, db, ids)
			if rerr != nil {
				return nil, rerr
			}
			byID = make(map[int]string, len(racks))
			for _, rack := range racks {
				byID[rack.ID] = rack.Name
			}
		default:
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			names[ScopeRef{ScopeType: scopeType, ScopeID: int64(id)}] = byID[id]
		}
	}
	return names, nil
}

// newTestRepository returns a Repository wired with the user-owned read ports
// and the scope-name/location/warehouse ports, mirroring the internal/wiring
// composition. Tests exercise the same provider implementations that run in
// production.
func newTestRepository() *Repository {
	repo := NewRepository(dbPool)
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetAssignableUserProvider(user.AssignableUsersProvider{})
	repo.SetUserRoleNameProvider(user.RoleNameProvider{})
	repo.SetScopeNameResolver(testScopeNameResolver{})
	repo.SetLocationScopeProvider(storagelocation.RackProvider{})
	repo.SetWarehouseStoreIDProvider(store.WarehouseStoreIDProvider{})
	repo.SetStockLocker(inventory.StockLocker{})
	repo.SetProductCatalogProvider(product.ProductMetaLookup{})
	repo.SetProductScopeProvider(product.ProductMetaLookup{})
	repo.SetUOMNameProvider(uom.UnitNameLookup{})
	repo.SetStockSnapshotProvider(inventory.StockSnapshotProvider{})
	return repo
}
