# Bulk Actions Feasibility Analysis

## Strong Candidates

| Page | Entity | Proposed Bulk Actions | Reasoning |
|---|---|---|---|
| **ProductsPage** | Products | • Delete selected<br>• Adjust stock (add/subtract)<br>• Toggle active/inactive<br>• Change category | Mutable, has multiple actionable states, permission-gated |
| **UsersPage** | Users | • Delete selected<br>• Assign role<br>• Activate/Deactivate | Mutable, superadmin/admin can manage, role filter exists |
| **CustomersPage** | Customers | • Deactivate/Activate selected<br>• Delete selected<br>• Export selected | Soft-delete pattern makes bulk status toggle natural |

## Moderate Candidates

| Page | Entity | Proposed Bulk Actions | Reasoning |
|---|---|---|---|
| **CategoriesPage** | Categories | • Delete selected (if `product_count === 0`)<br>• Export selected | Delete is conditional, fewer items |
| **TransactionsPage** | Transactions | • Export selected as CSV/Excel<br>• Print selected receipts | Read-only — only export/print actions make sense |

## Not Suitable

| Page | Reason |
|---|---|
| **RolesPage** | Few items (5-10), complex multi-step permissions, system roles protected |
| **AuditLogsPage** | Read-only audit trail |
| **ReportsPage** | Analytical/reporting, no entity to act on |
| **PosPage** | Interactive cart, not a management table |
