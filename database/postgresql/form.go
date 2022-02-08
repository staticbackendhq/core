package postgresql

import (
	"fmt"
	"time"
)

type FormData struct {
	ID        string
	AccountID string
	Name      string
	Date      map[string]interface{}
	Created   time.Time
}

func (pg *PostgreSQL) AddFormSubmission(dbName, form string, doc map[string]interface{}) error {
	qry := fmt.Sprintf(`
		INSERT INTO %s.sb_forms(name, data, created)
		VALUES($1, $2, $3)
	`, dbName)

	if _, err := pg.DB.Exec(qry, form, doc, time.Now()); err != nil {
		return err
	}
	return nil
}

func (pg *PostgreSQL) ListFormSubmissions(dbName, name string) (results []map[string]interface{}, err error) {
	where := ""
	if len(name) > 0 {
		where = "WHERE name = $2"
	}

	qry := fmt.Sprintf(`
		SELECT data  
		FROM %s.sb_forms 
		%s
		ORDER BY created DESC
		LIMIT 100;
	`, dbName, where)

	rows, err := pg.DB.Query(qry, name)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var data map[string]interface{}
		if err = rows.Scan(&data); err != nil {
			return
		}

		results = append(results, data)
	}

	err = rows.Err()
	return
}

func (pg *PostgreSQL) GetForms(dbName string) (results []string, err error) {
	qry := fmt.Sprintf(`
		SELECT name 
		FROM %s.sb_forms 
		GROUP BY name
	`, dbName)

	rows, err := pg.DB.Query(qry)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return
		}

		results = append(results, name)
	}

	err = rows.Err()
	return
}
