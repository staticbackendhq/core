package postgresql

import (
	"fmt"
	"time"

	"github.com/staticbackendhq/core/model"
)

func (pg *PostgreSQL) CreateAccount(dbName, email string) (id string, err error) {
	qry := fmt.Sprintf(`
		INSERT INTO %s.sb_accounts(email, created)
		VALUES($1, $2)
		RETURNING id;
	`, dbName)

	err = pg.DB.QueryRow(qry, email, time.Now()).Scan(&id)
	return
}

func (pg *PostgreSQL) CreateUser(dbName string, tok model.User) (id string, err error) {
	qry := fmt.Sprintf(`
		INSERT INTO %s.sb_tokens(account_id, email, password, token, role, reset_code, created)
		VALUES($1, $2, $3, $4, $5, $6, $7)
		RETURNING id;
	`, dbName)

	err = pg.DB.QueryRow(
		qry,
		tok.AccountID,
		tok.Email,
		tok.Password,
		tok.Token,
		tok.Role,
		tok.ResetCode,
		tok.Created,
	).Scan(&id)
	return
}

func (pg *PostgreSQL) UserEmailExists(dbName, email string) (exists bool, err error) {
	qry := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s.sb_tokens
		WHERE email = $1;
	`, dbName)

	var count int
	err = pg.DB.QueryRow(qry, email).Scan(&count)

	exists = count > 0
	return
}

func (pg *PostgreSQL) SetUserRole(dbName, email string, role int) error {
	qry := fmt.Sprintf(`
		UPDATE %s.sb_tokens SET role = $2
		WHERE email = $1;
	`, dbName)

	if _, err := pg.DB.Exec(qry, email, role); err != nil {
		return err
	}
	return nil
}

func (pg *PostgreSQL) UserSetPassword(dbName, tokenID, password string) error {
	qry := fmt.Sprintf(`
		UPDATE %s.sb_tokens SET password = $2
		WHERE id = $1;
	`, dbName)

	if _, err := pg.DB.Exec(qry, tokenID, password); err != nil {
		return err
	}
	return nil
}

func (pg *PostgreSQL) GetFirstUserFromAccountID(dbName, accountID string) (tok model.User, err error) {
	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s.sb_tokens 
		WHERE account_id = $1
		ORDER BY created ASC
		LIMIT 1
	`, dbName)

	row := pg.DB.QueryRow(qry, accountID)

	err = scanToken(row, &tok)
	return
}

func (pg *PostgreSQL) SetPasswordResetCode(dbName, userID, code string) error {
	qry := fmt.Sprintf(`
	UPDATE %s.sb_tokens SET
		reset_code = $2
	WHERE id = $1
`, dbName)

	_, err := pg.DB.Exec(qry, userID, code)
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
	`, dbName)

	if _, err := pg.DB.Exec(qry, email, code, password); err != nil {
		return err
	}
	return nil
}
