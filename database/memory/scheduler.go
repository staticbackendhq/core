package memory

import "github.com/staticbackendhq/core/model"

func (m *Memory) ListTasks() (results []model.Task, err error) {
	bases, err := m.ListDatabases()
	if err != nil {
		return
	}

	var tasks []model.Task
	for _, base := range bases {
		tasks, err = all[model.Task](m, base.Name, "sb_tasks")
		if err != nil {
			return
		}

		results = append(results, tasks...)
	}

	return
}
