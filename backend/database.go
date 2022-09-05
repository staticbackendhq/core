package backend

import (
	"encoding/json"

	"github.com/staticbackendhq/core/internal"
)

type Database struct{}

func (d Database) Create(auth internal.Auth, dbName, col string, body any, v any) error {
	doc, err := toDoc(body)
	if err != nil {
		return err
	}

	doc, err = datastore.CreateDocument(auth, dbName, col, doc)
	if err != nil {
		return err
	}
	return fromDoc(doc, v)
}

func (d Database) GetByID(auth internal.Auth, dbName, col, id string, v any) error {
	doc, err := datastore.GetDocumentByID(auth, dbName, col, id)
	if err != nil {
		return err
	}

	return fromDoc(doc, v)
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
