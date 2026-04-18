package models

import "time"

type Role struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	IsSystem    bool      `json:"is_system" db:"is_system"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Permission struct {
	ID          int       `json:"id" db:"id"`
	Code        string    `json:"code" db:"code"`
	Description string    `json:"description" db:"description"`
	Category    string    `json:"category" db:"category"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type RolePermission struct {
	RoleID       int `json:"role_id" db:"role_id"`
	PermissionID int `json:"permission_id" db:"permission_id"`
}
