package sqlite

import (
	"database/sql"
	"embed"
	"fmt"
	"os"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/logger"
)

//go:embed sql
var migrationFS embed.FS

type SQLite struct {
	DB              *sql.DB
	PublishDocument cache.PublishDocumentEvent
	log             *logger.Logger

	collections map[string]bool
}

func New(db *sql.DB, pubdoc cache.PublishDocumentEvent, log *logger.Logger) database.Persister {
	// run migrations
	if err := migrate(db); err != nil {
		fmt.Println("=== MIGRATION FAILED ===")
		fmt.Println(err)
		fmt.Println("=== /MIGRATION FAILED ===")
		os.Exit(1)
	}

	return &SQLite{
		DB:              db,
		PublishDocument: pubdoc,
		collections:     make(map[string]bool),
		log:             log,
	}
}

func (sl *SQLite) Ping() error {
	return sl.DB.Ping()
}

func (sl *SQLite) CreateIndex(dbName, col, field string) error {
	// TODO: this does not seems it's possible to create an index on a JSON field
	/*
		qry := `
			CREATE INDEX IF NOT EXISTS
				{schema}_idx_{col}_{field}
			ON {schema}_{col}({field}, json_extract(data, "$.{field}"));
		`

		qry = strings.Replace(qry, "{col}", model.CleanCollectionName(col), -1)
		qry = strings.Replace(qry, "{field}", field, -1)
		qry = strings.Replace(qry, "{schema}", dbName, -1)

		if _, err := sl.DB.Exec(qry); err != nil {
			return err
		}
	*/
	return nil
}
