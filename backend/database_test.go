package backend_test

import (
	"testing"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/model"
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
	db := backend.Collection[Task](adminAuth, base)

	task := newTask("db create", false)
	task, err := db.Create("tasks", task)
	if err != nil {
		t.Fatal(err)
	} else if len(task.ID) == 0 {
		t.Error("expected task id length to be > 0")
	}

	check, err := db.GetByID("tasks", task.ID)
	if err != nil {
		t.Fatal(err)
	} else if task.Title != check.Title {
		t.Errorf(`expected title to be "%s" got "%s"`, task.Title, check.Title)
	}
}

func TestDatabaseList(t *testing.T) {
	db := backend.Collection[Task](adminAuth, base)

	tasks := []Task{
		newTask("t1", false),
		newTask("t2", true),
	}

	if err := db.BulkCreate("tasks", tasks); err != nil {
		t.Fatal(err)
	}

	lp := model.ListParams{Page: 1, Size: 50}
	res, err := db.List("tasks", lp)
	if err != nil {
		t.Fatal(err)
	} else if len(res.Results) < 2 {
		t.Errorf("expected to have at least 2 elem, got %d", len(res.Results))
	}
}

func TestDatabaseQuery(t *testing.T) {
	db := backend.Collection[Task](adminAuth, base)

	tasks := []Task{
		newTask("qry1", false),
		newTask("qry2", true),
		newTask("qry2", false),
	}

	if err := db.BulkCreate("tasks", tasks); err != nil {
		t.Fatal(err)
	}

	filters, err := backend.BuildQueryFilters(
		"title", "==", "qry2",
		"done", "==", true,
	)
	if err != nil {
		t.Fatal(err)
	}

	lp := model.ListParams{Page: 1, Size: 50}
	res, err := db.Query("tasks", filters, lp)
	if err != nil {
		t.Fatal(err)
	} else if res.Total != 1 {
		t.Errorf("expected total to be 1 got %d", res.Total)
	} else if res.Results[0].Title != "qry2" {
		t.Error("got the wrong task", res.Results[0])
	}
}

func TestDatabaseBuildQueryFilters(t *testing.T) {
	filters, err := backend.BuildQueryFilters(
		"field", "=", "value",
		"field2", ">=", 123,
	)

	var expected [][]any
	f1 := []any{"field", "=", "value"}
	f2 := []any{"field2", ">=", 123}
	expected = append(expected, f1)
	expected = append(expected, f2)

	compare := func(s1, s2 [][]any) bool {
		if len(s1) != len(s2) {
			return false
		}

		for i := 0; i < len(s1); i++ {
			for j := 0; j < len(s1[i]); j++ {
				if s1[i][j] != s2[i][j] {
					return false
				}
			}
		}
		return true
	}

	if err != nil {
		t.Fatal(err)
	} else if !compare(filters, expected) {
		t.Log(filters)
		t.Error("filters are not properly created")
	}
}
