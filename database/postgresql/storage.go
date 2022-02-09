package postgresql

import (
	"fmt"

	"github.com/staticbackendhq/core/internal"
)

func (pg *PostgreSQL) AddFile(dbName string, f internal.File) (id string, err error) {
	qry := fmt.Sprintf(`
		INSERT INTO %s.sb_files(account_id, store_key, url, size, created)
		VALUES($1, $2, $3, $4, $5)
		RETURNING id;
	`, dbName)

	err = pg.DB.QueryRow(
		qry,
		f.AccountID,
		f.Key,
		f.URL,
		f.Size,
		f.Uploaded,
	).Scan(&id)
	return
}

func (pg *PostgreSQL) GetFileByID(dbName, fileID string) (f internal.File, err error) {
	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s.sb_files 
		WHERE id = $1
	`, dbName)

	row := pg.DB.QueryRow(qry, fileID)

	err = scanFile(row, &f)
	return

}

func (pg *PostgreSQL) DeleteFile(dbName, fileID string) error {
	qry := fmt.Sprintf(`
		DELETE FROM %s.sb_files 
		WHERE id = $1
	`, dbName)

	if _, err := pg.DB.Exec(qry, fileID); err != nil {
		return err
	}
	return nil
}

func scanFile(rows Scanner, f *internal.File) error {
	return rows.Scan(
		&f.ID,
		&f.AccountID,
		&f.Key,
		&f.URL,
		&f.Size,
		&f.Uploaded,
	)
}
