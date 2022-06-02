package memory

import (
	"testing"
	"time"

	"github.com/staticbackendhq/core/internal"
)

func TestFileStorage(t *testing.T) {
	f := internal.File{
		AccountID: adminAccount.ID,
		Key:       "key",
		URL:       "https://test",
		Size:      123456,
		Uploaded:  time.Now(),
	}

	id, err := datastore.AddFile(confDBName, f)
	if err != nil {
		t.Fatal(err)
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
