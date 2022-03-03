package postgresql

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/spf13/afero"
	"github.com/staticbackendhq/core/internal"
)

type PostgreSQL struct {
	DB              *sql.DB
	PublishDocument internal.PublishDocumentEvent
}

var (
	migrationPath string
	appFS         = afero.NewOsFs()
)

func New(db *sql.DB, pubdoc internal.PublishDocumentEvent, migPath string) internal.Persister {
	migrationPath = migPath

	// run migrations
	if err := migrate(db); err != nil {
		fmt.Println("=== MIGRATION FAILED ===")
		fmt.Println(err)
		fmt.Println("=== /MIGRATION FAILED ===")
		os.Exit(1)
	}

	return &PostgreSQL{DB: db, PublishDocument: pubdoc}
}

func (pg *PostgreSQL) Ping() error {
	return pg.DB.Ping()
}
