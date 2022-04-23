package dto

import (
	"database/sql"
	"time"
)

type Rolepermissions struct {
	RoleID           int           `db:"role_id"`
	Name             string        `db:"name"`
	ExtID            string        `db:"ext_id"`
	ResourcesscopeID sql.NullInt64 `db:"resourcesscope_id"`
	CreatedAt        time.Time     `db:"created_at"`
	UpdatedAt        sql.NullTime  `db:"updated_at"`
}
