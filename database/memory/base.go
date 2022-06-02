package memory

import (
	"errors"
	"fmt"

	"github.com/staticbackendhq/core/internal"
)

const (
	FieldID        = "id"
	FieldAccountID = "accountId"
	FieldOwnerID   = "ownerId"
	FieldCreated   = "sb_created"
)

func (m *Memory) CreateDocument(auth internal.Auth, dbName, col string, doc map[string]interface{}) (map[string]interface{}, error) {
	id := m.NewID()
	doc[FieldID] = id
	doc[FieldAccountID] = auth.AccountID
	doc[FieldOwnerID] = auth.UserID

	if err := create[map[string]any](m, dbName, col, id, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (m *Memory) BulkCreateDocument(auth internal.Auth, dbName, col string, docs []interface{}) error {
	for _, v := range docs {
		doc, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("cannot cast to map[sring]any")
		}

		if _, err := m.CreateDocument(auth, dbName, col, doc); err != nil {
			return err
		}
	}

	return nil
}

func (m *Memory) ListDocuments(auth internal.Auth, dbName, col string, params internal.ListParams) (result internal.PagedResult, err error) {
	list, err := all[map[string]any](m, dbName, col)
	if err != nil {
		return
	}

	if params.SortDescending {
		list = sortSlice[map[string]any](list, func(a, b map[string]any) bool {
			return fmt.Sprintf("%v", a[FieldCreated]) > fmt.Sprintf("%v", b[FieldCreated])
		})
	}

	start := (params.Page - 1) * params.Size
	end := start + params.Size - 1

	result.Page = params.Page
	result.Size = params.Size
	result.Total = int64(len(list))
	result.Results = list[start:end]

	return
}

func (m *Memory) QueryDocuments(auth internal.Auth, dbName, col string, filter map[string]interface{}, params internal.ListParams) (result internal.PagedResult, err error) {
	return
}

func (m *Memory) GetDocumentByID(auth internal.Auth, dbName, col, id string) (doc map[string]interface{}, err error) {
	err = getByID[map[string]any](m, dbName, col, id, doc)
	return
}

func (m *Memory) UpdateDocument(auth internal.Auth, dbName, col, id string, doc map[string]interface{}) (map[string]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (m *Memory) IncrementValue(auth internal.Auth, dbName, col, id, field string, n int) error {
	return errors.New("not implemented")
}

func (m *Memory) DeleteDocument(auth internal.Auth, dbName, col, id string) (int64, error) {
	return 0, errors.New("not implemented")
}

func (m *Memory) ListCollections(dbName string) ([]string, error) {
	return nil, errors.New("not implemented")
}
