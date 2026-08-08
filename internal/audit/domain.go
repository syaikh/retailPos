package audit

import "context"

type Creator interface {
	CreateAuditLog(ctx context.Context, log *Log) error
}

type Log struct {
	ID          int         `json:"id"`
	UserID      *int        `json:"user_id,omitempty"`
	Username    string      `json:"username"`
	Role        string      `json:"role"`
	Action      string      `json:"action"`
	EntityType  string      `json:"entity_type"`
	Description string      `json:"description"`
	IPAddress   string      `json:"ip_address,omitempty"`
	UserAgent   string      `json:"user_agent,omitempty"`
	EntityID    *int        `json:"entity_id,omitempty"`
	OldValues   interface{} `json:"old_values,omitempty"`
	NewValues   interface{} `json:"new_values,omitempty"`
	CreatedAt   string      `json:"created_at"`
}

type LogListItem struct {
	ID          int    `json:"id"`
	UserID      *int   `json:"user_id,omitempty"`
	Username    string `json:"username"`
	Role        string `json:"role"`
	Action      string `json:"action"`
	EntityType  string `json:"entity_type"`
	Description string `json:"description"`
	IPAddress   string `json:"ip_address,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
	EntityID    *int   `json:"entity_id,omitempty"`
	CreatedAt   string `json:"created_at"`
	OldValues   interface{} `json:"old_values,omitempty"`
	NewValues   interface{} `json:"new_values,omitempty"`
}
