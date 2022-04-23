package dto

import (
	"database/sql"
	"time"
)

type Tenants struct {
	TenantID  int          `db:"tenant_id"`
	Name      string       `db:"name"`
	CreatedAt time.Time    `db:"created_at"`
	UpdatedAt sql.NullTime `db:"updated_at"`
}
