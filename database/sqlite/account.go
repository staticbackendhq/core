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
		FROM %s.sb_tokens
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
