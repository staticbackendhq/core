package memory

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	doc[FieldCreated] = time.Now()

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

	list = secureRead(auth, col, list)

	if params.SortDescending {
		list = sortSlice[map[string]any](list, func(a, b map[string]any) bool {
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

func (m *Memory) QueryDocuments(auth internal.Auth, dbName, col string, filter map[string]any, params internal.ListParams) (result internal.PagedResult, err error) {
	list, err := all[map[string]any](m, dbName, col)
	if err != nil {
		return
	}

	list = secureRead(auth, col, list)

	var filtered []map[string]any
	for _, doc := range list {
		matches := 0
		for k, v := range filter {
			op, field := extractOperatorAndValue(k)
			switch op {
			case "=":
				if equal(doc[field], v) {
					matches++
				}
			case "!=":
				if notEqual(doc[field], v) {
					matches++
				}
			case ">":
				if greater(doc[field], v) {
					matches++
				}
			case "<":
				if lower(doc[field], v) {
					matches++
				}
			case ">=":
				if greaterThanEqual(doc[field], v) {
					matches++
				}
			case "<=":
				if lowerThanEqual(doc[field], v) {
					matches++
				}
			}
		}

		if matches == len(filter) {
			filtered = append(filtered, doc)
		}
	}

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

func (m *Memory) GetDocumentByID(auth internal.Auth, dbName, col, id string) (doc map[string]interface{}, err error) {
	err = getByID[*map[string]any](m, dbName, col, id, &doc)

	list := secureRead(auth, col, []map[string]any{doc})
	if len(list) == 0 {
		err = errors.New("not authorized")
	} else {
		doc = list[0]
	}
	return
}

func (m *Memory) UpdateDocument(auth internal.Auth, dbName, col, id string, doc map[string]any) (exists map[string]any, err error) {
	exists, err = m.GetDocumentByID(auth, dbName, col, id)
	if err != nil {
		return
	} else if !canWrite(auth, col, exists) {
		err = errors.New("not authorized")
		return
	}

	delete(doc, FieldID)
	delete(doc, FieldAccountID)
	delete(doc, FieldOwnerID)

	for k, v := range doc {
		exists[k] = v
	}

	err = create[map[string]any](m, dbName, col, id, exists)
	return
}

func (m *Memory) IncrementValue(auth internal.Auth, dbName, col, id, field string, n int) error {
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

	return create[map[string]any](m, dbName, col, id, doc)
}

func (m *Memory) DeleteDocument(auth internal.Auth, dbName, col, id string) (n int64, err error) {
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
