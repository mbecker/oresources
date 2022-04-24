package dto

import (
	"time"
)

type Resources struct {
	ResourceID int    `db:"resource_id"`
	Name       string `db:"name"`
	Type       string `db:"type"`
	// Parent     *int64     `db:"parent"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt *time.Time `db:"updated_at"`
}

type ResourcesPermission struct {
	ResourceID int    `db:"resource_id"`
	Name       string `db:"name"`
	Type       string `db:"type"`
	ScopeID    string `db:"scope_id"`
	ScopeName  string `db:"scope_name"`
}

type ResourcesTree struct {
	Name         string                   `json:"name"`
	OriginalName string                   `json:"original_name"`
	Type         string                   `json:"type"`
	Scopes       []string                 `json:"scopes"`
	Resources    map[string]ResourcesTree `json:"resources"`
}
