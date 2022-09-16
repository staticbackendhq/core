package mongo

import (
	"testing"
	"time"

	"github.com/staticbackendhq/core/model"
)

func TestFileStorage(t *testing.T) {
	f := model.File{
		AccountID: adminAccount.ID,
		Key:       "key",
		URL:       "https://test",
		Size:      123456,
		Uploaded:  time.Now(),
	}

	f1 := model.File{
		AccountID: adminAccount.ID,
		Key:       "key1",
		URL:       "https://test1",
		Size:      123456,
		Uploaded:  time.Now(),
	}

	id, err := datastore.AddFile(confDBName, f)
	if err != nil {
		t.Fatal(err)
	} else if len(id) < 10 {
		t.Fatalf("expected to get id got %s", id)
	}

	_, err = datastore.AddFile(confDBName, f1)
	if err != nil {
		t.Fatal(err)
	} else if len(id) < 10 {
		t.Fatalf("expected to get id got %s", id)
	}

	list, err := datastore.ListAllFiles(confDBName, f.AccountID)
	if err != nil {
		t.Fatal(err)
	} else if len(list) < 2 || len(list) > 2 {
		t.Errorf("expected file length to be 2 go %d", len(list))
	}

	f2, err := datastore.GetFileByID(confDBName, id)
	if err != nil {
		t.Fatal(err)
	} else if f2.Key != f.Key {
		t.Errorf("expected key to be %s got %s", f.Key, f2.Key)
	}

	if err := datastore.DeleteFile(confDBName, id); err != nil {
		t.Fatal(err)
	}

	check, err := datastore.GetFileByID(confDBName, id)
	if err == nil {
		t.Errorf("error should not be nil")
	} else if check.ID == id {
		t.Errorf("deleted file id returned? %v", check)
	}
}
