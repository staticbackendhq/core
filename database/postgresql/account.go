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
	FROM %s.sb_accounts
	ORDER BY created DESC;
	`, dbName)

	rows, err := pg.DB.Query(qry)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

func (pg *PostgreSQL) ListUsers(dbName, accountID string, role ...int) ([]model.User, error) {
	qry := fmt.Sprintf(`
	SELECT id, account_id, token, email, password, role, reset_code, created
	FROM %s.sb_tokens
	WHERE account_id = $1
	UNION ALL
	SELECT t.id, au.account_id, au.token, au.email, t.password, au.role, t.reset_code, au.created
	FROM %s.sb_account_users au
	INNER JOIN %s.sb_tokens t ON t.id = au.user_id
	WHERE au.account_id = $1
	`, dbName, dbName, dbName)

	args := []any{accountID}
	if len(role) > 0 {
		qry = fmt.Sprintf(`
	SELECT id, account_id, token, email, password, role, reset_code, created
	FROM %s.sb_tokens
	WHERE account_id = $1 AND role = $2
	UNION ALL
	SELECT t.id, au.account_id, au.token, au.email, t.password, au.role, t.reset_code, au.created
	FROM %s.sb_account_users au
	INNER JOIN %s.sb_tokens t ON t.id = au.user_id
	WHERE au.account_id = $1 AND au.role = $2
	`, dbName, dbName, dbName)
		args = append(args, role[0])
	}

	rows, err := pg.DB.Query(qry, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

func (pg *PostgreSQL) GetUserByID(dbName, accountID, userID string) (user model.User, err error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s.sb_tokens
	WHERE id = $1 AND account_id = $2;
`, dbName)

	row := pg.DB.QueryRow(qry, userID, accountID)

	err = scanToken(row, &user)
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
