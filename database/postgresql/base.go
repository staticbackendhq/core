package postgresql

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/staticbackendhq/core/model"
)

const (
	FieldID        = "id"
	FieldAccountID = "accountId"
	FieldOwnerID   = "sb_ownerId"
	FieldCreated   = "sb_created"
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

func (pg *PostgreSQL) CreateDocument(auth model.Auth, dbName, col string, doc map[string]interface{}) (inserted map[string]interface{}, err error) {
	inserted = doc
	removeNotEditableFields(inserted)

	cleancol := model.CleanCollectionName(col)

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

		CREATE INDEX IF NOT EXISTS %s_acctid_idx ON %s.%s (account_id);			
	`, dbName, cleancol, dbName, dbName, cleancol, dbName, cleancol)

	if _, err = pg.DB.Exec(qry); err != nil {
		err = fmt.Errorf("error creating table: %w", err)
		return
	}

	var id string

	qry = fmt.Sprintf(`
		INSERT INTO %s.%s(account_id, owner_id, data, created)
		VALUES($1, $2, $3, $4)
		RETURNING id;
	`, dbName, model.CleanCollectionName(col))

	b, err := json.Marshal(doc)
	if err != nil {
		err = fmt.Errorf("error executing INSERT: %w", err)
		return
	}

	created := time.Now()
	err = pg.DB.QueryRow(qry, auth.AccountID, auth.UserID, b, created).Scan(&id)
	if err != nil {
		err = fmt.Errorf("error getting the new row ID: %w", err)
	}

	inserted[FieldID] = id
	inserted[FieldAccountID] = auth.AccountID
	inserted[FieldOwnerID] = auth.UserID
	inserted[FieldCreated] = created

	pg.PublishDocument(auth, dbName, "db-"+col, model.MsgTypeDBCreated, inserted)

	return
}

func (pg *PostgreSQL) BulkCreateDocument(auth model.Auth, dbName, col string, docs []interface{}) error {
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

func (pg *PostgreSQL) ListDocuments(auth model.Auth, dbName, col string, params model.ListParams) (result model.PagedResult, err error) {
	where := secureRead(auth, col)

	paging := setPaging(params)

	result.Page = params.Page
	result.Size = params.Size

	qry := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s.%s 
		%s
	`, dbName, model.CleanCollectionName(col), where)

	if err = pg.DB.QueryRow(qry, auth.AccountID, auth.UserID).Scan(&result.Total); err != nil {
		if !isTableExists(err) {
			return result, nil
		}
		return
	}

	qry = fmt.Sprintf(`
		SELECT * 
		FROM %s.%s 
		%s
		%s
	`, dbName, model.CleanCollectionName(col), where, paging)

	rows, err := pg.DB.Query(qry, auth.AccountID, auth.UserID)
	if err != nil {
		slog.Error("error in select", "error", err)
		return
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var doc Document
		if err = scanDocument(rows, &doc); err != nil {
			return
		}

		result.Results = append(result.Results, doc.Map())
	}

	err = rows.Err()
	return
}

func (pg *PostgreSQL) QueryDocuments(auth model.Auth, dbName, col string, filters map[string]interface{}, params model.ListParams) (result model.PagedResult, err error) {
	where := secureRead(auth, col)
	where, filterArgs := applyFilter(where, filters, 3)
	queryArgs := append([]any{auth.AccountID, auth.UserID}, filterArgs...)

	paging := setPaging(params)

	result.Page = params.Page
	result.Size = params.Size

	qry := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s.%s 
		%s
	`, dbName, model.CleanCollectionName(col), where)

	if err = pg.DB.QueryRow(qry, queryArgs...).Scan(&result.Total); err != nil {
		if !isTableExists(err) {
			return result, nil
		}
		return
	}

	qry = fmt.Sprintf(`
		SELECT * 
		FROM %s.%s 
		%s
		%s
	`, dbName, model.CleanCollectionName(col), where, paging)

	rows, err := pg.DB.Query(qry, queryArgs...)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var doc Document
		if err = scanDocument(rows, &doc); err != nil {
			return
		}

		result.Results = append(result.Results, doc.Map())
	}

	err = rows.Err()
	return
}

func (pg *PostgreSQL) GetDocumentByID(auth model.Auth, dbName, col, id string) (map[string]interface{}, error) {
	where := secureRead(auth, col)

	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s.%s 
		%s AND id = $3
	`, dbName, model.CleanCollectionName(col), where)

	row := pg.DB.QueryRow(qry, auth.AccountID, auth.UserID, id)

	var doc Document
	if err := scanDocument(row, &doc); err != nil {
		return nil, err
	}

	return doc.Map(), nil
}

