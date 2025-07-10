package gpa

import "database/sql"

type Connection struct {
	*DBConfig
	*sql.DB
}
