package sqlite

import (
	"fmt"

	"github.com/staticbackendhq/core/model"
)

func (sl *SQLite) ListTasks() (results []model.Task, err error) {
	bases, err := sl.ListDatabases()
	if err != nil {
		return
	}

	for _, base := range bases {
		tasks, err := sl.ListTasksByBase(base.Name)
		if err != nil {
			return results, err
		}

		results = append(results, tasks...)
	}

	return
}

func (sl *SQLite) ListTasksByBase(dbName string) (results []model.Task, err error) {
	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s_sb_tasks 
	`, dbName)

	rows, err := sl.DB.Query(qry)
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

func (sl *SQLite) AddTask(dbName string, task model.Task) (id string, err error) {
	qry := fmt.Sprintf(`
	INSERT INTO %s_sb_tasks(id, name, type, value, meta, interval, last_run)
	VALUES($1, $2, $3, $4, $5, $6, $7);
	`, dbName)

	id = sl.NewID()

	_, err = sl.DB.Exec(
		qry,
		id,
		task.Name,
		task.Type,
		task.Value,
		task.Meta,
		task.Interval,
		task.LastRun,
	)
	return
}

func (sl *SQLite) DeleteTask(dbName, id string) error {
	qry := fmt.Sprintf(`
	DELETE FROM %s_sb_tasks
	WHERE id = $1;
	`, dbName)

	if _, err := sl.DB.Exec(qry, id); err != nil {
		return err
	}
	return nil
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
