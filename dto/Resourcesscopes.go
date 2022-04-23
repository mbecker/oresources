package dto

import (
	"database/sql"
	"time"
)

type Resourcesscopes struct {
	ResourcesscopeID int           `db:"resourcesscope_id"`
	ResourceID       sql.NullInt64 `db:"resource_id"`
	ScopeID          sql.NullInt64 `db:"scope_id"`
	CreatedAt        time.Time     `db:"created_at"`
	UpdatedAt        sql.NullTime  `db:"updated_at"`
	Defaultscope     bool          `db:"defaultscope"`
}
