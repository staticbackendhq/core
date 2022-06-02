package memory

import (
	"errors"

	"github.com/staticbackendhq/core/internal"
)

func (m *Memory) AddFile(dbName string, f internal.File) (id string, err error) {
	err = errors.New("not implemented")
	return
}

func (m *Memory) GetFileByID(dbName, fileID string) (f internal.File, err error) {
	err = errors.New("not implemented")
	return
}

func (m *Memory) DeleteFile(dbName, fileID string) error {
	return errors.New("not implemented")
}
