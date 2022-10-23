package memory

import (
	"github.com/staticbackendhq/core/model"
)

func (m *Memory) Count(auth model.Auth, dbName, col string, filter map[string]interface{}) (int64, error) {
	list, err := all[map[string]any](m, dbName, col)
	if err != nil {
		return -1, err
	}

	list = secureRead(auth, col, list)

	filtered := filterByClauses(list, filter)

	return int64(len(filtered)), nil
}
