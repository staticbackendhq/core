package backend

import (
	"encoding/json"
	"errors"

	"github.com/staticbackendhq/core/model"
)

type Database[T any] struct {
	auth model.Auth
	conf model.BaseConfig
}

func NewDatabase[T any](token, baseID string) Database[T] {
	return Database[T]{
		auth: findAuth(token),
		conf: findBase(baseID),
	}
}

func (d Database[T]) Create(col string, data T) (inserted T, err error) {
	doc, err := toDoc(data)
	if err != nil {
		return
	}

	doc, err = datastore.CreateDocument(d.auth, d.conf.Name, col, doc)
	if err != nil {
		return
	}
	err = fromDoc(doc, &inserted)
	return
}

func (d Database[T]) BulkCreate(col string, entities []T) error {
	docs := make([]interface{}, 0)

	for _, doc := range entities {
		x, err := toDoc(doc)
		if err != nil {
			return err
		}

		docs = append(docs, x)
	}
	return datastore.BulkCreateDocument(d.auth, d.conf.Name, col, docs)
}

type ListParams struct {
	Page           int64
	Size           int64
	SortBy         string
	SortDescending bool
}

type PagedResult[T any] struct {
	Page    int64
	Size    int64
	Total   int64
	Results []T
}

func (d Database[T]) List(col string, lp ListParams) (res PagedResult[T], err error) {
	ilp := model.ListParams{
		Page:           lp.Page,
		Size:           lp.Size,
		SortBy:         lp.SortBy,
		SortDescending: lp.SortDescending,
	}
	r, err := datastore.ListDocuments(d.auth, d.conf.Name, col, ilp)
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

func (d Database[T]) Query(col string, filters [][]any, lp ListParams) (res PagedResult[T], err error) {
	clauses, err := datastore.ParseQuery(filters)
	if err != nil {
		return
	}

	ilp := model.ListParams{
		Page:           lp.Page,
		Size:           lp.Size,
		SortBy:         lp.SortBy,
		SortDescending: lp.SortDescending,
	}

	r, err := datastore.QueryDocuments(d.auth, d.conf.Name, col, clauses, ilp)
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

func (d Database[T]) GetByID(col, id string) (entity T, err error) {
	doc, err := datastore.GetDocumentByID(d.auth, d.conf.Name, col, id)
	if err != nil {
		return
	}

	err = fromDoc(doc, &entity)
	return
}

func (d Database[T]) Update(col, id string, v any) (entity T, err error) {
	doc, err := toDoc(v)
	if err != nil {
		return
	}

	x, err := datastore.UpdateDocument(d.auth, d.conf.Name, col, id, doc)
	if err != nil {
		return
	}

	err = fromDoc(x, &entity)
	return
}

func (d Database[T]) UpdateMany(col string, filters [][]any, v any) (int64, error) {
	clauses, err := datastore.ParseQuery(filters)
	if err != nil {
		return 0, err
	}

	doc, err := toDoc(v)
	if err != nil {
		return 0, err
	}
	return datastore.UpdateDocuments(d.auth, d.conf.Name, col, clauses, doc)
}

func (d Database[T]) IncrementValue(col, id, field string, n int) error {
	return datastore.IncrementValue(d.auth, d.conf.Name, col, id, field, n)
}

func (d Database[T]) Delete(col, id string) (int64, error) {
	return datastore.DeleteDocument(d.auth, d.conf.Name, col, id)
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
