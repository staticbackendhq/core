package memory

import (
	"errors"

	"github.com/staticbackendhq/core/internal"
)

func (m *Memory) AddFunction(dbName string, data internal.ExecData) (id string, err error) {
	err = errors.New("not implemented")
	return
}

func (m *Memory) UpdateFunction(dbName, id, code, trigger string) error {
	return errors.New("not implemented")
}

func (m *Memory) GetFunctionForExecution(dbName, name string) (data internal.ExecData, err error) {
	err = errors.New("not implemented")
	return
}

func (m *Memory) GetFunctionByID(dbName, id string) (data internal.ExecData, err error) {
	err = errors.New("not implemented")
	return
}

func (m *Memory) GetFunctionByName(dbName, name string) (data internal.ExecData, err error) {
	err = errors.New("not implemented")
	return
}

func (m *Memory) ListFunctions(dbName string) ([]internal.ExecData, error) {
	return nil, errors.New("not implemented")
}

func (m *Memory) ListFunctionsByTrigger(dbName, trigger string) ([]internal.ExecData, error) {
	return nil, errors.New("not implemented")
}

func (m *Memory) DeleteFunction(dbName, name string) error {
	return errors.New("not implemented")
}

func (m *Memory) RanFunction(dbName, id string, rh internal.ExecHistory) error {
	return errors.New("not implemented")
}
