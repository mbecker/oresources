package dto

import (
	"database/sql"
	"time"
)

type Scopes struct {
	ScopeID   int          `db:"scope_id"`
	Name      string       `db:"name"`
	CreatedAt time.Time    `db:"created_at"`
	UpdatedAt sql.NullTime `db:"updated_at"`
}
