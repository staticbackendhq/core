package postgresql

import (
	"fmt"

	"github.com/staticbackendhq/core/internal"
)

func (pg *PostgreSQL) FindToken(dbName, tokenID, token string) (tok internal.Token, err error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s.sb_tokens
	WHERE id = $1 AND token = $2
`, dbName)

	row := pg.DB.QueryRow(qry, tokenID, token)

	err = scanToken(row, &tok)
	return
}

func (pg *PostgreSQL) FindRootToken(dbName, tokenID, accountID, token string) (tok internal.Token, err error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s.sb_tokens
	WHERE id = $1 AND account_id = &2 AND token = $3
`, dbName)

	row := pg.DB.QueryRow(qry, tokenID, accountID, token)

	err = scanToken(row, &tok)
	return
}

func (pg *PostgreSQL) GetRootForBase(dbName string) (tok internal.Token, err error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s.sb_tokens
	WHERE role = 100
`, dbName)

	row := pg.DB.QueryRow(qry)

	err = scanToken(row, &tok)
	return
}

func (pg *PostgreSQL) FindTokenByEmail(dbName, email string) (tok internal.Token, err error) {
	qry := fmt.Sprintf(`
	SELECT * 
	FROM %s.sb_tokens
	WHERE email = $1
`, dbName)

	row := pg.DB.QueryRow(qry, email)

	err = scanToken(row, &tok)
	return
}

func (pg *PostgreSQL) SetPasswordResetCode(dbName, tokenID, code string) error {
	qry := fmt.Sprintf(`
	UPDATE %s.sb_tokens SET
		reset_code = $2
	WHERE id = $1
`, dbName)

	_, err := pg.DB.Exec(qry, tokenID, code)
	if err != nil {
		return err
	}
	return nil
}

func (pg *PostgreSQL) ResetPassword(dbName, email, code, password string) error {
	qry := fmt.Sprintf(`
		UPDATE %s.sb_tokens SET
			password = $3
		WHERE email = $1 AND reset_code = $2
	`)

	if _, err := pg.DB.Exec(qry, email, code, password); err != nil {
		return err
	}
	return nil
}

func scanToken(rows Scanner, tok *internal.Token) error {
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
