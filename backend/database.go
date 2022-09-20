package backend

import (
	"encoding/json"
	"errors"

	"github.com/staticbackendhq/core/model"
)

// Database enables all CRUD and querying operations on a specific type
type Database[T any] struct {
	auth model.Auth
	conf model.DatabaseConfig
}

// Collection returns a ready to use Database to perform operations on a specific type
func Collection[T any](auth model.Auth, base model.DatabaseConfig) Database[T] {
	return Database[T]{
		auth: auth,
		conf: base,
	}
}

// Create creates a record in the collection/repository
func (d Database[T]) Create(col string, data T) (inserted T, err error) {
	doc, err := toDoc(data)
	if err != nil {
		return
	}

	doc, err = DB.CreateDocument(d.auth, d.conf.Name, col, doc)
	if err != nil {
		return
	}
	err = fromDoc(doc, &inserted)
	return
}

// BulkCreate creates multiple records in the collection/repository
func (d Database[T]) BulkCreate(col string, entities []T) error {
	docs := make([]interface{}, 0)

	for _, doc := range entities {
		x, err := toDoc(doc)
		if err != nil {
			return err
		}

		docs = append(docs, x)
	}
	return DB.BulkCreateDocument(d.auth, d.conf.Name, col, docs)
}

// PageResult wraps a slice of type T with paging information
type PagedResult[T any] struct {
	Page    int64
	Size    int64
	Total   int64
	Results []T
}

// List returns records from a collection/repository using paging/sorting params
func (d Database[T]) List(col string, lp model.ListParams) (res PagedResult[T], err error) {
	r, err := DB.ListDocuments(d.auth, d.conf.Name, col, lp)
	if err != nil {
		return
	}

	for _, doc := range r.Results {
		var v T
		if err = fromDoc(doc, &v); err != nil {
			return
		}

		res.Results = append(res.Results, v)
	}

	res.Page = r.Page
	res.Size = r.Size
	res.Total = r.Total

	return
}

// Query returns records that match with the provided filters.
func (d Database[T]) Query(col string, filters [][]any, lp model.ListParams) (res PagedResult[T], err error) {
	clauses, err := DB.ParseQuery(filters)
	if err != nil {
		return
	}

	r, err := DB.QueryDocuments(d.auth, d.conf.Name, col, clauses, lp)
	if err != nil {
		return
	}

	for _, doc := range r.Results {
		var v T
		if err = fromDoc(doc, &v); err != nil {
			return
		}

		res.Results = append(res.Results, v)
	}

	res.Page = r.Page
	res.Size = r.Size
	res.Total = r.Total

	return
}

// GetByID returns a specific record from a collection/repository
func (d Database[T]) GetByID(col, id string) (entity T, err error) {
	doc, err := DB.GetDocumentByID(d.auth, d.conf.Name, col, id)
	if err != nil {
		return
	}

	err = fromDoc(doc, &entity)
	return
}

// Update updates some fields of a record
func (d Database[T]) Update(col, id string, v any) (entity T, err error) {
	doc, err := toDoc(v)
	if err != nil {
		return
	}

	x, err := DB.UpdateDocument(d.auth, d.conf.Name, col, id, doc)
	if err != nil {
		return
	}

	err = fromDoc(x, &entity)
	return
}

// UpdateMany updates multiple records matching filters
func (d Database[T]) UpdateMany(col string, filters [][]any, v any) (int64, error) {
	clauses, err := DB.ParseQuery(filters)
	if err != nil {
		return 0, err
	}

	doc, err := toDoc(v)
	if err != nil {
		return 0, err
	}
	return DB.UpdateDocuments(d.auth, d.conf.Name, col, clauses, doc)
}

// IncrementValue increments or decrements a specifc field from a collection/repository
func (d Database[T]) IncrementValue(col, id, field string, n int) error {
	return DB.IncrementValue(d.auth, d.conf.Name, col, id, field, n)
}

// Delete removes a record from a collection
func (d Database[T]) Delete(col, id string) (int64, error) {
	return DB.DeleteDocument(d.auth, d.conf.Name, col, id)
}

func toDoc(v any) (doc map[string]any, err error) {
	// TODO: this is certainly not the most performant way to do this.

	b, err := json.Marshal(v)
	if err != nil {
		return
	}

	err = json.Unmarshal(b, &doc)
	return
}

func fromDoc(doc map[string]any, v interface{}) error {
	b, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, v)
}

// BuildQueryFilters helps building the proper slice of filters
func BuildQueryFilters(p ...any) (q [][]any, err error) {
	if len(p)%3 != 0 {
		err = errors.New("parameters should all have 3 values for each criteria")
		return
	}

	for i := 0; i < len(p); i++ {
		q = append(q, []any{
			p[i], p[i+1], p[i+2],
		})

		i += 2
	}

	return
}
