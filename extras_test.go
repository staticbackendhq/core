package staticbackend

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/staticbackendhq/core/middleware"
)

func TestUploadAndResizeImage(t *testing.T) {
	var b []byte
	buf := bytes.NewBuffer(b)

	writer := multipart.NewWriter(buf)
	defer writer.Close()

	part, err := writer.CreateFormFile("file", "src.png")
	if err != nil {
		t.Fatal(err)
	}

	src, err := os.Open("./extra/testdata/src.png")
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	if _, err := io.Copy(part, src); err != nil {
		t.Fatal(err)
	}

	ww, err := writer.CreateFormField("width")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ww.Write([]byte("640")); err != nil {
		t.Fatal(err)
	}

	writer.Close()

	req := httptest.NewRequest("POST", "/extra/resizeimg", bytes.NewReader(buf.Bytes()))
	req.Header.Add("Content-Type", writer.FormDataContentType())

	resp := httptest.NewRecorder()

	req.Header.Set("SB-PUBLIC-KEY", pubKey)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

	stdAuth := []middleware.Middleware{
		middleware.WithDB(datastore, volatile),
		middleware.RequireAuth(datastore, volatile),
	}

	h := middleware.Chain(http.HandlerFunc(extexec.resizeImage), stdAuth...)

	h.ServeHTTP(resp, req)

	if resp.Code > 299 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		t.Errorf("expected status < 299 got %s, %s", resp.Result().Status, string(body))
	}
}
