package audit

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Creator interface {
	CreateAuditLog(ctx context.Context, log *Log) error
}

// TxCreator is a Creator that can also persist an audit log inside an existing
// transaction, enabling atomic "mutation + its audit" operations.
type TxCreator interface {
	Creator
	CreateAuditLogTx(ctx context.Context, tx pgx.Tx, log *Log) error
}

// WriteFailClosed records a security-critical audit event and reports whether
// the write succeeded. It is the fail-closed counterpart to the best-effort
// `CreateAuditLog` calls used elsewhere: when svc is nil (auditing disabled,
// e.g. in tests) it returns true so callers can proceed; when the write fails
// it returns false, signalling the caller to abort the business operation
// rather than persist an unaudited, security-relevant change. The repository
// already meters and logs write failures, so this helper only propagates the
// success/failure signal.
func WriteFailClosed(ctx context.Context, svc Creator, log *Log) bool {
	if svc == nil {
		return true
	}
	return svc.CreateAuditLog(ctx, log) == nil
}

type Log struct {
	ID          int         `json:"id"`
	UserID      *int        `json:"user_id,omitempty"`
	StoreID     *int        `json:"store_id,omitempty"`
	StoreName   string      `json:"store_name,omitempty"`
	Username    string      `json:"username"`
	Role        string      `json:"role"`
	Action        string      `json:"action"`
	EntityType    string      `json:"entity_type"`
	Description   string      `json:"description"`
	IPAddress     string      `json:"ip_address,omitempty"`
	UserAgent     string      `json:"user_agent,omitempty"`
	EntityID      *int        `json:"entity_id,omitempty"`
	OldValues     interface{} `json:"old_values,omitempty"`
	NewValues     interface{} `json:"new_values,omitempty"`
	CorrelationID string      `json:"correlation_id,omitempty"`
	CreatedAt     string      `json:"created_at"`
}

type LogListItem struct {
	ID          int         `json:"id"`
	UserID      *int        `json:"user_id,omitempty"`
	StoreID     *int        `json:"store_id,omitempty"`
	StoreName   string      `json:"store_name,omitempty"`
	Username    string      `json:"username"`
	Role        string      `json:"role"`
	Action      string      `json:"action"`
	EntityType  string      `json:"entity_type"`
	Description string      `json:"description"`
	IPAddress   string      `json:"ip_address,omitempty"`
	UserAgent   string      `json:"user_agent,omitempty"`
	EntityID    *int        `json:"entity_id,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CreatedAt   string      `json:"created_at"`
	OldValues   interface{} `json:"old_values,omitempty"`
	NewValues   interface{} `json:"new_values,omitempty"`
}
