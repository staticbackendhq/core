package postgresql

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
)

type PostgreSQL struct {
	DB              *sql.DB
	PublishDocument cache.PublishDocumentEvent
	log             *logger.Logger
}

//go:embed sql
var migrationFS embed.FS

func New(db *sql.DB, pubdoc cache.PublishDocumentEvent, log *logger.Logger) database.Persister {
	// run migrations
	if err := migrate(db); err != nil {
		fmt.Println("=== MIGRATION FAILED ===")
		fmt.Println(err)
		fmt.Println("=== /MIGRATION FAILED ===")
		os.Exit(1)
	}

	return &PostgreSQL{DB: db, PublishDocument: pubdoc, log: log}
}

func (pg *PostgreSQL) Ping() error {
	return pg.DB.Ping()
}

func (pg *PostgreSQL) CreateIndex(dbName, col, field string) error {
	qry := `
		CREATE INDEX IF NOT EXISTS 
			idx_{col}_{field} 
		ON {schema}.{col} 
		USING btree ((data->'{col}'))
	`

	qry = strings.Replace(qry, "{col}", model.CleanCollectionName(col), -1)
	qry = strings.Replace(qry, "{field}", field, -1)
	qry = strings.Replace(qry, "{schema}", dbName, -1)

	if _, err := pg.DB.Exec(qry); err != nil {
		return err
	}
	return nil
}
