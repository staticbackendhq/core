package sqlite

import (
	"fmt"

	"github.com/staticbackendhq/core/model"
)

func (sl *SQLite) AddFile(dbName string, f model.File) (id string, err error) {
	id = sl.NewID()

	qry := fmt.Sprintf(`
		INSERT INTO %s_sb_files(id, account_id, key, url, size, uploaded)
		VALUES($1, $2, $3, $4, $5, $6);
	`, dbName)

	_, err = sl.DB.Exec(
		qry,
		id,
		f.AccountID,
		f.Key,
		f.URL,
		f.Size,
		f.Uploaded,
	)
	return
}

func (sl *SQLite) GetFileByID(dbName, fileID string) (f model.File, err error) {
	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s_sb_files 
		WHERE id = $1
	`, dbName)

	row := sl.DB.QueryRow(qry, fileID)

	err = scanFile(row, &f)
	return

}

func (sl *SQLite) DeleteFile(dbName, fileID string) error {
	qry := fmt.Sprintf(`
		DELETE FROM %s_sb_files 
		WHERE id = $1;
	`, dbName)

	if _, err := sl.DB.Exec(qry, fileID); err != nil {
		return err
	}
	return nil
}

func (sl *SQLite) ListAllFiles(dbName, accountID string) (results []model.File, err error) {
	where := "WHERE account_id = $1"

	// if no accountID is specify, the admin UI
	// display all files uploaded.
	if len(accountID) == 0 {
		where = "WHERE $1 = $1"
	}

	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s_sb_files
		%s
	`, dbName, where)

	rows, err := sl.DB.Query(qry, accountID)
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
