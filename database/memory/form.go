package memory

import "errors"

func (m *Memory) AddFormSubmission(dbName, form string, doc map[string]interface{}) error {
	return errors.New("not implemented")
}

func (m *Memory) ListFormSubmissions(dbName, name string) ([]map[string]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (m *Memory) GetForms(dbName string) ([]string, error) {
	return nil, errors.New("not implemented")
}
