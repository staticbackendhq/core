package memory

import (
	"errors"
	"fmt"
	"time"

	"github.com/staticbackendhq/core/model"
)

func (m *Memory) AddFunction(dbName string, data model.ExecData) (id string, err error) {
	id = m.NewID()

	data.ID = id
	data.LastUpdated = time.Now()

	err = create(m, dbName, "sb_functions", id, data)
	return
}

func (m *Memory) UpdateFunction(dbName, id, code, trigger string) error {
	var data model.ExecData
	if err := getByID(m, dbName, "sb_functions", id, &data); err != nil {
		return err
	}

	data.TriggerTopic = trigger
	data.Code = code
	data.Version += 1

	return create(m, dbName, "sb_functions", id, data)
}

func (m *Memory) GetFunctionForExecution(dbName, name string) (data model.ExecData, err error) {
	list, err := all[model.ExecData](m, dbName, "sb_functions")
	if err != nil {
		return
	}

	list = filter(list, func(data model.ExecData) bool {
		return data.FunctionName == name
	})

	if len(list) != 1 {
		err = errors.New("too many result returned")
		return
	}

	data = list[0]
	return
}

func (m *Memory) GetFunctionByID(dbName, id string) (data model.ExecData, err error) {
	err = getByID(m, dbName, "sb_functions", id, &data)
	return
}

func (m *Memory) GetFunctionByName(dbName, name string) (data model.ExecData, err error) {
	return m.GetFunctionForExecution(dbName, name)
}

func (m *Memory) ListFunctions(dbName string) (list []model.ExecData, err error) {
	list, err = all[model.ExecData](m, dbName, "sb_functions")
	return

}

func (m *Memory) ListFunctionsByTrigger(dbName, trigger string) (list []model.ExecData, err error) {
	list, err = m.ListFunctions(dbName)
	if err != nil {
		return
	}

	list = filter(list, func(data model.ExecData) bool {
		return data.TriggerTopic == trigger
	})

	return
}

func (m *Memory) DeleteFunction(dbName, name string) error {
	exists, err := m.GetFunctionByName(dbName, name)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s_sb_functions", dbName)

	list, ok := m.DB[key]
	if !ok {
		return errors.New("no functions found")
	}

	delete(list, exists.ID)

	m.DB[key] = list

	return nil
}

func (m *Memory) RanFunction(dbName, id string, rh model.ExecHistory) error {
	exists, err := m.GetFunctionByID(dbName, id)
	if err != nil {
		return err
	}

	exists.History = append(exists.History, rh)

	return create(m, dbName, "sb_functions", id, exists)
}
