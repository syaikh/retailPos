export {
  getProducts,
  getProductById,
  createProduct,
  updateProduct,
  deleteProduct,
  bulkUpdateStatus,
  getCategories,
  getBrands,
  getTaxClasses,
  getUnitsOfMeasure,
  getStockThresholds,
  adjustStock,
  getProductsByIds,
  getNextSku,
  getProductOptions,
  getWarehouses,
} from "./services/product-service";
export { statusInfo, formatCurrency, formatDate } from "./lib/product-utils";
export type {
  Product,
  ProductFormData,
  ProductFilters,
  StockThreshold,
  Category,
  Brand,
  TaxClass,
  UnitOfMeasure,
} from "./types";
