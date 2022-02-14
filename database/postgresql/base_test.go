package postgresql

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/staticbackendhq/core/internal"
)

type Task struct {
	ID        string    `json:"id"`
	AccountID string    `json:"accountId"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	Todos     []Todo    `json:"todos"`
	Created   time.Time `json:"created"`
}

type Todo struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// simulates receiving json via the HTTP endpoint
func enc(task Task) map[string]interface{} {
	b, err := json.Marshal(task)
	if err != nil {
		return nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}

func dec(m map[string]interface{}) Task {
	b, err := json.Marshal(m)
	if err != nil {
		return Task{}
	}

	var t Task
	if err := json.Unmarshal(b, &t); err != nil {
		return Task{}
	}
	return t
}

func newTask(title string, done bool) map[string]interface{} {
	return enc(Task{
		Title:   title,
		Done:    done,
		Todos:   []Todo{Todo{Title: "sub", Done: done}, Todo{Title: "sub2", Done: done}},
		Created: time.Now(),
	})
}

func TestCreateDocument(t *testing.T) {
	task1 := newTask("task1", false)
	_, err := datastore.CreateDocument(adminAuth, confDBName, colName, task1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBulkCreateDocument(t *testing.T) {
	var many []interface{}
	for i := 0; i < 5; i++ {
		many = append(many, newTask(fmt.Sprintf("title %d", i), true))
	}

	if err := datastore.BulkCreateDocument(adminAuth, confDBName, colName, many); err != nil {
		t.Fatal(err)
	}
}

func TestListDocuments(t *testing.T) {
	task1 := newTask("should be in list", false)
	inserted, err := datastore.CreateDocument(adminAuth, confDBName, colName, task1)
	if err != nil {
		t.Fatal(err)
	}

	insertedTask := dec(inserted)

	lp := internal.ListParams{Page: 1, Size: 25, SortDescending: true}

	result, err := datastore.ListDocuments(adminAuth, confDBName, colName, lp)
	if err != nil {
		t.Fatal(err)
	} else if result.Total <= 0 {
		t.Fatalf("expected to has result")
	}

	found := false
	for _, res := range result.Results {
		tmp := dec(res)
		if tmp.ID == insertedTask.ID && tmp.AccountID == adminToken.AccountID {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected to find inserted task in list")
	}
}
