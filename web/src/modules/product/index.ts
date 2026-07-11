export {
  getProducts, getProductById, createProduct, updateProduct, deleteProduct,
  bulkUpdateStatus, getCategories, getBrands, getTaxClasses, getUnitsOfMeasure,
  getStockThresholds, adjustStock,
} from './services/product-service';
export { statusInfo, formatCurrency, formatDate, validateProductForm, getUserRoleName, hasPermission, buildProductPayload } from './lib/product-utils';
export type { Product, ProductFormData, ProductFilters, StockThreshold, Category, Brand, TaxClass, UnitOfMeasure } from './types';
