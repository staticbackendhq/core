package mongo

import "testing"

func TestCount(t *testing.T) {
	task1 := newTask("task_with_filter", false)
	_, err := datastore.CreateDocument(adminAuth, confDBName, colName, task1)
	if err != nil {
		t.Fatal(err)
	}

	task2 := newTask("task_with_filter", false)
	_, err = datastore.CreateDocument(adminAuth, confDBName, colName, task2)
	if err != nil {
		t.Fatal(err)
	}

	var clauses [][]interface{}

	clauses = append(clauses, []interface{}{"title", "=", "task_with_filter"})

	filters, err := datastore.ParseQuery(clauses)
	if err != nil {
		t.Fatal(err)
	}

	count, err := datastore.Count(adminAuth, confDBName, colName, filters)
	if err != nil {
		t.Fatal(err)
	}

	if count != 2 {
		t.Fatalf("expected 2 got %v", count)
	}
}

func TestCountWithNoFilters(t *testing.T) {
	task1 := newTask("should be in list", false)
	_, err := datastore.CreateDocument(adminAuth, confDBName, colName, task1)
	if err != nil {
		t.Fatal(err)
	}

	count, err := datastore.Count(adminAuth, confDBName, colName, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Here we expect this count because after running all tests total count of documents will be 16 + 3
	if count != 19 {
		t.Fatalf("expected 19 got %v", count)
	}
}
