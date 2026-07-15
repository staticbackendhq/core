package sqlite

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/logger"
)

//go:embed sql
var migrationFS embed.FS

type SQLite struct {
	DB              *sql.DB
	PublishDocument cache.PublishDocumentEvent

	collections map[string]bool
}

func New(db *sql.DB, pubdoc cache.PublishDocumentEvent) database.Persister {
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		logger.FatalError("SQLITE PRAGMA FAILED", err)
	}

	// run migrations
	if err := migrate(db); err != nil {
		logger.FatalError("MIGRATION FAILED", err)
	}

	return &SQLite{
		DB:              db,
		PublishDocument: pubdoc,
		collections:     make(map[string]bool),
	}
}

func (sl *SQLite) Ping() error {
	return sl.DB.Ping()
}

// Close closes the underlying database connection pool.
func (sl *SQLite) Close() error {
	return sl.DB.Close()
}

func (sl *SQLite) CreateIndex(dbName, col, field string) error {
	// TODO: this does not seems it's possible to create an index on a JSON field
	/*
		qry := `
			CREATE INDEX IF NOT EXISTS
				{schema}_idx_{col}_{field}
			ON {schema}_{col}({field}, json_extract(data, "$.{field}"));
		`

		qry = strings.ReplaceAll(qry, "{col}", model.CleanCollectionName(col))
		qry = strings.ReplaceAll(qry, "{field}", field)
		qry = strings.ReplaceAll(qry, "{schema}", dbName)

		if _, err := sl.DB.Exec(qry); err != nil {
			return err
		}
	*/
	return nil
}

func (sl *SQLite) CreateTypedIndex(dbName, col, field string, typ database.IndexType) error {
	if !database.IsSupportedIndexType(typ) {
		return fmt.Errorf("index type %q is not supported", typ)
	}
	return nil
}
