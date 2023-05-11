package sqlite

import (
	"database/sql"
	"embed"
	"strings"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
)

//go:embed sql
var migrationFS embed.FS

type SQLite struct {
	DB              *sql.DB
	PublishDocument cache.PublishDocumentEvent
	log             logger.Logger
}

func New(db *sql.DB, pubdoc cache.PublishDocumentEvent) database.Persister {
	return &SQLite{DB: db, PublishDocument: pubdoc}
}

func (sl *SQLite) Ping() error {
	return sl.DB.Ping()
}

func (sl *SQLite) CreateIndex(dbName, col, field string) error {
	qry := `
		CREATE INDEX IF NOT EXISTS 
			{schema}_idx_{col}_{field} 
		ON {schema}_{col} 
		USING btree ((data->'{col}'))
	`

	qry = strings.Replace(qry, "{col}", model.CleanCollectionName(col), -1)
	qry = strings.Replace(qry, "{field}", field, -1)
	qry = strings.Replace(qry, "{schema}", dbName, -1)

	if _, err := sl.DB.Exec(qry); err != nil {
		return err
	}
	return nil
}
