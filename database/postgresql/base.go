package postgresql

import (
	"fmt"

	"github.com/staticbackendhq/core/internal"
)

const (
	FieldID        = "id"
	FieldAccountID = "accountId"
)

type Document struct {
	ID        string
	AccountID string
	OwnerID   string
	Data      map[string]interface{}
}

func (pg *PostgreSQL) CreateDocument(auth Auth, dbName, col string, doc map[string]interface{}) (inserted map[string]interface{}, err error) {
	inserted = make(map[string]interface{})

	//TODO: find a good way to prevent doing the create
	// table if not exists each time

	qry := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s (
			id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
			account_id uuid REFERENCES %s.sb_accounts(id) ON DELETE CASCADE,
			owner_id uuid REFERENCES %s.sb_tokens(id) ON DELETE CASCADE,
			data jsonb
		)
	`, dbName, col, dbName)

	if _, err = pg.DB.Exec(qry); err != nil {
		return
	}

	var id string

	qry = fmt.Sprintf(`
		INSERT INTO %s.%s(account_id, owner_id, data)
		VALUES($1, $2, $3)
		RETURNING id;
	`, dbName, col)

	err = pg.DB.QueryRow(qry, auth.accountId, auth.UserID).Scan(&id)

	inserted[FieldID] = id
	inserted[FieldAccountID] = auth.accountId

	return
}

func (pg *PostgreSQL) ListDocuments(auth Auth, dbName, col string, params ListParams) (result internal.PagedResult, err error) {
	where := secureRead(auth, col)

	paging := setPaging(params)

	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s.%s 
		%s
		%s
	`, dbName, col, where, paging)

	rows, err := pg.DB.Query(qry, auth.accountId, auth.userID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var doc Document
		if err = scanDocument(rows, &doc); err != nil {
			return
		}

		doc.Data[FieldID] = doc.ID
		doc.Data[FieldAccountID] = doc.AccountID

		result = append(results, doc.Data)
	}

	err = rows.Err()
	return
}

/*func (pg *PostgreSQL) QueryDocuments(auth Auth, dbName, col string, filter map[string]interface{}, params ListParams) (internal.PagedResult, error) {
	return errors.New("not implemented")
}*/

func (pg *PostgreSQL) GetDocumentByID(auth internal.Auth, dbName, col, id string) (map[string]interface{}, error) {
	where := secureRead(auth, col)

	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s.%s 
		%s AND id = $3
	`, dbName, col)

	row := pg.DB.QueryRow(qry, auth.accountId, auth.userID, id)

	var doc Document
	if err := scanDocument(row, &doc); err != nil {
		return nil, err
	}

	doc.Data[FieldID] = doc.ID
	doc.Data[FieldAccountID] = doc.AccountID

	return doc.Data, nil
}

func (pg *PostgreSQL) UpdateDocument(auth internal.Auth, dbName, col, id string, doc map[string]interface{}) (map[string]interface{}, error) {
	where := secureWrite(auth, col)

	qry := fmt.Sprintf(`
		UPDATE %s.%s SET
			data = data || $4
		%s AND id = $3
	`, dbName, col, where)

	if _, err := pg.DB.Exec(qry, auth.AccountID, auth.UserID, id, doc); err != nil {
		return nil, err
	}

	return pg.GetDocumentByID(auth, dbName, col, id)
}

func scanDocument(rows Scanner, doc *Document) error {
	return rows.Scan(
		&doc.ID,
		&doc.AccountID,
		&doc.OwnerID,
		&doc.Data,
	)
}
