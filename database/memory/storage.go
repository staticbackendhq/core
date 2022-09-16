package memory

import (
	"errors"
	"fmt"

	"github.com/staticbackendhq/core/model"
)

func (m *Memory) AddFile(dbName string, f model.File) (id string, err error) {
	id = m.NewID()
	f.ID = id
	err = create(m, dbName, "sb_files", id, f)
	return
}

func (m *Memory) GetFileByID(dbName, fileID string) (f model.File, err error) {
	err = getByID(m, dbName, "sb_files", fileID, &f)
	return
}

func (m *Memory) DeleteFile(dbName, fileID string) error {
	key := fmt.Sprintf("%s_sb_files", dbName)

	files, ok := m.DB[key]
	if !ok {
		return errors.New("no files available for delete")
	}

	delete(files, fileID)

	m.DB[key] = files
	return nil
}

func (m *Memory) ListAllFiles(dbName, accountID string) (results []model.File, err error) {
	files, err := all[model.File](m, dbName, "sb_files")
	if err != nil {
		return
	}

	results = filter(files, func(x model.File) bool {
		return x.AccountID == accountID
	})

	return
}
