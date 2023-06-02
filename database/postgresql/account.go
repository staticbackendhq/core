package postgresql

import (
	"fmt"

	"github.com/staticbackendhq/core/model"
)

func (pg *PostgreSQL) FindUser(dbName, userID, token string) (tok model.User, err error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s.sb_tokens
	WHERE id = $1 AND token = $2
`, dbName)

	row := pg.DB.QueryRow(qry, userID, token)

	err = scanToken(row, &tok)
	return
}

func (pg *PostgreSQL) FindRootUser(dbName, userID, accountID, token string) (tok model.User, err error) {
	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s.sb_tokens
		WHERE id = $1 AND account_id = $2 AND token = $3
`, dbName)

	row := pg.DB.QueryRow(qry, userID, accountID, token)

	err = scanToken(row, &tok)
	return
}

func (pg *PostgreSQL) GetRootForBase(dbName string) (tok model.User, err error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s.sb_tokens
	WHERE role = 100
`, dbName)

	row := pg.DB.QueryRow(qry)

	err = scanToken(row, &tok)
	return
}

func (pg *PostgreSQL) ListAccounts(dbName string) ([]model.Account, error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s_sb.accounts
	ORDER BY created DESC;
	`, dbName)

	rows, err := pg.DB.Query(qry)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.Account
	for rows.Next() {
		var a model.Account
		if err = scanAccount(rows, &a); err != nil {
			return nil, err
		}

		list = append(list, a)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (pg *PostgreSQL) ListUsers(dbName, accountID string) ([]model.User, error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s.sb_tokens
	WHERE account_id = $1;
	`, dbName)

	rows, err := pg.DB.Query(qry, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.User
	for rows.Next() {
		var u model.User
		if err = scanToken(rows, &u); err != nil {
			return nil, err
		}

		list = append(list, u)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (pg *PostgreSQL) FindUserByEmail(dbName, email string) (tok model.User, err error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s.sb_tokens
	WHERE email = $1
`, dbName)

	row := pg.DB.QueryRow(qry, email)

	err = scanToken(row, &tok)
	return
}

func scanToken(rows Scanner, tok *model.User) error {
	return rows.Scan(
		&tok.ID,
		&tok.AccountID,
		&tok.Token,
		&tok.Email,
		&tok.Password,
		&tok.Role,
		&tok.ResetCode,
		&tok.Created,
	)
}

func scanAccount(rows Scanner, a *model.Account) error {
	return rows.Scan(
		&a.ID,
		&a.Email,
		&a.Created,
	)
}
