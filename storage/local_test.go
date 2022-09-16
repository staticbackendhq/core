package storage

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/staticbackendhq/core/model"
)

func TestLocalSave(t *testing.T) {
	rdr := bytes.NewReader([]byte("unit test"))

	local := Local{}

	data := model.UploadFileData{FileKey: "unit/test/file.txt", File: rdr}
	url, err := local.Save(data)
	if err != nil {
		t.Fatal(err)
	} else if !strings.Contains(url, "/unit/test/file.txt") {
		fmt.Errorf("expected ~/tmp/unit/test/file.txt got %s", url)
	}

	fmt.Println(url)
}
