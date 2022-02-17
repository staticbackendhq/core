package postgresql

import (
	"database/sql"

	"github.com/staticbackendhq/core/internal"
)

type PostgreSQL struct {
	DB              *sql.DB
	PublishDocument internal.PublishDocumentEvent
}

func New(db *sql.DB, pubdoc internal.PublishDocumentEvent) internal.Persister {
	return &PostgreSQL{DB: db, PublishDocument: pubdoc}
}

func (pg *PostgreSQL) Ping() error {
	return pg.DB.Ping()
}
