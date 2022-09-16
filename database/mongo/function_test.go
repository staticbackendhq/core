package mongo

import (
	"testing"
	"time"

	"github.com/staticbackendhq/core/model"
)

func createFunction(name string, args ...string) (string, error) {
	topic := "web"
	if len(args) == 1 {
		topic = args[0]
	}

	fn := model.ExecData{
		AccountID:    adminAccount.ID,
		FunctionName: name,
		TriggerTopic: topic,
		Code:         "unit-test-not-relevant",
		Version:      1,
		LastUpdated:  time.Now(),
	}
	return datastore.AddFunction(confDBName, fn)
}

func TestAddFunction(t *testing.T) {
	id, err := createFunction("fn1")
	if err != nil {
		t.Fatal(err)
	} else if len(id) < 10 {
		t.Errorf("expected id len to > 10 got %s", id)
	}

	fn, err := datastore.GetFunctionByID(confDBName, id)
	if err != nil {
		t.Fatal(err)
	} else if fn.ID != id || fn.FunctionName != "fn1" {
		t.Errorf("expected id and name to be %s | fn1 got %s | %s", id, fn.ID, fn.FunctionName)
	}
}

func TestUpdateFunction(t *testing.T) {
	id, err := createFunction("update")
	if err != nil {
		t.Fatal(err)
	}

	if err := datastore.UpdateFunction(confDBName, id, "update", "web"); err != nil {
		t.Fatal(err)
	}

	fn, err := datastore.GetFunctionByID(confDBName, id)
	if err != nil {
		t.Fatal(err)
	} else if fn.Code != "update" {
		t.Errorf("expected code to be 'update' got %s", fn.Code)
	} else if fn.Version != 2 {
		t.Errorf("expected version to be 2 got %d", fn.Version)
	}
}

func TestGetFunctionForExecution(t *testing.T) {
	fnName := "get-for-exec"
	id, err := createFunction(fnName)
	if err != nil {
		t.Fatal(err)
	}

	fn, err := datastore.GetFunctionForExecution(confDBName, fnName)
	if err != nil {
		t.Fatal(err)
	} else if fn.ID != id || fn.FunctionName != fnName {
		t.Errorf("expected id, name to be %s, %s got %s, %s", id, fnName, fn.ID, fn.FunctionName)
	} else if len(fn.Code) != 22 {
		t.Errorf("expected code len to be 22 got %d | %s", len(fn.Code), fn.Code)
	}
}

func TestGetFunctionByName(t *testing.T) {
	id, err := createFunction("get-by-name")
	if err != nil {
		t.Fatal(err)
	}

	fn, err := datastore.GetFunctionByName(confDBName, "get-by-name")
	if err != nil {
		t.Fatal(err)
	} else if fn.ID != id || fn.FunctionName != "get-by-name" {
		t.Errorf("expected id, name to be %s, get-by-name got %s,%s", id, fn.ID, fn.FunctionName)
	}
}

func TestListFunctions(t *testing.T) {
	id1, err := createFunction("lfn1")
	if err != nil {
		t.Fatal(err)
	}

	id2, err := createFunction("lfn2")
	if err != nil {
		t.Fatal(err)
	}

	results, err := datastore.ListFunctions(confDBName)
	if err != nil {
		t.Fatal(err)
	}

	var found []bool
	for _, fn := range results {
		if fn.ID == id1 || fn.ID == id2 {
			found = append(found, true)
		}
	}

	if len(found) != 2 {
		t.Errorf("expected to find 2 functions found %d", len(found))
	}
}

func TestListFunctionsByTrigger(t *testing.T) {
	id1, err := createFunction("tfn1", "topic-here")
	if err != nil {
		t.Fatal(err)
	}

	id2, err := createFunction("tfn2", "topic-here")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := createFunction("tfn3", "topic-here-not"); err != nil {
		t.Fatal(err)
	}

	results, err := datastore.ListFunctionsByTrigger(confDBName, "topic-here")
	if err != nil {
		t.Fatal(err)
	}

	var found []bool
	for _, fn := range results {
		if fn.ID == id1 || fn.ID == id2 {
			found = append(found, true)
		}
	}

	if len(found) != 2 {
		t.Errorf("expected to find 2 functions found %d", len(found))
	}
}

func TestDeleteFunction(t *testing.T) {
	id, err := createFunction("del", "del")
	if err != nil {
		t.Fatal(err)
	}

	if err := datastore.DeleteFunction(confDBName, "del"); err != nil {
		t.Fatal(err)
	}

	results, err := datastore.GetFunctionByID(confDBName, id)
	if err == nil {
		t.Errorf("expected to get an error")
	} else if results.ID == id {
		t.Errorf("function should have been deleted")
	}
}

func TestRanFunction(t *testing.T) {
	id, err := createFunction("test-run", "test")
	if err != nil {
		t.Fatal(err)
	}

	rh := model.ExecHistory{
		FunctionID: id,
		Version:    1,
		Started:    time.Now().Add(-2 * time.Second),
		Completed:  time.Now(),
		Success:    true,
		Output:     []string{"started", "run", "completed"},
	}

	if err := datastore.RanFunction(confDBName, id, rh); err != nil {
		t.Fatal(err)
	}

	fn, err := datastore.GetFunctionByID(confDBName, id)
	if err != nil {
		t.Fatal(err)
	} else if len(fn.History) == 0 {
		t.Errorf("expected history to have 1 item, got %d", len(fn.History))
	} else if !fn.History[0].Success || fn.History[0].Version != 1 {
		t.Errorf("expected history[0] to have succeeded and version at 1 got %v", fn.History[0])
	}
}
