package postgresql

import (
	"fmt"

	"github.com/staticbackendhq/core/model"
)

func (pg *PostgreSQL) AddFile(dbName string, f model.File) (id string, err error) {
	qry := fmt.Sprintf(`
		INSERT INTO %s.sb_files(account_id, key, url, size, uploaded)
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

func (pg *PostgreSQL) GetFileByID(dbName, fileID string) (f model.File, err error) {
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

func (pg *PostgreSQL) ListAllFiles(dbName, accountID string) (results []model.File, err error) {
	where := "WHERE account_id = $1"

	// if no accountID is specify, the admin UI
	// display all files uploaded.
	if len(accountID) == 0 {
		where = "WHERE $1 = $1"
	}

	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s.sb_files
		%s
	`, dbName, where)

	rows, err := pg.DB.Query(qry, accountID)
	if err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		var f model.File
		if err = scanFile(rows, &f); err != nil {
			return
		}

		results = append(results, f)
	}

	err = rows.Err()

	return
}

func scanFile(rows Scanner, f *model.File) error {
	return rows.Scan(
		&f.ID,
		&f.AccountID,
		&f.Key,
		&f.URL,
		&f.Size,
		&f.Uploaded,
	)
}
