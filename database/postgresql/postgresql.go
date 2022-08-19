package postgresql

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/afero"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/logger"
)

type PostgreSQL struct {
	DB              *sql.DB
	PublishDocument internal.PublishDocumentEvent
	log             *logger.Logger
}

var (
	migrationPath string
	appFS         = afero.NewOsFs()
)

func New(db *sql.DB, pubdoc internal.PublishDocumentEvent, migPath string, log *logger.Logger) internal.Persister {
	migrationPath = migPath

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

	qry = strings.Replace(qry, "{col}", internal.CleanCollectionName(col), -1)
	qry = strings.Replace(qry, "{field}", field, -1)
	qry = strings.Replace(qry, "{schema}", dbName, -1)

	if _, err := pg.DB.Exec(qry); err != nil {
		return err
	}
	return nil
}
