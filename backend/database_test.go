package backend_test

import (
	"testing"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	AccountID string    `json:"accountId"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	Comments  []Comment `json:"comments"`
}

type Comment struct {
	Comment string    `json:"comment"`
	Date    time.Time `json:"date"`
}

func newTask(title string, done bool) Task {
	return Task{
		Title: title,
		Done:  done,
		Comments: []Comment{
			Comment{Comment: "comment 1", Date: time.Now()},
		},
	}
}

func TestDatabaseCreate(t *testing.T) {
	task := newTask("db create", false)
	if err := bkn.DB.Create(adminAuth, base.Name, "tasks", task, &task); err != nil {
		t.Fatal(err)
	} else if len(task.ID) == 0 {
		t.Error("expected task id length to be > 0")
	}

	var check Task
	if err := bkn.DB.GetByID(adminAuth, base.Name, "tasks", task.ID, &check); err != nil {
		t.Fatal(err)
	} else if task.Title != check.Title {
		t.Errorf(`expected title to be "%s" got "%s"`, task.Title, check.Title)
	}
}
