package postgresql

import (
	"errors"
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

func (pg *PostgreSQL) CreateDocument(auth internal.Auth, dbName, col string, doc map[string]interface{}) (inserted map[string]interface{}, err error) {
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

	err = pg.DB.QueryRow(qry, auth.AccountID, auth.UserID).Scan(&id)

	inserted[FieldID] = id
	inserted[FieldAccountID] = auth.AccountID

	return
}

func (pg *PostgreSQL) BulkCreateDocument(auth internal.Auth, dbName, col string, docs []interface{}) error {
	//TODO: Naive implementation, not sure if PostgreSQL
	// has a better way for bulk insert, but will suffice for now.
	for _, doc := range docs {
		d, ok := doc.(map[string]interface{})
		if !ok {
			return errors.New("unable to cast doc as map[string]interface{}")
		}

		if _, err := pg.CreateDocument(auth, dbName, col, d); err != nil {
			return err
		}
	}
	return nil
}

func (pg *PostgreSQL) ListDocuments(auth internal.Auth, dbName, col string, params internal.ListParams) (result internal.PagedResult, err error) {
	where := secureRead(auth, col)

	paging := setPaging(params)

	result.Page = params.Page
	result.Size = params.Size

	qry := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s.%s 
		%s
	`, dbName, col, where)

	if err = pg.DB.QueryRow(qry, auth.AccountID, auth.UserID).Scan(&result.Total); err != nil {
		return
	}

	qry = fmt.Sprintf(`
		SELECT * 
		FROM %s.%s 
		%s
		%s
	`, dbName, col, where, paging)

	rows, err := pg.DB.Query(qry, auth.AccountID, auth.UserID)
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

		result.Results = append(result.Results, doc.Data)
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
	`, dbName, col, where)

	row := pg.DB.QueryRow(qry, auth.AccountID, auth.UserID, id)

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

func (pg *PostgreSQL) DeleteDocument(auth internal.Auth, dbName, col, id string) (int64, error) {
	where := secureWrite(auth, col)

	qry := fmt.Sprintf(`
		DELETE 
		FROM %s.%s 
		%s AND id = $3
	`, dbName, col, where)

	res, err := pg.DB.Exec(qry, auth.AccountID, auth.UserID, id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (pg *PostgreSQL) ListCollections(dbName string) (results []string, err error) {
	qry := fmt.Sprintf(`
		SELECT table_name FROM information_schema.tables WHERE table_schema='%s'
	`, dbName)

	rows, err := pg.DB.Query(qry)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return
		}

		results = append(results, name)
	}

	err = rows.Err()
	return
}

func scanDocument(rows Scanner, doc *Document) error {
	return rows.Scan(
		&doc.ID,
		&doc.AccountID,
		&doc.OwnerID,
		&doc.Data,
	)
}
