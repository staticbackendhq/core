package storage

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/model"
)

func TestS3Storage(t *testing.T) {
	config.Current = config.LoadConfig()

	s3store := S3{}

	f, err := os.Open("s3_test.go")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	finfo, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}

	udata := model.UploadFileData{
		FileKey: "unit-test",
		File:    f,
		Size:    finfo.Size(),
	}

	url, err := s3store.Save(udata)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	x, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	} else if !strings.Contains(string(x), "func TestS3Storage") {
		t.Log(url)
		t.Error("could not find func TestS3Storage in stored file.")
		t.Log(string(x))
	}

	if err := s3store.Delete(udata.FileKey); err != nil {
		t.Fatal(err)
	}
}
