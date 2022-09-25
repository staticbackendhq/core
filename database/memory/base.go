package memory

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/staticbackendhq/core/model"
)

const (
	FieldID        = "id"
	FieldAccountID = "accountId"
	FieldOwnerID   = "ownerId"
	FieldCreated   = "sb_created"
)

func (m *Memory) CreateDocument(auth model.Auth, dbName, col string, doc map[string]interface{}) (map[string]interface{}, error) {
	id := m.NewID()
	doc[FieldID] = id
	doc[FieldAccountID] = auth.AccountID
	doc[FieldOwnerID] = auth.UserID
	doc[FieldCreated] = time.Now()

	if err := create(m, dbName, col, id, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (m *Memory) BulkCreateDocument(auth model.Auth, dbName, col string, docs []interface{}) error {
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

func (m *Memory) ListDocuments(auth model.Auth, dbName, col string, params model.ListParams) (result model.PagedResult, err error) {
	list, err := all[map[string]any](m, dbName, col)
	if err != nil {
		if errors.Is(err, collectionNotFoundErr) {
			return model.PagedResult{Page: params.Page, Size: params.Size}, nil
		}
		return
	}

	list = secureRead(auth, col, list)

	if params.SortDescending {
		list = sortSlice(list, func(a, b map[string]any) bool {
			return fmt.Sprintf("%v", a[FieldCreated]) > fmt.Sprintf("%v", b[FieldCreated])
		})
	}

	start := (params.Page - 1) * params.Size
	end := start + params.Size - 1

	if l := int64(len(list)); end > l {
		end = l
	}

	result.Page = params.Page
	result.Size = params.Size
	result.Total = int64(len(list))
	result.Results = list[start:end]

	return
}

func (m *Memory) QueryDocuments(auth model.Auth, dbName, col string, filter map[string]any, params model.ListParams) (result model.PagedResult, err error) {
	list, err := all[map[string]any](m, dbName, col)
	if err != nil {
		if errors.Is(err, collectionNotFoundErr) {
			return model.PagedResult{Page: params.Page, Size: params.Size}, nil
		}
		return
	}

	list = secureRead(auth, col, list)

	filtered := filterByClauses(list, filter)

	start := (params.Page - 1) * params.Size
	end := start + params.Size - 1

	if l := int64(len(filtered)); end > l {
		end = l
	}

	result.Page = params.Page
	result.Size = params.Size
	result.Total = int64(len(filtered))
	result.Results = filtered[start:end]

	return
}

func (m *Memory) GetDocumentByID(auth model.Auth, dbName, col, id string) (doc map[string]interface{}, err error) {
	err = getByID(m, dbName, col, id, &doc)

	list := secureRead(auth, col, []map[string]any{doc})
	if len(list) == 0 {
		err = errors.New("not authorized")
	} else {
		doc = list[0]
	}
	return
}

func (m *Memory) GetDocumentsByIDs(auth model.Auth, dbName, col string, ids []string) (docs []map[string]interface{}, err error) {

	for _, id := range ids {
		var doc map[string]interface{}
		if err := getByID(m, dbName, col, id, &doc); err != nil {
			return []map[string]interface{}{}, err
		}
		docs = append(docs, doc)
	}

	docs = secureRead(auth, col, docs)
	return docs, nil
}

func (m *Memory) UpdateDocument(auth model.Auth, dbName, col, id string, doc map[string]any) (exists map[string]any, err error) {
	exists, err = m.GetDocumentByID(auth, dbName, col, id)
	if err != nil {
		return
	} else if !canWrite(auth, col, exists) {
		err = errors.New("not authorized")
		return
	}

	removeNotEditableFields(doc)

	for k, v := range doc {
		exists[k] = v
	}

	err = create(m, dbName, col, id, exists)
	return
}

func (m *Memory) UpdateDocuments(auth model.Auth, dbName, col string, filter map[string]interface{}, updateFields map[string]interface{}) (n int64, err error) {
	list, err := all[map[string]any](m, dbName, col)

	if err != nil {
		return
	}
	list = secureRead(auth, col, list)

	removeNotEditableFields(updateFields)
	filtered := filterByClauses(list, filter)

	for _, v := range filtered {
		_, err := m.UpdateDocument(auth, dbName, col, v[FieldID].(string), updateFields)
		if err != nil {
			return n, err
		}
		n++
	}

	return
}

func (m *Memory) IncrementValue(auth model.Auth, dbName, col, id, field string, n int) error {
	doc, err := m.GetDocumentByID(auth, dbName, col, id)
	if err != nil {
		return err
	} else if !canWrite(auth, col, doc) {
		return errors.New("unauthorized")
	}

	v, ok := doc[field]
	if !ok {
		return fmt.Errorf(`field "%s" not found`, field)
	}

	i, err := strconv.Atoi(fmt.Sprintf("%v", v))
	if err != nil {
		return err
	}

	i += n

	doc[field] = i

	return create(m, dbName, col, id, doc)
}

func (m *Memory) DeleteDocument(auth model.Auth, dbName, col, id string) (n int64, err error) {
	doc, err := m.GetDocumentByID(auth, dbName, col, id)
	if err != nil {
		return
	} else if !canWrite(auth, col, doc) {
		err = errors.New("not authorized")
		return
	}

	key := fmt.Sprintf("%s_%s", dbName, col)
	docs, ok := m.DB[key]
	if !ok {
		err = errors.New("cannot find repo")
		return
	}

	delete(docs, id)

	m.DB[key] = docs
	n = 1
	return
}

func (m *Memory) ListCollections(dbName string) (repos []string, err error) {
	for key := range m.DB {
		pairs := strings.Split(key, "_")
		if strings.EqualFold(pairs[0], dbName) {
			repos = append(repos, strings.Join(pairs[1:], "_"))
		}
	}

	return

}

func extractOperatorAndValue(s string) (op string, field string) {
	parts := strings.Split(s, " ")
	if len(parts) < 2 {
		return
	}

	op = parts[0]
	field = strings.Join(parts[1:], " ")
	return
}

func removeNotEditableFields(m map[string]any) {
	delete(m, FieldID)
	delete(m, FieldAccountID)
	delete(m, FieldOwnerID)
}

func equal(v any, val any) bool {
	return fmt.Sprintf("%v", v) == fmt.Sprintf("%v", val)
}

func notEqual(v any, val any) bool {
	return fmt.Sprintf("%v", v) != fmt.Sprintf("%v", val)
}

func greater(v any, val any) bool {
	return fmt.Sprintf("%v", v) > fmt.Sprintf("%v", val)
}

func lower(v any, val any) bool {
	return fmt.Sprintf("%v", v) < fmt.Sprintf("%v", val)
}

func greaterThanEqual(v any, val any) bool {
	return fmt.Sprintf("%v", v) >= fmt.Sprintf("%v", val)
}

func lowerThanEqual(v any, val any) bool {
	return fmt.Sprintf("%v", v) <= fmt.Sprintf("%v", val)
}
