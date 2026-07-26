package postgresql

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/database"
	sbquery "github.com/staticbackendhq/core/internal/query"
	"github.com/staticbackendhq/core/model"
)

type PostgreSQL struct {
	DB              *sql.DB
	PublishDocument cache.PublishDocumentEvent
}

//go:embed sql
var migrationFS embed.FS

func New(db *sql.DB, pubdoc cache.PublishDocumentEvent) database.Persister {
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

// Close closes the underlying database connection pool.
func (pg *PostgreSQL) Close() error {
	return pg.DB.Close()
}

func (pg *PostgreSQL) CreateIndex(dbName, col, field string) error {
	return pg.createIndex(dbName, col, field, sbquery.TypeDefault)
}

func (pg *PostgreSQL) CreateTypedIndex(dbName, col, field string, typ database.IndexType) error {
	switch typ {
	case database.IndexTypeDefault:
		return pg.CreateIndex(dbName, col, field)
	case database.IndexTypeNumber:
		return pg.createIndex(dbName, col, field, sbquery.TypeNumber)
	case database.IndexTypeBoolean:
		return pg.createIndex(dbName, col, field, sbquery.TypeBoolean)
	default:
		return fmt.Errorf("index type %q is not supported", typ)
	}
}

func (pg *PostgreSQL) createIndex(dbName, col, field string, typ sbquery.ValueType) error {
	if err := sbquery.ValidateField(field); err != nil {
		return err
	}

	cleanCol := model.CleanCollectionName(col)
	expr := fieldExpr(field, typ)
	indexName := sanitizeIndexPart(fmt.Sprintf("idx_%s_%s", cleanCol, field))
	if typ != sbquery.TypeDefault {
		indexName = sanitizeIndexPart(fmt.Sprintf("%s_%s", indexName, typ))
	}

	qry := `
		CREATE INDEX IF NOT EXISTS 
			{index} 
		ON {schema}.{col} 
		USING btree (({expr}))
	`

	qry = strings.ReplaceAll(qry, "{index}", indexName)
	qry = strings.ReplaceAll(qry, "{col}", cleanCol)
	qry = strings.ReplaceAll(qry, "{expr}", expr)
	qry = strings.ReplaceAll(qry, "{schema}", dbName)

	if _, err := pg.DB.Exec(qry); err != nil {
		return err
	}
	return nil
}

var indexNameRE = regexp.MustCompile(`[^A-Za-z0-9_]`)

func sanitizeIndexPart(s string) string {
	return indexNameRE.ReplaceAllString(s, "_")
}
