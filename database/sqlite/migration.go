package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"
)

func migrate(db *sql.DB) error {
	if err := ensureSchema(db); err != nil {
		return err
	}

	if err := ensureMigrationTable(db); err != nil {
		return err
	}

	if err := ensureVersion(db); err != nil {
		return err
	}
	return nil
}

func ensureSchema(db *sql.DB) error {
	var schema string
	db.QueryRow(`
		SELECT name 
		FROM sqlite_master 
		WHERE type='table' AND name='sb_customers';
	`).Scan(&schema)

	if len(schema) == 0 {
		// the bootstrap script has not been executed yet.
		b, err := fs.ReadFile(migrationFS, "sql/0001_bootstrap_db.sql")
		if err != nil {
			return err
		}

		if _, err := db.Exec(string(b)); err != nil {
			return err
		}
	}
	return nil
}

func ensureMigrationTable(db *sql.DB) error {
	var table string
	db.QueryRow(`
		SELECT name 
		FROM sqlite_master 
		WHERE type='table' AND name='sb_migrations';
	`).Scan(&table)

	if len(table) == 0 {
		// the migrations table does not exists, we create it.
		b, err := fs.ReadFile(migrationFS, "sql/0002_add_migrations_table.sql")
		if err != nil {
			return err
		}

		if _, err := db.Exec(string(b)); err != nil {
			return err
		}
	}
	return nil
}

func ensureVersion(db *sql.DB) error {
	dbVersion, err := getDBLastMigration(db)
	if err != nil {
		return err
	}

	last, err := getLastMigration()
	if err != nil {
		return err
	}

	// if both version are the same, no migration needed
	if last == dbVersion {
		return nil
	}

	for i := dbVersion + 1; i <= last; i++ {
		prefix := fmt.Sprintf("%04d", i)
		if err := up(db, prefix, i); err != nil {
			return err
		}
	}
	return nil
}

func getDBLastMigration(db *sql.DB) (dbVersion int, err error) {
	err = db.QueryRow(`
		SELECT MAX(version)
		FROM sb_migrations 
`).Scan(&dbVersion)

	return
}

func getLastMigration() (last int, err error) {
	files, err := fs.ReadDir(migrationFS, "sql")
	if err != nil {
		return
	}

	for _, file := range files {
		i, err := strconv.Atoi(file.Name()[:4])
		if err != nil {
			return 0, err
		}

		if last < i {
			last = i
		}
	}

	return
}

func up(db *sql.DB, prefix string, version int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	var migFile string
	files, err := fs.ReadDir(migrationFS, "sql")
	if err != nil {
		return err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), prefix) {
			migFile = file.Name()
			break
		}
	}

	if len(migFile) == 0 {
		return errors.New("unable to find migration file starting with: " + prefix)
	}

	b, err := fs.ReadFile(migrationFS, path.Join("sql", migFile))
	if err != nil {
		return err
	}

	if _, err := tx.Exec(string(b)); err != nil {
		return err
	}

	qry := `
		INSERT INTO sb_migrations(id, version, files)
		VALUES($1, $2, $3);
	`

	if _, err := tx.Exec(qry, prefix, version, migFile); err != nil {
		return err
	}

	return tx.Commit()
}
