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
	Roles        []string                 `json:"roles"`
	Resources    map[string]ResourcesTree `json:"resources"`
}

type DBRole struct {
	RoleID int    `db:"role_id"`
	Name   string `db:"name"`
}

type Role struct {
	Name        string                   `json:"original_name"`
	Role        string                   `json:"role"`
	Type        string                   `json:"type"`
	Permission  []ResourcesPermission    `json:"permissions"`
	ResourceTee map[string]ResourcesTree `json:"resource_tree"`
}

type ResourcesRolesResult struct {
	ResourcesRoles ResourcesRoles           `json:"resources_roles"`
	ResourceTee    map[string]ResourcesTree `json:"resource_tree"`
}

type RoleSingle struct {
	Name string `db:"name"`
}

type ResourcesRoles map[string][]string

func NewResourcesRoles() ResourcesRoles {
	return ResourcesRoles{}
}
