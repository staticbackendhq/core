package mongo

import (
	"testing"
)

func TestListTasks(t *testing.T) {
	_, err := datastore.ListTasks()
	if err != nil {
		t.Fatal(err)
	}
}