func (pg *PostgreSQL) GetDocumentsByIDs(auth model.Auth, dbName, col string, ids []string) (docs []map[string]interface{}, err error) {
	where := secureRead(auth, col)

	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s.%s 
		%s AND id in ('%s'::uuid)
	`, dbName, model.CleanCollectionName(col), where, strings.Join(ids, "'::uuid,'"))

	rows, err := pg.DB.Query(qry, auth.AccountID, auth.UserID)
	if err != nil {
		return []map[string]interface{}{}, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var doc Document
		if err = scanDocument(rows, &doc); err != nil {
			return []map[string]interface{}{}, err
		}

		docs = append(docs, doc.Map())
	}

	return docs, nil
}

func (pg *PostgreSQL) UpdateDocument(auth model.Auth, dbName, col, id string, doc map[string]interface{}) (map[string]interface{}, error) {
	where := secureWrite(auth, col)
	removeNotEditableFields(doc)

	qry := fmt.Sprintf(`
		UPDATE %s.%s SET
			data = data || $4
		%s AND id = $3
	`, dbName, model.CleanCollectionName(col), where)

	b, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}

	if _, err := pg.DB.Exec(qry, auth.AccountID, auth.UserID, id, b); err != nil {
		return nil, err
	}

	updated, err := pg.GetDocumentByID(auth, dbName, col, id)
	if err != nil {
		return nil, err
	}

	pg.PublishDocument(auth, dbName, "db-"+col, model.MsgTypeDBUpdated, updated)

	return updated, nil
}

func (pg *PostgreSQL) UpdateDocuments(auth model.Auth, dbName, col string, filters map[string]interface{}, updateFields map[string]interface{}) (n int64, err error) {
	where := secureWrite(auth, col)
	where, filterArgs := applyFilter(where, filters, 3)
	queryArgs := append([]any{auth.AccountID, auth.UserID}, filterArgs...)
	removeNotEditableFields(updateFields)

	var ids []string
	qry := fmt.Sprintf(`
		SELECT id
		FROM %s.%s 
		%s
	`, dbName, model.CleanCollectionName(col), where)

	rows, err := pg.DB.Query(qry, queryArgs...)
	if err != nil {
		return
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			slog.Error("error occurred during scanning id for UpdateDocument event", "error", err)
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	qry = fmt.Sprintf(`
		UPDATE %s.%s SET
			data = data || $%d
		%s
	`, dbName, model.CleanCollectionName(col), len(queryArgs)+1, where)

	b, err := json.Marshal(updateFields)
	if err != nil {
		return 0, err
	}
	execArgs := append(queryArgs, b)
	res, err := pg.DB.Exec(qry, execArgs...)
	if err != nil {
		return 0, err
	}
	n, err = res.RowsAffected()
	if err != nil {
		return 0, err
	}

	go func() {
		docs, err := pg.GetDocumentsByIDs(auth, dbName, col, ids)
		if err != nil {
			slog.Error("the documents are not received for publishDocument event", "ids", ids, "error", err)
		}
		for _, doc := range docs {
			pg.PublishDocument(auth, dbName, "db-"+col, model.MsgTypeDBUpdated, doc)
		}
	}()
	return
}

func (pg *PostgreSQL) IncrementValue(auth model.Auth, dbName, col, id, field string, n int) error {
	where := secureWrite(auth, col)

	qry := fmt.Sprintf(`
		UPDATE %s.%s SET
		data = jsonb_set(data, '{%s}', (COALESCE(data->>'%s','0')::int + $4)::text::jsonb)
		%s AND id = $3
	`, dbName, model.CleanCollectionName(col), field, field, where)

	if _, err := pg.DB.Exec(qry, auth.AccountID, auth.UserID, id, n); err != nil {
		return err
	}

	updated, err := pg.GetDocumentByID(auth, dbName, col, id)
	if err != nil {
		return err
	}

	pg.PublishDocument(auth, dbName, "db-"+col, model.MsgTypeDBUpdated, updated)

	return nil
}

func (pg *PostgreSQL) DeleteDocument(auth model.Auth, dbName, col, id string) (int64, error) {
	where := secureWrite(auth, col)

	qry := fmt.Sprintf(`
		DELETE 
		FROM %s.%s 
		%s AND id = $3
	`, dbName, model.CleanCollectionName(col), where)

	res, err := pg.DB.Exec(qry, auth.AccountID, auth.UserID, id)
	if err != nil {
		return 0, err
	}

	pg.PublishDocument(auth, dbName, "db-"+col, model.MsgTypeDBDeleted, id)
	return res.RowsAffected()
}

func (pg *PostgreSQL) DeleteDocuments(auth model.Auth, dbName, col string, filters map[string]any) (n int64, err error) {
	where := secureWrite(auth, col)
	where, filterArgs := applyFilter(where, filters, 3)
	queryArgs := append([]any{auth.AccountID, auth.UserID}, filterArgs...)

	var ids []string
	qry := fmt.Sprintf(`
		SELECT id
		FROM %s.%s 
		%s
	`, dbName, model.CleanCollectionName(col), where)

	rows, err := pg.DB.Query(qry, queryArgs...)
	if err != nil {
		return
	}

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			slog.Error("error occurred during scanning id for DeleteDocuments event", "error", err)
			continue
		}

		ids = append(ids, id)
	}

	qry = fmt.Sprintf(`
		DELETE 
		FROM %s.%s 
		%s
	`, dbName, model.CleanCollectionName(col), where)

	res, err := pg.DB.Exec(qry, queryArgs...)
	if err != nil {
		return 0, err
	}

	go func() {
		for _, id := range ids {
			pg.PublishDocument(auth, dbName, "db-"+col, model.MsgTypeDBDeleted, id)
		}
	}()

	return res.RowsAffected()
}

func (pg *PostgreSQL) ListCollections(dbName string) (results []string, err error) {
	qry := fmt.Sprintf(`
		SELECT table_name FROM information_schema.tables WHERE table_schema='%s'
	`, strings.ToLower(dbName))

	rows, err := pg.DB.Query(qry)
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

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

func (doc Document) Map() map[string]interface{} {
	doc.Data[FieldID] = doc.ID
	doc.Data[FieldAccountID] = doc.AccountID
	doc.Data[FieldOwnerID] = doc.OwnerID
	doc.Data[FieldCreated] = doc.Created
	return doc.Data
}

func removeNotEditableFields(m map[string]any) {
	delete(m, FieldID)
	delete(m, FieldAccountID)
	delete(m, FieldOwnerID)
	delete(m, FieldCreated)
}

func isTableExists(err error) bool {
	if err, ok := err.(*pq.Error); ok {
		if err.Code.Name() == "undefined_table" {
			return false
		}
	}
	return true
}
