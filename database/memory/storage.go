package memory

import (
	"errors"
	"fmt"

	"github.com/staticbackendhq/core/internal"
)

func (m *Memory) AddFile(dbName string, f internal.File) (id string, err error) {
	id = m.NewID()
	f.ID = id
	err = create(m, dbName, "sb_files", id, f)
	return
}

func (m *Memory) GetFileByID(dbName, fileID string) (f internal.File, err error) {
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

func (m *Memory) ListAllFiles(dbName, accountID string) (results []internal.File, err error) {
	files, err := all[internal.File](m, dbName, "sb_files")
	if err != nil {
		return
	}

	results = filter(files, func(x internal.File) bool {
		return x.AccountID == accountID
	})

	return
}
