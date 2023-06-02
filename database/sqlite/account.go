package sqlite

import (
	"fmt"

	"github.com/staticbackendhq/core/model"
)

func (sl *SQLite) FindUser(dbName, userID, token string) (tok model.User, err error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s_sb_tokens
	WHERE id = $1 AND token = $2
`, dbName)

	row := sl.DB.QueryRow(qry, userID, token)

	err = scanToken(row, &tok)
	return
}

func (sl *SQLite) FindRootUser(dbName, userID, accountID, token string) (tok model.User, err error) {
	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s_sb_tokens
		WHERE id = $1 AND account_id = $2 AND token = $3
`, dbName)

	row := sl.DB.QueryRow(qry, userID, accountID, token)

	err = scanToken(row, &tok)
	return
}

func (sl *SQLite) GetRootForBase(dbName string) (tok model.User, err error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s_sb_tokens
	WHERE role = 100
`, dbName)

	row := sl.DB.QueryRow(qry)

	err = scanToken(row, &tok)
	return
}

func (sl *SQLite) ListAccounts(dbName string) ([]model.Account, error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s_sb_accounts
	ORDER BY created DESC;
	`, dbName)

	rows, err := sl.DB.Query(qry)
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

func (sl *SQLite) ListUsers(dbName, accountID string) ([]model.User, error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s_sb_tokens
	WHERE account_id = $1
	`, dbName)

	rows, err := sl.DB.Query(qry, accountID)
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

func (sl *SQLite) FindUserByEmail(dbName, email string) (tok model.User, err error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s_sb_tokens
	WHERE email = $1
`, dbName)

	row := sl.DB.QueryRow(qry, email)

	err = scanToken(row, &tok)
	return
}

func (sl *SQLite) GetUserByID(dbName, accountID, userID string) (user model.User, err error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s_sb_tokens
	WHERE id = $1 AND account_id = $2;
`, dbName)

	row := sl.DB.QueryRow(qry, userID, accountID)

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
