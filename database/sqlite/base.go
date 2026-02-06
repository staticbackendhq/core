package sqlite

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/staticbackendhq/core/model"
)

const (
	FieldID        = "id"
	FieldAccountID = "accountId"
	FieldFormName  = "sb_form"
)

type JSON map[string]interface{}

type Document struct {
	ID        string
	AccountID string
	OwnerID   string
	Data      JSON
	Created   time.Time
}

func (j JSON) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSON) Scan(value interface{}) error {
	var b []byte
	switch v := value.(type) {
	case []uint8:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &j)
}

func (sl *SQLite) CreateDocument(auth model.Auth, dbName, col string, doc map[string]interface{}) (inserted map[string]interface{}, err error) {
	inserted = doc

	cleancol := model.CleanCollectionName(col)

	//TODO: find a good way to prevent doing the create
	// table if not exists each time

	// for SQLite, this seems to cause issue with tests
	// so I'm using a map to hold if the collection was already
	// created

	m := &sync.RWMutex{}
	m.Lock()
	defer m.Unlock()

	if _, ok := sl.collections[col]; !ok {
		qry := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s_%s (
			id TEXT PRIMARY KEY,
			account_id TEXT REFERENCES %s_sb_accounts(id) ON DELETE CASCADE,
			owner_id TEXT REFERENCES %s_sb_tokens(id) ON DELETE CASCADE,
			data JSON NOT NULL,
			created timestamp NOT NULL
		);

		CREATE INDEX IF NOT EXISTS %s_%s_acctid_idx ON %s_%s (account_id);			
	`, dbName, cleancol, dbName, dbName, dbName, cleancol, dbName, cleancol)

		if _, err = sl.DB.Exec(qry); err != nil {
			err = fmt.Errorf("error creating table: %w", err)
			return
		}

		sl.collections[col] = true
	}

	id := sl.NewID()

	qry := fmt.Sprintf(`
		INSERT INTO %s_%s(id, account_id, owner_id, data, created)
		VALUES($1, $2, $3, $4, $5);
	`, dbName, model.CleanCollectionName(col))

	b, err := json.Marshal(doc)
	if err != nil {
		err = fmt.Errorf("error executing INSERT: %w", err)
		return
	}

	// TODO: sqlite BUSY error in unit test
	time.Sleep(10 * time.Millisecond)

	_, err = sl.DB.Exec(qry, id, auth.AccountID, auth.UserID, b, time.Now())
	if err != nil {
		err = fmt.Errorf("error getting the new row ID: %w", err)
	}

	inserted[FieldID] = id
	inserted[FieldAccountID] = auth.AccountID

	sl.PublishDocument(auth, dbName, "db-"+col, model.MsgTypeDBCreated, inserted)

	return
}

func (sl *SQLite) BulkCreateDocument(auth model.Auth, dbName, col string, docs []interface{}) error {
	//TODO: Naive implementation, not sure if SQLite
	// has a better way for bulk insert, but will suffice for now.
	for _, doc := range docs {
		d, ok := doc.(map[string]interface{})
		if !ok {
			return errors.New("unable to cast doc as map[string]interface{}")
		}

		if _, err := sl.CreateDocument(auth, dbName, col, d); err != nil {
			return err
		}
	}
	return nil
}

func (sl *SQLite) ListDocuments(auth model.Auth, dbName, col string, params model.ListParams) (result model.PagedResult, err error) {
	where := secureRead(auth, col)

	paging := setPaging(params)

	result.Page = params.Page
	result.Size = params.Size

	qry := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s_%s 
		%s
	`, dbName, model.CleanCollectionName(col), where)

	if err = sl.DB.QueryRow(qry, auth.AccountID, auth.UserID).Scan(&result.Total); err != nil {
		if !isTableExists(err) {
			return result, nil
		}
		return
	}

	qry = fmt.Sprintf(`
		SELECT * 
		FROM %s_%s 
		%s
		%s
	`, dbName, model.CleanCollectionName(col), where, paging)

	rows, err := sl.DB.Query(qry, auth.AccountID, auth.UserID)
	if err != nil {
		sl.log.Error().Err(err).Msg("error in select")
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

func (sl *SQLite) QueryDocuments(auth model.Auth, dbName, col string, filters map[string]interface{}, params model.ListParams) (result model.PagedResult, err error) {
	where := secureRead(auth, col)
	where = applyFilter(where, filters)

	paging := setPaging(params)

	result.Page = params.Page
	result.Size = params.Size

	qry := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s_%s 
		%s
	`, dbName, model.CleanCollectionName(col), where)

	if err = sl.DB.QueryRow(qry, auth.AccountID, auth.UserID).Scan(&result.Total); err != nil {
		if !isTableExists(err) {
			return result, nil
		}
		return
	}

	qry = fmt.Sprintf(`
		SELECT * 
		FROM %s_%s 
		%s
		%s
	`, dbName, model.CleanCollectionName(col), where, paging)

	rows, err := sl.DB.Query(qry, auth.AccountID, auth.UserID)
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

func (sl *SQLite) GetDocumentByID(auth model.Auth, dbName, col, id string) (map[string]interface{}, error) {
	where := secureRead(auth, col)

	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s_%s 
		%s AND id = $3
	`, dbName, model.CleanCollectionName(col), where)

	row := sl.DB.QueryRow(qry, auth.AccountID, auth.UserID, id)

	var doc Document
	if err := scanDocument(row, &doc); err != nil {
		return nil, err
	}

	doc.Data[FieldID] = doc.ID
	doc.Data[FieldAccountID] = doc.AccountID

	return doc.Data, nil
}

func (sl *SQLite) GetDocumentsByIDs(auth model.Auth, dbName, col string, ids []string) (docs []map[string]interface{}, err error) {
	where := secureRead(auth, col)

	qry := fmt.Sprintf(`
		SELECT * 
		FROM %s_%s 
		%s AND id in ('%s')
	`, dbName, model.CleanCollectionName(col), where, strings.Join(ids, "','"))

	rows, err := sl.DB.Query(qry, auth.AccountID, auth.UserID)
	if err != nil {
		return []map[string]interface{}{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var doc Document
		if err = scanDocument(rows, &doc); err != nil {
			return []map[string]interface{}{}, err
		}

		doc.Data[FieldID] = doc.ID
		doc.Data[FieldAccountID] = doc.AccountID
		docs = append(docs, doc.Data)
	}

	return docs, nil
}

func (sl *SQLite) UpdateDocument(auth model.Auth, dbName, col, id string, doc map[string]interface{}) (map[string]interface{}, error) {
	orig, err := sl.GetDocumentByID(auth, dbName, col, id)
	if err != nil {
		return nil, err
	}

	for key, val := range doc {
		orig[key] = val
	}

	where := secureWrite(auth, col)

	qry := fmt.Sprintf(`
		UPDATE %s_%s SET
			data = json($4)
		%s AND id = $3
	`, dbName, model.CleanCollectionName(col), where)

	b, err := json.Marshal(orig)
	if err != nil {
		return nil, err
	}

	if _, err := sl.DB.Exec(qry, auth.AccountID, auth.UserID, id, string(b)); err != nil {
		return nil, err
	}

	updated, err := sl.GetDocumentByID(auth, dbName, col, id)
	if err != nil {
		fmt.Println("DEBUG: in getbyid", err)
		return nil, err
	}

	sl.PublishDocument(auth, dbName, "db-"+col, model.MsgTypeDBUpdated, updated)

	return updated, nil
}

func (sl *SQLite) UpdateDocuments(auth model.Auth, dbName, col string, filters map[string]interface{}, updateFields map[string]interface{}) (n int64, err error) {
	where := secureWrite(auth, col)
	where = applyFilter(where, filters)

	var ids []string
	qry := fmt.Sprintf(`
		SELECT id
		FROM %s_%s 
		%s
	`, dbName, model.CleanCollectionName(col), where)

	rows, err := sl.DB.Query(qry, auth.AccountID, auth.UserID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			sl.log.Error().Err(err).Msg("error occurred during scanning id for UpdateDocument event")
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	qry = fmt.Sprintf(`
		UPDATE %s_%s SET
			data = json($3)
		%s
	`, dbName, model.CleanCollectionName(col), where)

	b, err := json.Marshal(updateFields)
	if err != nil {
		return 0, err
	}
	res, err := sl.DB.Exec(qry, auth.AccountID, auth.UserID, string(b))
	if err != nil {
		return 0, err
	}
	n, err = res.RowsAffected()
	if err != nil {
		return 0, err
	}

	go func() {
		docs, err := sl.GetDocumentsByIDs(auth, dbName, col, ids)
		if err != nil {
			sl.log.Error().Err(err).Msgf("the documents with ids=%s are not received for publishDocument event", ids)
		}
		for _, doc := range docs {
			sl.PublishDocument(auth, dbName, "db-"+col, model.MsgTypeDBUpdated, doc)
		}
	}()
	return
}

func (sl *SQLite) IncrementValue(auth model.Auth, dbName, col, id, field string, n int) error {
	doc, err := sl.GetDocumentByID(auth, dbName, col, id)
	if err != nil {
		return err
	}

	switch doc[field].(type) {
	case float64:
		v := doc[field].(float64)
		doc[field] = v + float64(n)
	case int64:
		v := doc[field].(int64)
		doc[field] = v + int64(n)
	default:
		return fmt.Errorf("invalid type for increment: %s", reflect.TypeOf(doc[field]))
	}

	update := make(map[string]any)
	update[field] = doc[field]

	doc, err = sl.UpdateDocument(auth, dbName, col, id, update)
	if err != nil {
		return err
	}

	sl.PublishDocument(auth, dbName, "db-"+col, model.MsgTypeDBUpdated, doc)

	return nil
}

func (sl *SQLite) DeleteDocument(auth model.Auth, dbName, col, id string) (int64, error) {
	where := secureWrite(auth, col)

	qry := fmt.Sprintf(`
		DELETE 
		FROM %s_%s 
		%s AND id = $3
	`, dbName, model.CleanCollectionName(col), where)

	res, err := sl.DB.Exec(qry, auth.AccountID, auth.UserID, id)
	if err != nil {
		return 0, err
	}

	sl.PublishDocument(auth, dbName, "db-"+col, model.MsgTypeDBDeleted, id)
	return res.RowsAffected()
}

func (sl *SQLite) DeleteDocuments(auth model.Auth, dbName, col string, filters map[string]any) (n int64, err error) {
	where := secureWrite(auth, col)
	where = applyFilter(where, filters)

	var ids []string
	qry := fmt.Sprintf(`
		SELECT id
		FROM %s_%s 
		%s
	`, dbName, model.CleanCollectionName(col), where)

	rows, err := sl.DB.Query(qry, auth.AccountID, auth.UserID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			sl.log.Error().Err(err).Msg("error occurred during scanning id for DeleteDocuments event")
			continue
		}

		ids = append(ids, id)
	}

	qry = fmt.Sprintf(`
		DELETE 
		FROM %s_%s 
		%s
	`, dbName, model.CleanCollectionName(col), where)

	res, err := sl.DB.Exec(qry, auth.AccountID, auth.UserID)
	if err != nil {
		return 0, err
	}

	go func() {
		for _, id := range ids {
			sl.PublishDocument(auth, dbName, "db-"+col, model.MsgTypeDBDeleted, id)
		}
	}()

	return res.RowsAffected()
}

func (sl *SQLite) ListCollections(dbName string) (results []string, err error) {
	qry := fmt.Sprintf(`
		SELECT name 
		FROM sqlite_schema 
		WHERE type='table' AND name LIKE '%s'
		ORDER BY name;
	`, strings.ToLower(dbName)+"_%")

	rows, err := sl.DB.Query(qry)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return
		}

		// we remove the dbname from the collection name
		name = strings.Replace(name, dbName+"_", "", -1)
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

func isTableExists(err error) bool {
	return !strings.Contains(err.Error(), "no such table")
}
