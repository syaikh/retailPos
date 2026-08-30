package audit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"retail-pos-system/internal/metrics"
	"retail-pos-system/internal/shared"
)

type Repository struct {
	db shared.DBPool
}

// AuditRetentionDays is the recommended retention window for audit logs.
// Operators should schedule PurgeOlderThan (e.g. nightly) to delete logs older
// than now minus this window. Keeping raw audit rows forever is both a storage
// and a long-term PII-exposure concern.
const AuditRetentionDays = 365

// nullIfEmpty returns nil for an empty string so the column stores NULL rather
// than an empty string.
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

// PurgeOlderThan deletes audit logs created before the given time. It is the
// mechanical half of the retention policy and is intended to be invoked by a
// scheduled job; it is intentionally not wired to any business action so that
// routine operations can never silently erase the trail.
func (r *Repository) PurgeOlderThan(ctx context.Context, before time.Time) (int64, error) {
	cmd, err := r.db.Exec(ctx, `DELETE FROM audit_logs WHERE created_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return cmd.RowsAffected(), nil
}

// queryExecer is satisfied by both shared.DBPool and pgx.Tx, so the same
// insert logic runs inside or outside an explicit transaction.
type queryExecer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (r *Repository) CreateAuditLog(ctx context.Context, log *Log) error {
	return r.createAuditLog(ctx, r.db, log)
}

// CreateAuditLogTx persists an audit log within an existing transaction; the
// caller is responsible for committing/rolling back tx. This is what makes a
// "mutation + its audit" pair atomic.
func (r *Repository) CreateAuditLogTx(ctx context.Context, tx pgx.Tx, log *Log) error {
	return r.createAuditLog(ctx, tx, log)
}

// insertAuditLog persists a single audit row against the given query execer
// (either the pool or a pgx.Tx). `userID` is passed separately so a dangling
// reference can be retried as NULL without mutating the caller's Log.
func (r *Repository) insertAuditLog(ctx context.Context, qx queryExecer, log *Log, ipAddr interface{}, userID any) error {
	return qx.QueryRow(ctx, `
		INSERT INTO audit_logs (user_id, store_id, role, action, entity_type, entity_id, ip_address, user_agent, old_values, new_values, description, correlation_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`, userID, log.StoreID, log.Role, log.Action, log.EntityType, log.EntityID, ipAddr, log.UserAgent, log.OldValues, log.NewValues, log.Description, nullIfEmpty(log.CorrelationID)).Scan(&log.ID)
}

// isForeignKeyViolation reports whether err is a Postgres foreign-key violation
// (SQLSTATE 23503), i.e. the only self-inflicted reason an audit insert can fail
// for a well-formed row: a non-nil user_id pointing to a deleted/bad user.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func (r *Repository) createAuditLog(ctx context.Context, qx queryExecer, log *Log) error {
	var ipAddr interface{}
	if log.IPAddress != "" {
		ipAddr = log.IPAddress
	}
	// Default the correlation ID from the request context when the caller did
	// not set one explicitly (e.g. seeded/background events). The request
	// logging middleware attaches X-Request-ID to the context for every HTTP
	// request, so normal API-driven events are traced automatically.
	if log.CorrelationID == "" {
		log.CorrelationID = shared.GetRequestID(ctx)
	}

	err := r.insertAuditLog(ctx, qx, log, ipAddr, log.UserID)
	if err == nil {
		return nil
	}

	// A non-nil user_id referencing a deleted or otherwise-missing user must
	// never block the surrounding business operation (audit fail-closed spans a
	// sale/PO/inventory mutation). Retry once with user_id = NULL, keeping the
	// denormalized username/role, so the audit row is still persisted and the
	// transaction is not rolled back over a user-data hiccup.
	if log.UserID != nil && isForeignKeyViolation(err) {
		if retryErr := r.insertAuditLog(ctx, qx, log, ipAddr, nil); retryErr == nil {
			return nil
		} else {
			metrics.AuditWriteFailures.Inc()
			shared.LogError(ctx, "failed to write audit log after dropping dangling user_id",
				retryErr,
				"action", log.Action,
				"entity_type", log.EntityType,
				"entity_id", log.EntityID,
				"user_id", log.UserID,
				"store_id", log.StoreID,
				"username", log.Username,
			)
			return retryErr
		}
	}

	metrics.AuditWriteFailures.Inc()
	shared.LogError(ctx, "failed to write audit log",
		err,
		"action", log.Action,
		"entity_type", log.EntityType,
		"entity_id", log.EntityID,
		"user_id", log.UserID,
		"store_id", log.StoreID,
		"username", log.Username,
	)
	return err
}

func (r *Repository) GetDistinctEntityTypes(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT entity_type FROM audit_logs ORDER BY entity_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, nil
}

func (r *Repository) GetAuditLogs(ctx context.Context, limit, offset int, userID *int, search string, action string, entityType string, entityID *int, startDate *time.Time, endDate *time.Time) ([]LogListItem, int, error) {
	var logs []LogListItem
	var total int

	query := `SELECT COUNT(*) FROM audit_logs al LEFT JOIN users u ON al.user_id = u.id WHERE 1=1`
	args := []interface{}{}
	if userID != nil {
		query += fmt.Sprintf(" AND al.user_id = $%d", len(args)+1)
		args = append(args, *userID)
	}
	if action != "" {
		query += fmt.Sprintf(" AND al.action = $%d", len(args)+1)
		args = append(args, action)
	}
	if search != "" {
		query += fmt.Sprintf(" AND (u.username ILIKE $%d OR al.role ILIKE $%d OR al.action ILIKE $%d OR al.entity_type ILIKE $%d OR al.ip_address::text ILIKE $%d)", len(args)+1, len(args)+2, len(args)+3, len(args)+4, len(args)+5)
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if entityType != "" {
		query += fmt.Sprintf(" AND al.entity_type = $%d", len(args)+1)
		args = append(args, entityType)
	}
	if entityID != nil {
		query += fmt.Sprintf(" AND al.entity_id = $%d", len(args)+1)
		args = append(args, *entityID)
	}
	if startDate != nil {
		query += fmt.Sprintf(" AND al.created_at >= $%d", len(args)+1)
		args = append(args, *startDate)
	}
	if endDate != nil {
		query += fmt.Sprintf(" AND al.created_at < $%d", len(args)+1)
		args = append(args, endDate.Add(24*time.Hour))
	}

	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query = `SELECT al.id, al.user_id, al.store_id, COALESCE(s.name, ''), COALESCE(u.username, 'Unknown'), COALESCE(al.role, ''), al.action, al.entity_type, al.entity_id, COALESCE(al.ip_address::text, ''), COALESCE(al.user_agent, ''), to_char(al.created_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM-DD"T"HH24:MI:SS+07:00'), COALESCE(al.description, ''), COALESCE(al.old_values, '{}'::jsonb), COALESCE(al.new_values, '{}'::jsonb), COALESCE(al.correlation_id, '') FROM audit_logs al LEFT JOIN users u ON al.user_id = u.id LEFT JOIN stores s ON al.store_id = s.id WHERE 1=1`
	args2 := []interface{}{}
	if userID != nil {
		query += fmt.Sprintf(" AND al.user_id = $%d", len(args2)+1)
		args2 = append(args2, *userID)
	}
	if action != "" {
		query += fmt.Sprintf(" AND al.action = $%d", len(args2)+1)
		args2 = append(args2, action)
	}
	if search != "" {
		query += fmt.Sprintf(" AND (u.username ILIKE $%d OR al.role ILIKE $%d OR al.action ILIKE $%d OR al.entity_type ILIKE $%d OR al.ip_address::text ILIKE $%d)", len(args2)+1, len(args2)+2, len(args2)+3, len(args2)+4, len(args2)+5)
		args2 = append(args2, "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if entityType != "" {
		query += fmt.Sprintf(" AND al.entity_type = $%d", len(args2)+1)
		args2 = append(args2, entityType)
	}
	if entityID != nil {
		query += fmt.Sprintf(" AND al.entity_id = $%d", len(args2)+1)
		args2 = append(args2, *entityID)
	}
	if startDate != nil {
		query += fmt.Sprintf(" AND al.created_at >= $%d", len(args2)+1)
		args2 = append(args2, *startDate)
	}
	if endDate != nil {
		query += fmt.Sprintf(" AND al.created_at < $%d", len(args2)+1)
		args2 = append(args2, endDate.Add(24*time.Hour))
	}
	query += fmt.Sprintf(" ORDER BY al.created_at DESC LIMIT $%d OFFSET $%d", len(args2)+1, len(args2)+2)
	args2 = append(args2, limit, offset)

	rows, err := r.db.Query(ctx, query, args2...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var al LogListItem
		err = rows.Scan(&al.ID, &al.UserID, &al.StoreID, &al.StoreName, &al.Username, &al.Role, &al.Action, &al.EntityType, &al.EntityID, &al.IPAddress, &al.UserAgent, &al.CreatedAt, &al.Description, &al.OldValues, &al.NewValues, &al.CorrelationID)
		if err != nil {
			slog.Error("error scanning audit log row", "error", err)
			continue
		}
		logs = append(logs, al)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func (r *Repository) GetAuditLogByID(ctx context.Context, id int) (*Log, error) {
	var al Log
	err := r.db.QueryRow(ctx, `
		SELECT al.id, al.user_id, al.store_id, COALESCE(s.name, ''), COALESCE(u.username, 'Unknown'), COALESCE(al.role, ''), al.action, al.entity_type, al.entity_id, COALESCE(al.ip_address::text, ''), COALESCE(al.user_agent, ''), COALESCE(al.old_values, '{}'::jsonb), COALESCE(al.new_values, '{}'::jsonb), COALESCE(al.correlation_id, ''), to_char(al.created_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM-DD"T"HH24:MI:SS+07:00'), COALESCE(al.description, '')
		FROM audit_logs al
		LEFT JOIN users u ON al.user_id = u.id
		LEFT JOIN stores s ON al.store_id = s.id
		WHERE al.id = $1
	`, id).Scan(&al.ID, &al.UserID, &al.StoreID, &al.StoreName, &al.Username, &al.Role, &al.Action, &al.EntityType, &al.EntityID, &al.IPAddress, &al.UserAgent, &al.OldValues, &al.NewValues, &al.CorrelationID, &al.CreatedAt, &al.Description)
	if err != nil {
		return nil, err
	}
	return &al, nil
}
