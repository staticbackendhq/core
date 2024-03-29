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
	col  string
}

// Collection returns a ready to use Database to perform DB operations on a
// specific type. You must pass auth which is the user performing the action and
// the tenant's database in which this action will be executed. The col is the
// name of the collection.
//
// Collection name only accept alpha-numberic values and cannot start with a digit.
func Collection[T any](auth model.Auth, base model.DatabaseConfig, col string) Database[T] {
	return Database[T]{
		auth: auth,
		conf: base,
		col:  col,
	}
}

// Create creates a record in the collection/repository
func (d Database[T]) Create(data T) (inserted T, err error) {
	doc, err := toDoc(data)
	if err != nil {
		return
	}

	doc, err = DB.CreateDocument(d.auth, d.conf.Name, d.col, doc)
	if err != nil {
		return
	}
	err = fromDoc(doc, &inserted)
	return
}

// BulkCreate creates multiple records in the collection/repository
func (d Database[T]) BulkCreate(entities []T) error {
	docs := make([]interface{}, 0)

	for _, doc := range entities {
		x, err := toDoc(doc)
		if err != nil {
			return err
		}

		docs = append(docs, x)
	}
	return DB.BulkCreateDocument(d.auth, d.conf.Name, d.col, docs)
}

// PageResult wraps a slice of type T with paging information
type PagedResult[T any] struct {
	Page    int64
	Size    int64
	Total   int64
	Results []T
}

// List returns records from a collection/repository using paging/sorting params
func (d Database[T]) List(lp model.ListParams) (res PagedResult[T], err error) {
	r, err := DB.ListDocuments(d.auth, d.conf.Name, d.col, lp)
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
func (d Database[T]) Query(filters [][]any, lp model.ListParams) (res PagedResult[T], err error) {
	clauses, err := DB.ParseQuery(filters)
	if err != nil {
		return
	}

	r, err := DB.QueryDocuments(d.auth, d.conf.Name, d.col, clauses, lp)
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
func (d Database[T]) GetByID(id string) (entity T, err error) {
	doc, err := DB.GetDocumentByID(d.auth, d.conf.Name, d.col, id)
	if err != nil {
		return
	}

	err = fromDoc(doc, &entity)
	return
}

// Update updates some fields of a record
func (d Database[T]) Update(id string, v any) (entity T, err error) {
	doc, err := toDoc(v)
	if err != nil {
		return
	}

	x, err := DB.UpdateDocument(d.auth, d.conf.Name, d.col, id, doc)
	if err != nil {
		return
	}

	err = fromDoc(x, &entity)
	return
}

// UpdateMany updates multiple records matching filters
func (d Database[T]) UpdateMany(filters [][]any, v any) (int64, error) {
	clauses, err := DB.ParseQuery(filters)
	if err != nil {
		return 0, err
	}

	doc, err := toDoc(v)
	if err != nil {
		return 0, err
	}
	return DB.UpdateDocuments(d.auth, d.conf.Name, d.col, clauses, doc)
}

// IncrementValue increments or decrements a specifc field from a collection/repository
func (d Database[T]) IncrementValue(id, field string, n int) error {
	return DB.IncrementValue(d.auth, d.conf.Name, d.col, id, field, n)
}

// Delete removes a record from a collection
func (d Database[T]) Delete(id string) (int64, error) {
	return DB.DeleteDocument(d.auth, d.conf.Name, d.col, id)
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

// BuildQueryFilters helps building the proper slice of filters.
//
// The arguments must be divided by 3 and has the following order:
//
// field name | operator | value
//
//    backend.BuildQueryFilters("done", "=", false)
//
// This would filter for the false value in the "done" field.
//
// Supported operators: =, !=, >, <, >=, <=, in, !in
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
