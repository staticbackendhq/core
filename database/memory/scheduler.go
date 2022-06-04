package memory

import (
	"github.com/staticbackendhq/core/internal"
)

func (m *Memory) ListTasks() (results []internal.Task, err error) {
	bases, err := m.ListDatabases()
	if err != nil {
		return
	}

	var tasks []internal.Task
	for _, base := range bases {
		tasks, err = all[internal.Task](m, base.Name, "sb_tasks")
		if err != nil {
			return
		}

		results = append(results, tasks...)
	}

	return
}
