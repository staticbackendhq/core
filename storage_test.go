package staticbackend

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"
)

func TestFileUpload(t *testing.T) {
	pr, pw := io.Pipe()

	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		part, err := writer.CreateFormFile("file", "upload.test")
		if err != nil {
			t.Error(err)
		}

		if _, err := part.Write([]byte("testing file upload")); err != nil {
			t.Error(err)
		}
	}()

	req := httptest.NewRequest("POST", "/storage/upload", pr)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	req.Header.Set("SB-PUBLIC-KEY", pubKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

	stdAuth := []middleware.Middleware{
		middleware.WithDB(backend.DB, backend.Cache, getStripePortalURL),
		middleware.RequireAuth(backend.DB, backend.Cache),
	}

	// prevent DB (SQLite) from being busy
	time.Sleep(35 * time.Millisecond)

	w := httptest.NewRecorder()
	h := middleware.Chain(http.HandlerFunc(upload), stdAuth...)
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var data backend.SavedFile
	if err := parseBody(w.Result().Body, &data); err != nil {
		t.Error(err)
	}
	defer w.Result().Body.Close()

	t.Log(data)

	// let's remove the web-based prefix to test if file was saved
	localFilePath := strings.Replace(data.URL, "http://localhost:8099/localfs", "", -1)

	localFilePath = path.Join(os.TempDir(), localFilePath)

	if _, err := os.Stat(localFilePath); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exists", localFilePath)
	}

	// test the delete file endpoint
	delPath := fmt.Sprintf("/sudostorage/delete?id=%s", data.ID)
	resp := dbReq(t, deleteFile, "DELETE", delPath, nil)
	if resp.StatusCode != 200 {
		t.Errorf("Expected 200, go %d", resp.StatusCode)
	}

	// the file should not exists anymore
	if _, err := os.Stat(localFilePath); !os.IsNotExist(err) {
		t.Errorf("Expected file %s to not exists", localFilePath)
	}
}

func TestCleanUpFileName(t *testing.T) {
	fakeNames := make(map[string]string)
	fakeNames[""] = ""
	fakeNames["abc.def"] = "abc"
	fakeNames["ok!.test"] = "ok"
	fakeNames["@file-name_here!.ext"] = "file-name_here"

	for k, v := range fakeNames {
		if clean := internal.CleanUpFileName(k); clean != v {
			t.Errorf("expected %s got %s", v, clean)
		}
	}
}
