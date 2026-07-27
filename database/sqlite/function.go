package sqlite

import (
	"fmt"

	"github.com/lib/pq"
	"github.com/staticbackendhq/core/model"
)

func (sl *SQLite) AddFunction(dbName string, data model.ExecData) (id string, err error) {
	id = sl.NewID()

	qry := fmt.Sprintf(`
		INSERT INTO %s_sb_functions(id, function_name, trigger_topic, code, function_secrets, version, last_updated, last_run)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8);
	`, dbName)

	_, err = sl.DB.Exec(
		qry,
		id,
		data.FunctionName,
		data.TriggerTopic,
		data.Code,
		data.Secrets,
		data.Version,
		data.LastUpdated,
		data.LastRun,
	)
	return
}
func (sl *SQLite) UpdateFunction(dbName string, update model.FunctionUpdate) error {
	qry := fmt.Sprintf(`
		UPDATE %s_sb_functions SET
			code = $2,
			trigger_topic = $3,
			version = version + 1
		WHERE id = $1
	`, dbName)

	if update.UpdateSecrets {
		qry = fmt.Sprintf(`
			UPDATE %s_sb_functions SET
				code = $2,
				trigger_topic = $3,
				function_secrets = $4,
				version = version + 1
			WHERE id = $1
		`, dbName)

		if _, err := sl.DB.Exec(qry, update.ID, update.Code, update.TriggerTopic, update.Secrets); err != nil {
			return err
		}
		return nil
	}

	if _, err := sl.DB.Exec(qry, update.ID, update.Code, update.TriggerTopic); err != nil {
		return err
	}
	return nil
}

func (sl *SQLite) GetFunctionForExecution(dbName, name string) (result model.ExecData, err error) {
	qry := fmt.Sprintf(`
		SELECT id, function_name, trigger_topic, code, function_secrets, version, last_updated, last_run
		FROM %s_sb_functions 
		WHERE function_name = $1
	`, dbName)

	row := sl.DB.QueryRow(qry, name)

	err = scanExecData(row, &result)
	return
}

func (sl *SQLite) GetFunctionByID(dbName, id string) (result model.ExecData, err error) {
	qry := fmt.Sprintf(`
		SELECT id, function_name, trigger_topic, code, function_secrets, version, last_updated, last_run
		FROM %s_sb_functions 
		WHERE id = $1
	`, dbName)

	row := sl.DB.QueryRow(qry, id)

	err = scanExecData(row, &result)
	if err != nil {
		return
	}

	qry = fmt.Sprintf(`
		SELECT id, function_id, version, started, completed, success, output
		FROM %s_sb_function_logs 
		WHERE function_id = $1
		ORDER BY completed DESC
		LIMIT 50;
	`, dbName)

	rows, err := sl.DB.Query(qry, id)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var h model.ExecHistory
		if err = scanExecHistory(rows, &h); err != nil {
			return
		}

		result.History = append(result.History, h)
	}

	err = rows.Err()
	return
}

func (sl *SQLite) GetFunctionByName(dbName, name string) (result model.ExecData, err error) {
	qry := fmt.Sprintf(`
		SELECT id, function_name, trigger_topic, code, function_secrets, version, last_updated, last_run
		FROM %s_sb_functions 
		WHERE function_name = $1
	`, dbName)

	row := sl.DB.QueryRow(qry, name)

	err = scanExecData(row, &result)
	if err != nil {
		return
	}

	//TODO: this should be its own function and re-used from prev function
	qry = fmt.Sprintf(`
		SELECT id, function_id, version, started, completed, success, output
		FROM %s_sb_function_logs 
		WHERE function_id = $1
		ORDER BY completed DESC
		LIMIT 50;
	`, dbName)

	rows, err := sl.DB.Query(qry, result.ID)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var h model.ExecHistory
		if err = scanExecHistory(rows, &h); err != nil {
			return
		}

		result.History = append(result.History, h)
	}

	err = rows.Err()
	return
}

func (sl *SQLite) ListFunctions(dbName string) (results []model.ExecData, err error) {
	qry := fmt.Sprintf(`
		SELECT id, function_name, trigger_topic, code, function_secrets, version, last_updated, last_run
		FROM %s_sb_functions 
		ORDER BY last_updated DESC
	`, dbName)

	rows, err := sl.DB.Query(qry)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var ex model.ExecData
		if err = scanExecData(rows, &ex); err != nil {
			return
		}

		results = append(results, ex)
	}

	err = rows.Err()
	return
}

func (sl *SQLite) ListFunctionsByTrigger(dbName, trigger string) (results []model.ExecData, err error) {
	qry := fmt.Sprintf(`
		SELECT id, function_name, trigger_topic, code, function_secrets, version, last_updated, last_run
		FROM %s_sb_functions 
		WHERE trigger_topic = $1
		ORDER BY last_updated DESC
	`, dbName)

	rows, err := sl.DB.Query(qry, trigger)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var ex model.ExecData
		if err = scanExecData(rows, &ex); err != nil {
			return
		}

		results = append(results, ex)
	}

	err = rows.Err()
	return
}

func (sl *SQLite) DeleteFunction(dbName, name string) error {
	qry := fmt.Sprintf(`
		DELETE FROM %s_sb_functions
		WHERE function_name = $1
	`, dbName)

	if _, err := sl.DB.Exec(qry, name); err != nil {
		return err
	}
	return nil
}

func (sl *SQLite) RanFunction(dbName, id string, rh model.ExecHistory) error {
	qry := fmt.Sprintf(`
		UPDATE %s_sb_functions SET
			last_run = $2
		WHERE id = $1
	`, dbName)

	if _, err := sl.DB.Exec(qry, id, rh.Completed); err != nil {
		return err
	}

	qry = fmt.Sprintf(`
		INSERT INTO %s_sb_function_logs(id, function_id, version, started, completed, success, output)
		VALUES($1, $2, $3, $4, $5, $6, $7)
	`, dbName)

	newID := sl.NewID()

	_, err := sl.DB.Exec(
		qry,
		newID,
		id,
		rh.Version,
		rh.Started,
		rh.Completed,
		rh.Success,
		pq.Array(rh.Output),
	)
	if err != nil {
		return err
	}

	qry = fmt.Sprintf(`
		DELETE FROM %s_sb_function_logs
		WHERE function_id = $1 AND completed < $2
		`, dbName)

	_, err = sl.DB.Exec(qry, id, model.FunctionHistoryRetentionCutoff())

	return err
}

func scanExecData(rows Scanner, ex *model.ExecData) error {
	return rows.Scan(
		&ex.ID,
		&ex.FunctionName,
		&ex.TriggerTopic,
		&ex.Code,
		&ex.Secrets,
		&ex.Version,
		&ex.LastUpdated,
		&ex.LastRun,
	)
}

func scanExecHistory(rows Scanner, h *model.ExecHistory) error {
	return rows.Scan(
		&h.ID,
		&h.FunctionID,
		&h.Version,
		&h.Started,
		&h.Completed,
		&h.Success,
		pq.Array(&h.Output),
	)
}
