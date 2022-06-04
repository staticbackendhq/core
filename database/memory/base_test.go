package memory

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
	Likes     int64     `json:"likes"`
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

func TestQueryDocuments(t *testing.T) {
	task1 := newTask("where1", false)
	task2 := newTask("where2", true)

	var many []interface{}
	many = append(many, task1)
	many = append(many, task2)

	if err := datastore.BulkCreateDocument(adminAuth, confDBName, colName, many); err != nil {
		t.Fatal(err)
	}

	var clauses [][]interface{}
	clauses = append(clauses, []interface{}{"title", "=", "where1"})
	clauses = append(clauses, []interface{}{"done", "=", false})

	lp := internal.ListParams{Page: 1, Size: 5}

	filters, err := datastore.ParseQuery(clauses)
	if err != nil {
		t.Fatal(err)
	}

	result, err := datastore.QueryDocuments(adminAuth, confDBName, colName, filters, lp)
	if err != nil {
		t.Fatal(err)
	} else if result.Total != 1 {
		t.Fatalf("expected total to be 1 got %d", result.Total)
	}

	r1 := dec(result.Results[0])
	if r1.Title != "where1" || r1.Done {
		t.Errorf("expected r1 to have where1 and false as value: %v", r1)
	}
}

func TestGetDocumentByID(t *testing.T) {
	task1 := newTask("getbyid", false)

	m, err := datastore.CreateDocument(adminAuth, confDBName, colName, task1)
	if err != nil {
		t.Fatal(err)
	}

	inserted := dec(m)

	m2, err := datastore.GetDocumentByID(adminAuth, confDBName, colName, inserted.ID)
	if err != nil {
		t.Fatal(err)
	}

	found := dec(m2)
	if len(found.ID) < 10 || found.ID != inserted.ID {
		t.Errorf("expected id to be %s got %s", inserted.ID, found.ID)
	}
}

func TestUpdateDocument(t *testing.T) {
	task1 := newTask("inserted", false)

	m, err := datastore.CreateDocument(adminAuth, confDBName, colName, task1)
	if err != nil {
		t.Fatal(err)
	}

	inserted := dec(m)

	update := inserted
	update.Title = "updated"
	update.Done = true
	update.Todos[0].Title = "updated"
	update.Todos[0].Done = true

	um, err := datastore.UpdateDocument(adminAuth, confDBName, colName, inserted.ID, enc(update))
	if err != nil {
		t.Fatal(err)
	}

	m2, err := datastore.GetDocumentByID(adminAuth, confDBName, colName, inserted.ID)
	if err != nil {
		t.Fatal(err)
	} else if um["title"] != m2["title"] {
		t.Errorf("update return map differ than found one")
	}

	updated := dec(m2)
	if updated.Title != "updated" {
		t.Errorf("expected updated title to be updated got %s", updated.Title)
	} else if !updated.Done {
		t.Errorf("expected updated done to be true")
	} else if updated.Todos[0].Title != "updated" {
		t.Errorf("expected todos[0] title to be updated got %s", updated.Todos[0].Title)
	} else if !updated.Todos[0].Done {
		t.Errorf("expected todos[0] done to be true")
	}
}

func TestIncrementValue(t *testing.T) {
	task1 := newTask("incr", false)
	m, err := datastore.CreateDocument(adminAuth, confDBName, colName, task1)
	if err != nil {
		t.Fatal(err)
	}

	inserted := dec(m)

	if err := datastore.IncrementValue(adminAuth, confDBName, colName, inserted.ID, "likes", 2); err != nil {
		t.Fatal(err)
	}

	m2, err := datastore.GetDocumentByID(adminAuth, confDBName, colName, inserted.ID)
	if err != nil {
		t.Fatal(err)
	}

	found := dec(m2)
	if found.Likes != 2 {
		t.Errorf("expected like to be 2 got %d", found.Likes)
	}
}

func TestDeleteDocument(t *testing.T) {
	task1 := newTask("to delete", true)
	m, err := datastore.CreateDocument(adminAuth, confDBName, colName, task1)
	if err != nil {
		t.Fatal(err)
	}

	inserted := dec(m)

	n, err := datastore.DeleteDocument(adminAuth, confDBName, colName, inserted.ID)
	if err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatalf("expected row count to be 1 got %d", n)
	}

	m2, err := datastore.GetDocumentByID(adminAuth, confDBName, colName, inserted.ID)
	if err == nil {
		t.Fatal("error should have a value")
	} else if m2 != nil {
		t.Errorf("map should be nil got %v", m2)
	}
}

func TestListCollections(t *testing.T) {
	results, err := datastore.ListCollections(confDBName)
	if err != nil {
		t.Fatal(err)
	} else if len(results) < 3 {
		t.Log(results)
		t.Errorf("expected to have at least one collection got %d", len(results))
	}
}
