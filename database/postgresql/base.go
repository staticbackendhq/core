package postgresql

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/staticbackendhq/core/internal"
)

const (
	FieldID        = "id"
	FieldAccountID = "accountId"
	FieldFormName  = "sb_form"
)

type JSONB map[string]interface{}

type Document struct {
	ID        string
	AccountID string
	OwnerID   string
	Data      JSONB
	Created   time.Time
}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &j)
}

func (pg *PostgreSQL) CreateDocument(auth internal.Auth, dbName, col string, doc map[string]interface{}) (inserted map[string]interface{}, err error) {
	inserted = doc

	//TODO: find a good way to prevent doing the create
	// table if not exists each time

	qry := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.%s (
			id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
			account_id uuid REFERENCES %s.sb_accounts(id) ON DELETE CASCADE,
			owner_id uuid REFERENCES %s.sb_tokens(id) ON DELETE CASCADE,
			data jsonb NOT NULL,
			created timestamp NOT NULL
		);
	`, dbName, col, dbName, dbName)

	if _, err = pg.DB.Exec(qry); err != nil {
		return
	}

	var id string

	qry = fmt.Sprintf(`
		INSERT INTO %s.%s(account_id, owner_id, data, created)
		VALUES($1, $2, $3, $4)
		RETURNING id;
	`, dbName, col)

	b, err := json.Marshal(doc)
	if err != nil {
		return
	}

	err = pg.DB.QueryRow(qry, auth.AccountID, auth.UserID, b, time.Now()).Scan(&id)

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
		fmt.Println("error in count")
		fmt.Println(qry)
		return
	}

	qry = fmt.Sprintf(`
		-- $1: account_id
		-- $2: user_id
		SELECT * 
		FROM %s.%s 
		%s
		%s
	`, dbName, col, where, paging)

	rows, err := pg.DB.Query(qry, auth.AccountID, auth.UserID)
	if err != nil {
		fmt.Println("error in select")
		fmt.Println(qry)
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

func (pg *PostgreSQL) QueryDocuments(auth internal.Auth, dbName, col string, filters map[string]interface{}, params internal.ListParams) (result internal.PagedResult, err error) {
	where := secureRead(auth, col)
	where = applyFilter(where, filters)

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

func (pg *PostgreSQL) IncrementValue(auth internal.Auth, dbName, col, id, field string, n int) error {
	where := secureWrite(auth, col)

	qry := fmt.Sprintf(`
		UPDATE %s.%s SET
			%s = %s + $4
		%s AND id = $3
	`, dbName, col, field, field, where)

	if _, err := pg.DB.Exec(qry, auth.AccountID, auth.UserID, id, n); err != nil {
		return err
	}
	return nil
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
		&doc.Created,
	)
}
