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
- An **Instructions** sheet explaining each column.
- A **Data** sheet with column headers where you fill in your records.
- **Reference sheets** (hidden) that list valid values for dropdown columns.

### Step 2: Fill Out the Template

1. In the **Data** sheet, start filling in rows below the header.
2. **Required columns** are marked with a `*` in the header. These must have a value in every row.
3. Some columns have dropdown menus with allowed values (e.g., **Category** for products). Pick from the list.
4. Save the file when done.

**Rules per module:**

| Module | Required | Notes |
|---|---|---|
| Products | SKU, Name | Category, Brand, UnitOfUse must match existing values |
| Categories | Name | Slug is auto-generated if left blank |
| Brands | Name | — |
| Units of Measure | Code, Name | Code must be unique |
| Customers | Name | Phone is used to match existing customers |

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
