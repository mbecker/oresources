package dto

import (
	"database/sql"
	"time"
)

type Userpermissions struct {
	UUID             *string      `db:"uuid"`
	ResourcesscopeID int          `db:"resourcesscope_id"`
	CreatedAt        time.Time    `db:"created_at"`
	UpdatedAt        sql.NullTime `db:"updated_at"`
}
