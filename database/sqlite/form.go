package sqlite

import (
	"fmt"
	"time"
)

type FormData struct {
	ID      string
	Name    string
	Data    JSON
	Created time.Time
}

func (sl *SQLite) AddFormSubmission(dbName, form string, doc map[string]interface{}) error {
	var jsonb JSON = doc

	qry := fmt.Sprintf(`
		INSERT INTO %s_sb_forms(name, data, created)
		VALUES($1, $2, $3)
	`, dbName)

	if _, err := sl.DB.Exec(qry, form, jsonb, time.Now()); err != nil {
		return err
	}
	return nil
}

func (sl *SQLite) ListFormSubmissions(dbName, name string) (results []map[string]interface{}, err error) {
	where := "WHERE $1=$1"
	if len(name) > 0 {
		where = "WHERE name = $1"
	}

	qry := fmt.Sprintf(`
		SELECT data  
		FROM %s_sb_forms 
		%s
		ORDER BY created DESC
		LIMIT 100;
	`, dbName, where)

	rows, err := sl.DB.Query(qry, name)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var data JSON
		if err = rows.Scan(&data); err != nil {
			return
		}

		results = append(results, data)
	}

	err = rows.Err()
	return
}

func (sl *SQLite) GetForms(dbName string) (results []string, err error) {
	qry := fmt.Sprintf(`
		SELECT name 
		FROM %s_sb_forms 
		GROUP BY name
	`, dbName)

	rows, err := sl.DB.Query(qry)
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
