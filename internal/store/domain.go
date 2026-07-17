package store

type Store struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Address   string `json:"address,omitempty"`
	Phone     string `json:"phone,omitempty"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at,omitempty"`
}

type StoreCreateRequest struct {
	Name    string `json:"name" binding:"required,max=100"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

type StoreUpdateRequest struct {
	Name     *string `json:"name" binding:"omitempty,max=100"`
	Address  *string `json:"address"`
	Phone    *string `json:"phone"`
	IsActive *bool   `json:"is_active"`
}

type StoreImportRow struct {
	Row      int
	Name     string
	Address  string
	Phone    string
	IsActive bool
}
