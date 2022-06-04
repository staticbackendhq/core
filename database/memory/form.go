package memory

import (
	"time"
)

func (m *Memory) AddFormSubmission(dbName, form string, doc map[string]any) error {
	id := m.NewID()

	doc[FieldID] = id
	doc["sb_form"] = form
	doc[FieldCreated] = time.Now()

	return create(m, dbName, "sb_forms", id, doc)
}

func (m *Memory) ListFormSubmissions(dbName, name string) (docs []map[string]any, err error) {
	docs, err = all[map[string]any](m, dbName, "sb_forms")
	if err != nil {
		return
	}

	if len(name) > 0 {
		docs = filter(docs, func(f map[string]any) bool {
			return f["sb_form"] == name
		})
	}

	return
}

func (m *Memory) GetForms(dbName string) (names []string, err error) {
	docs, err := all[map[string]any](m, dbName, "sb_forms")
	if err != nil {
		return
	}

	uniq := make(map[string]bool)

	for _, doc := range docs {
		name := doc["sb_form"].(string)
		if _, ok := uniq[name]; !ok {
			uniq[name] = true
		}
	}

	for k := range uniq {
		names = append(names, k)
	}
	return
}
