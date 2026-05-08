#!/bin/bash
# Test script untuk verifikasi dummy seeder requirements

echo "🔍 Testing Dummy Seeder Requirements"
echo "====================================="

# Database connection
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-pos}
DB_PASSWORD=${DB_PASSWORD:-admin123}
DB_NAME=${DB_NAME:-retail_pos}

# Function to run SQL queries
run_sql() {
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -c "$1" -t
}

echo "📊 Checking overall statistics..."
OVERALL_STATS=$(run_sql "
SELECT
    COUNT(*) as total_sales,
    COUNT(DISTINCT DATE(created_at)) as active_days,
    MIN(DATE(created_at)) as earliest_date,
    MAX(DATE(created_at)) as latest_date,
    MAX(created_at) as latest_timestamp
FROM sales;
")

echo "$OVERALL_STATS"

echo ""
echo "📅 Checking daily transaction counts (last 10 days)..."
DAILY_STATS=$(run_sql "
SELECT
    DATE(created_at) as date,
    COUNT(*) as transaction_count,
    MIN(created_at) as first_transaction,
    MAX(created_at) as last_transaction
FROM sales
WHERE DATE(created_at) >= CURRENT_DATE - INTERVAL '10 days'
GROUP BY DATE(created_at)
ORDER BY DATE(created_at) DESC;
")

echo "$DAILY_STATS"

echo ""
echo "⚠️  Checking for days with less than 10 transactions..."
LOW_DAYS=$(run_sql "
SELECT
    DATE(created_at) as date,
    COUNT(*) as transaction_count
FROM sales
GROUP BY DATE(created_at)
HAVING COUNT(*) < 10
ORDER BY DATE(created_at);
")

if [ -z "$LOW_DAYS" ]; then
    echo "✅ All days have at least 10 transactions!"
else
    echo "❌ Days with less than 10 transactions:"
    echo "$LOW_DAYS"
fi

echo ""
echo "📅 Checking today's transactions..."
TODAY_STATS=$(run_sql "
SELECT
    COUNT(*) as today_transactions,
    MIN(created_at) as first_today,
    MAX(created_at) as last_today,
    EXTRACT(hour from MAX(created_at)) as last_hour,
    EXTRACT(minute from MAX(created_at)) as last_minute,
    EXTRACT(second from MAX(created_at)) as last_second
FROM sales
WHERE DATE(created_at) = CURRENT_DATE;
")

echo "$TODAY_STATS"

# Check if there are transactions today
TODAY_COUNT=$(run_sql "SELECT COUNT(*) FROM sales WHERE DATE(created_at AT TIME ZONE 'Asia/Jakarta') = CURRENT_DATE;")
if [ "$TODAY_COUNT" -eq 0 ]; then
    echo "❌ No transactions today!"
else
    echo "✅ Today has transactions"

    # Check if latest transaction is before current time (considering timezone)
    LATEST_TIME=$(run_sql "SELECT MAX(created_at AT TIME ZONE 'Asia/Jakarta') FROM sales WHERE DATE(created_at AT TIME ZONE 'Asia/Jakarta') = CURRENT_DATE;")
    CURRENT_TIME_WIB=$(run_sql "SELECT NOW() AT TIME ZONE 'Asia/Jakarta';")

    echo "Latest transaction (WIB): $LATEST_TIME"
    echo "Current time (WIB): $CURRENT_TIME_WIB"

    # Use PostgreSQL to compare times directly
    TIME_COMPARISON=$(run_sql "
    SELECT
        CASE
            WHEN MAX(created_at AT TIME ZONE 'Asia/Jakarta') < NOW() AT TIME ZONE 'Asia/Jakarta'
            THEN 'PAST'
            ELSE 'FUTURE'
        END as comparison
    FROM sales
    WHERE DATE(created_at AT TIME ZONE 'Asia/Jakarta') = CURRENT_DATE;
    ")

    # Trim whitespace from TIME_COMPARISON
    TIME_COMPARISON=$(echo "$TIME_COMPARISON" | tr -d '[:space:]')

    if [ "$TIME_COMPARISON" = "PAST" ]; then
        echo "✅ Latest transaction is in the past"
    else
        echo "❌ Latest transaction is in the future!"
    fi
fi

echo ""
echo "🎫 Checking invoice number format..."
INVOICE_CHECK=$(run_sql "
SELECT
    COUNT(*) as total_invoices,
    COUNT(*) FILTER (WHERE invoice_number ~ '^INV-\d{4}-\d{6}$') as valid_format,
    MIN(invoice_number) as first_invoice,
    MAX(invoice_number) as last_invoice
FROM sales;
")

echo "$INVOICE_CHECK"

# Check for duplicate invoice numbers
DUPLICATES=$(run_sql "
SELECT invoice_number, COUNT(*)
FROM sales
GROUP BY invoice_number
HAVING COUNT(*) > 1
ORDER BY invoice_number
LIMIT 5;
")

if [ -z "$DUPLICATES" ]; then
    echo "✅ No duplicate invoice numbers!"
else
    echo "❌ Duplicate invoice numbers found:"
    echo "$DUPLICATES"
fi

echo ""
echo "🔍 Checking data quality..."

# Check for future transactions (considering timezone)
FUTURE_TRANSACTIONS=$(run_sql "
SELECT COUNT(*) as future_transactions
FROM sales
WHERE created_at AT TIME ZONE 'Asia/Jakarta' > NOW() AT TIME ZONE 'Asia/Jakarta';
")

if [ "$FUTURE_TRANSACTIONS" -eq 0 ]; then
    echo "✅ No transactions in the future"
else
    echo "❌ Found $FUTURE_TRANSACTIONS transactions in the future!"
fi

# Check for very old transactions (should be reasonable range)
OLD_TRANSACTIONS=$(run_sql "
SELECT COUNT(*) as very_old_transactions
FROM sales
WHERE created_at < CURRENT_DATE - INTERVAL '200 days';
")

if [ "$OLD_TRANSACTIONS" -eq 0 ]; then
    echo "✅ No excessively old transactions"
else
    echo "⚠️ Found $OLD_TRANSACTIONS transactions older than 200 days"
fi

echo ""
echo "🎯 Summary of Requirements:"
echo "✅ Every day should have at least 10 transactions"
echo "✅ Transactions should include today (before current time)"
echo "✅ Invoice format: INV-yyyy-xxxxxx"
echo "✅ No duplicate invoices"
echo "✅ No future transactions"
echo ""
echo "Test completed! 🎉"