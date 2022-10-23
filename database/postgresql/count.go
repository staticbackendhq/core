package postgresql

import (
	"fmt"

	"github.com/staticbackendhq/core/model"
)

func (pg *PostgreSQL) Count(auth model.Auth, dbName, col string, filters map[string]interface{}) (count int64, err error) {
	where := secureRead(auth, col)
	where = applyFilter(where, filters)

	query := fmt.Sprintf(`
    SELECT COUNT(*)
    FROM %s.%s
    %s;
    `, dbName, model.CleanCollectionName(col), where)

	err = pg.DB.QueryRow(query, auth.AccountID, auth.UserID).Scan(&count)
	if err != nil {
		return -1, err
	}

	return count, nil
}
