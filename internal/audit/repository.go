package audit

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"retail-pos-system/internal/shared"
)

type Repository struct {
	db shared.DBPool
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	var ipAddr interface{}
	if log.IPAddress != "" {
		ipAddr = log.IPAddress
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO audit_logs (user_id, role, action, entity_type, entity_id, ip_address, user_agent, old_values, new_values, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, log.UserID, log.Role, log.Action, log.EntityType, log.EntityID, ipAddr, log.UserAgent, log.OldValues, log.NewValues, log.Description).Scan(&log.ID)
	if err != nil {
		shared.LogError(ctx, "failed to write audit log",
			err,
			"action", log.Action,
			"entity_type", log.EntityType,
			"entity_id", log.EntityID,
			"username", log.Username,
		)
	}
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

func (r *Repository) GetAuditLogs(ctx context.Context, limit, offset int, userID *int, search string, action string, entityType string, entityID *int, startDate *time.Time, endDate *time.Time) ([]AuditLogListItem, int, error) {
	var logs []AuditLogListItem
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

	query = `SELECT al.id, al.user_id, COALESCE(u.username, 'Unknown'), COALESCE(al.role, ''), al.action, al.entity_type, al.entity_id, COALESCE(al.ip_address::text, ''), COALESCE(al.user_agent, ''), to_char(al.created_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM-DD"T"HH24:MI:SS+07:00'), COALESCE(al.description, '') FROM audit_logs al LEFT JOIN users u ON al.user_id = u.id WHERE 1=1`
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
		var al AuditLogListItem
		err = rows.Scan(&al.ID, &al.UserID, &al.Username, &al.Role, &al.Action, &al.EntityType, &al.EntityID, &al.IPAddress, &al.UserAgent, &al.CreatedAt, &al.Description)
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

func (r *Repository) GetAuditLogByID(ctx context.Context, id int) (*AuditLog, error) {
	var al AuditLog
	err := r.db.QueryRow(ctx, `
		SELECT al.id, al.user_id, COALESCE(u.username, 'Unknown'), COALESCE(al.role, ''), al.action, al.entity_type, al.entity_id, COALESCE(al.ip_address::text, ''), COALESCE(al.user_agent, ''), COALESCE(al.old_values, '{}'::jsonb), COALESCE(al.new_values, '{}'::jsonb), to_char(al.created_at AT TIME ZONE 'Asia/Jakarta', 'YYYY-MM-DD"T"HH24:MI:SS+07:00'), COALESCE(al.description, '')
		FROM audit_logs al
		LEFT JOIN users u ON al.user_id = u.id
		WHERE al.id = $1
	`, id).Scan(&al.ID, &al.UserID, &al.Username, &al.Role, &al.Action, &al.EntityType, &al.EntityID, &al.IPAddress, &al.UserAgent, &al.OldValues, &al.NewValues, &al.CreatedAt, &al.Description)
	if err != nil {
		return nil, err
	}
	return &al, nil
}
