package postgresql

import "fmt"

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

func (pg *PostgreSQL) GetForms(dbName string) ([]string, error) {
	//rendula
}
