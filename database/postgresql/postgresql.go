package postgresql

import (
	"database/sql"

	"github.com/staticbackendhq/core/internal"
)

type PostgreSQL struct {
	DB *sql.DB
}

func New(db *sql.DB) internal.Persister {
	return &PostgreSQL{DB: db}
}

func (pg *PostgreSQL) Ping() error {
	return pg.DB.Ping()
}
