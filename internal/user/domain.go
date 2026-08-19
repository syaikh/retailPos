package user

import "time"

type User struct {
	ID                int    `json:"id"`
	Username          string `json:"username"`
	Email             string `json:"email"`
	Password          string `json:"-"`
	RoleID            int    `json:"role_id"`
	Role              Role   `json:"role"`
	StoreID           *int   `json:"store_id,omitempty"`
	ReportsToID       *int   `json:"reports_to,omitempty"`
	ReportsToUsername string `json:"reports_to_username,omitempty"`
	IsActive          bool   `json:"is_active"`
	Language          string `json:"language"`
	Theme             string `json:"theme"`
	LastLogin         string `json:"last_login,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

type Role struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

type Permission struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at,omitempty"`
}

type Claims struct {
	ID          int      `json:"id"`
	Username    string   `json:"username"`
	RoleID      int      `json:"role_id"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	StoreID     *int     `json:"store_id,omitempty"`
	ReportsToID *int     `json:"reports_to,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type WithPermissions struct {
	ID                int      `json:"id"`
	Username          string   `json:"username"`
	Role              string   `json:"role"`
	Permissions       []string `json:"permissions"`
	StoreID           *int     `json:"store_id,omitempty"`
	Language          string   `json:"language"`
	Theme             string   `json:"theme"`
	ReportsToID       *int     `json:"reports_to,omitempty"`
	ReportsToUsername string   `json:"reports_to_username,omitempty"`
}
