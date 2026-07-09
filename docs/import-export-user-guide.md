# Import / Export User Guide

This guide explains how to import and export data (products, categories, brands, units of measure, and customers) in the POS system.

## Table of Contents

- [Exporting Data](#exporting-data)
- [Importing Data](#importing-data)
  - [Step 1: Download a Template](#step-1-download-a-template)
  - [Step 2: Fill Out the Template](#step-2-fill-out-the-template)
  - [Step 3: Upload and Preview](#step-3-upload-and-preview)
  - [Step 4: Confirm the Import](#step-4-confirm-the-import)
- [Understanding the Preview](#understanding-the-preview)
- [Tracking Import Progress](#tracking-import-progress)
- [Cancelling an Import](#cancelling-an-import)
- [Troubleshooting](#troubleshooting)

---

## Exporting Data

To download your existing data:

1. Navigate to the page for the data you want to export (e.g., **Products**, **Categories**, **Brands**, **Units of Measure**, or **Customers**).
2. Click the **Bulk Actions** button near the top of the page, then choose **Export CSV** or **Export XLSX**.
3. The file will download automatically.

> **Tip**: Use CSV if you plan to edit in a spreadsheet program like Excel or Google Sheets. Use XLSX if you want formatting preserved.

---

## Importing Data

Importing lets you add or update many records at once by uploading a file.

### Step 1: Download a Template

1. On the page you want to import into (e.g., **Products**), click the **Bulk Actions** button then **Download Template**.
2. Open the downloaded `.xlsx` file.

The template contains:
- An **Instructions** sheet explaining each column with a reference table.
- A **Data** sheet with column headers where you fill in your records.
- A hidden **Meta** sheet containing template metadata (module, schema version, generation timestamp).
- **Reference sheets** (hidden) that list valid values for dropdown columns.

### Step 2: Fill Out the Template

1. In the **Data** sheet, start filling in rows below the header.
2. **Required columns** have a yellow header. These must have a value in every row.
3. **Reference columns** have a green header. Use the dropdown or enter an existing value.
4. **Optional columns** have a blue header. Leave blank if not needed.
5. Some columns have dropdown menus with allowed values (e.g., **Category** for products). Pick from the list.
6. Save the file when done.

**Rules per module:**

| Module | Required | Notes |
|---|---|---|
| Products | SKU, Name | Category, Brand, UnitOfMeasure must match existing values |
| Categories | Name | Slug is auto-generated if left blank |
| Brands | Name | — |
| Units of Measure | Code, Name | Code must be unique |
| Customers | Name | Phone is used to match existing customers |

**Examples of filling in the template:**

**Brands template** — Data sheet:

| Brand Name | Description | Active |
|---|---|---|
| Nike | Sports footwear and apparel | true |
| Adidas | Performance sportswear | true |
| Samsung | Consumer electronics | true |

- **Brand Name** (yellow header, required): Enter the brand name.
- **Description** (blue header, optional): Brief description.
- **Active** (blue header, optional, default `true`): `true` or `false`. Leave blank to keep active.

---

**Categories template** — Data sheet:

| Category Name | Slug | Description | Active |
|---|---|---|---|
| Electronics | electronics | Electronic devices and accessories | true |
| Clothing | clothing | Apparel and fashion items | true |
| Food & Beverage | | | true |

- **Category Name** (yellow, required): The category name.
- **Slug** (blue, optional): URL-friendly identifier. Auto-generated from the name if left blank.
- **Description** (blue, optional): Free text.
- **Active** (blue, optional, default `true`): `true` or `false`.

---

**Units of Measure template** — Data sheet:

| Code | Name | Description | Active |
|---|---|---|---|
| PCS | Pieces | Individual units | true |
| KG | Kilogram | Weight in kilograms | true |
| LTR | Liter | Volume in liters | true |
| BOX | Box | Carton or box quantity | true |

- **Code** (yellow, required): Short unique code (max 10 characters). Example: `PCS`, `KG`, `MTR`.
- **Name** (yellow, required): Display name for the unit.
- **Description** (blue, optional): Free text.
- **Active** (blue, optional, default `true`).

---

**Customers template** — Data sheet:

| Customer Name | Phone | Email | Address | Note | Active |
|---|---|---|---|---|---|
| John Doe | 081234567890 | john@example.com | Jl. Merdeka No. 10 | Regular customer | true |
| PT Maju Jaya | 0211234567 | info@majujaya.com | | | true |
| Sari Store | 087812345678 | | | New customer | |

- **Customer Name** (yellow, required): Customer or company name.
- **Phone** (yellow, required): Phone number. Used to match existing customers on re-import.
- **Email** (blue, optional): Email address.
- **Address** (blue, optional): Physical or mailing address.
- **Note** (blue, optional): Internal notes.
- **Active** (blue, optional, default `true`).

---

**Products template** — Data sheet:

| SKU | Product Name | Barcode | Category | Brand | Price | Cost | Status | Unit of Measure |
|---|---|---|---|---|---|---|---|---|
| SKU001 | T-Shirt Cotton | 8991234567890 | Clothing | Nike | 150000 | 90000 | active | PCS |
| SKU002 | Running Shoes | | Clothing | Adidas | 350000 | 250000 | active | PCS |
| SKU003 | Mineral Water 600ml | | Food & Beverage | | 5000 | 3000 | active | PCS |

- **SKU** (yellow, required): Unique product SKU.
- **Product Name** (yellow, required): Product display name.
- **Barcode** (blue, optional): Barcode number. Leave blank if no barcode.
- **Category, Brand, Unit of Measure** (green, reference): Must match an existing record in the system. Use the dropdown or type the exact name/code.
- **Price** (yellow, required): Selling price in whole number (IDR).
- **Cost** (blue, optional): Purchase cost in whole number.
- **Status** (blue, optional, default `active`): Allowed values: `active`, `inactive`, `draft`, `archived`.
- **Unit of Measure** (green, reference): Must match an existing UOM code (e.g., `PCS`, `KG`).

> **Important**: For reference columns (green headers), the value you enter must exactly match an existing record. For example, entering "nik" instead of "Nike" will fail validation. Use the dropdown in the template to pick the correct value.

> **Example files**: Pre-filled example templates for all modules are available in `docs/examples/`:
> - [`example_brands_filled.xlsx`](./examples/example_brands_filled.xlsx) — Brands with 10 sample rows
> - [`example_categories_filled.xlsx`](./examples/example_categories_filled.xlsx) — Categories with 10 sample rows
> - [`example_uoms_filled.xlsx`](./examples/example_uoms_filled.xlsx) — Units of Measure with 10 sample rows
> - [`example_customers_filled.xlsx`](./examples/example_customers_filled.xlsx) — Customers with 10 sample rows
> - [`example_products_filled.xlsx`](./examples/example_products_filled.xlsx) — Products with 10 sample rows
>
> Open any of these files in Excel or Google Sheets to see how data should be formatted.

### Step 3: Upload and Preview

1. Click the **Bulk Actions** button, then choose **Import Data**. The **Import** dialog opens. Click the upload area and select your filled-out file (or drag and drop it).
2. Click **Preview**.
3. The system checks your file and shows a preview:
   - **Insert** rows (green) — new records that will be added.
   - **Update** rows (amber) — existing records that will be updated.
   - **Error** rows (red) — rows with problems that will be skipped.

> **Always check the preview before importing.** It shows you exactly what will happen.

### Step 4: Confirm the Import

1. If the preview looks correct, click **Import N Rows**.
2. The import starts in the background. You can watch its progress:
   - A progress bar shows how many rows have been processed.
   - The status changes to **Completed** when done.
   - If there are errors, the status shows **Failed** with details.

3. Click **Close** to finish.

---

## Understanding the Preview

The preview screen shows:

- **Total rows**: Number of rows detected in your file.
- **Insert count**: Rows that will be added as new records.
- **Update count**: Rows that will update existing records (matched by name, code, SKU, or phone).
- **Error count**: Rows with problems that will be skipped.

Each row in the preview table shows:
- **Status** (insert, update, or error).
- **New values** — how the record will look after import.
- **Old values** (updates only) — how the record looks now.
- **Error messages** (errors only) — what went wrong and how to fix it.

> **Note**: If any rows have errors, the **Import** button is disabled. Fix the errors in your file and re-upload.

---

## Tracking Import Progress

After confirming an import, the progress dialog shows:

- **Progress bar**: Visual indication of completion percentage.
- **Rows processed**: How many rows have been handled so far.
- **Inserted / Updated / Errors**: Final counts when the import completes.

If you close the dialog while an import is running, you can check progress later from **any page** by going to the **Import** menu again (the import continues in the background).

---

## Cancelling an Import

To stop a running import:

1. In the progress dialog, click **Cancel**.
2. The import stops as soon as the current row finishes processing.
3. Any rows already imported remain in the system.

---

## Troubleshooting

**"Unknown module" error** — Make sure you're importing to the correct page. Each module has its own import.

**"File is required" error** — You must select a file before clicking Preview.

**"Preview state not found" error** — The preview session expired. Upload the file again and re-preview.

**All rows show errors** — Common causes:
- The file format is wrong. Download a fresh template and copy your data into it.
- Required columns are missing or empty.
- Reference values (e.g., category names) don't match existing records.

**"Permission denied" error** — You don't have permission to import or export. Contact your administrator.

**Import is stuck at 0%** — The system is processing. Large imports (thousands of rows) take time. Wait for the progress to update.

**Job not found** — The import may have already completed and been cleaned up. Check your data directly on the relevant page.

---

## Import History

To view past imports:

1. Click **Bulk Actions** then **Import History**.
2. A dialog shows all past imports for that module with their status (Completed, Failed, Cancelled).
3. Each entry shows the date, row count, and insert/update/error counts.
