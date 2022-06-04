package memory

import (
	"errors"
	"fmt"

	"github.com/staticbackendhq/core/internal"
)

func (m *Memory) AddFile(dbName string, f internal.File) (id string, err error) {
	id = m.NewID()
	f.ID = id
	err = create[internal.File](m, dbName, "sb_files", id, f)
	return
}

func (m *Memory) GetFileByID(dbName, fileID string) (f internal.File, err error) {
	err = getByID[*internal.File](m, dbName, "sb_files", fileID, &f)
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
