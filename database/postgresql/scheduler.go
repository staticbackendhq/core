package postgresql

import (
	"fmt"

	"github.com/staticbackendhq/core/model"
)

func (pg *PostgreSQL) ListTasks() (results []model.Task, err error) {
	bases, err := pg.ListDatabases()
	if err != nil {
		return
	}

	for _, base := range bases {
		tasks, err := pg.ListTasksByBase(base.Name)
		if err != nil {
			return results, err
		}

		results = append(results, tasks...)
	}

	return
}

func (pg *PostgreSQL) ListTasksByBase(dbName string) (results []model.Task, err error) {
	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s.sb_tasks 
	`, dbName)

	rows, err := pg.DB.Query(qry)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var t model.Task
		if err = scanTask(rows, &t); err != nil {
			return
		}

		results = append(results, t)
	}

	err = rows.Err()
	return
}

func scanTask(rows Scanner, t *model.Task) error {
	return rows.Scan(
		&t.ID,
		&t.Name,
		&t.Type,
		&t.Value,
		&t.Meta,
		&t.Interval,
		&t.LastRun,
	)
}
