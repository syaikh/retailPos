# Revenue Chart Implementation

This plan outlines the steps to wire up the "Revenue Overview" chart on the Reports page using **Chart.js**. We will choose Chart.js over a Svelte-specific wrapper because standard Chart.js integrates beautifully with Svelte 5 via standard Svelte Actions (`use:action`), providing maximum flexibility, performance, and access to all native Chart.js features without intermediate library bugs.

## Proposed Changes

### 1. Backend: Data Aggregation
The backend currently has a stub endpoint for chart data. We need to implement it and expose it via the API.

#### [MODIFY] internal/delivery/http/handler/handler.go
- Implement `GetSalesChartData` to accept `startDate` and `endDate` query parameters.
- Query the `saleRepo` for transactions within that date range.
- Aggregate the `TotalAmount` grouped by day (`YYYY-MM-DD`).
- Return a sorted array of data points: `[{ "date": "2026-05-01", "total": 1500000 }, ...]`.

#### [MODIFY] cmd/server/main.go
- Add the route: `protected.GET("/dashboard/chart", h.GetSalesChartData)` to the API router.

---

### 2. Frontend: Chart.js Integration

#### [NEW] Dependency
- Run `npm install chart.js` in the `web` directory.

#### [NEW] web/src/lib/actions/chart.js
- Create a reusable Svelte Action for Chart.js.
- This action will initialize a `new Chart()` on a canvas node and automatically call `chart.update()` or `chart.destroy()` when Svelte updates the component's state or unmounts it.

#### [MODIFY] web/src/lib/pages/ReportsPage.svelte
- Fetch chart data from `/api/dashboard/chart` alongside the regular `fetchSales` transaction history.
- Replace the current breathing placeholder `<div>` with a `<canvas use:chart={chartConfig}>`.
- Configure the chart as a Line Chart with:
  - Tension (`0.4`) for smooth, elegant curves.
  - A primary color stroke (`#7c3aed`).
  - A beautiful semi-transparent gradient fill underneath the line to match our Deep Space Dark theme.
  - Custom tooltip formatting to show currency (`Rp`).

## Verification Plan

### Automated Tests
- Build the Go backend and ensure the new `/api/dashboard/chart` endpoint returns valid aggregated JSON data.
- Build the Svelte frontend to ensure no build errors exist with the new `chart.js` dependency.

### Manual Verification
- Navigate to the Reports page.
- Verify the chart renders smoothly and matches the visual aesthetics of the application.
- Change the date range and verify the chart updates reactively alongside the transaction table.
