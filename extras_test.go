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

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/sms"
)

func TestUploadAndResizeImage(t *testing.T) {
	var b []byte
	buf := bytes.NewBuffer(b)

	writer := multipart.NewWriter(buf)
	defer func() { _ = writer.Close() }()

	part, err := writer.CreateFormFile("file", "src.png")
	if err != nil {
		t.Fatal(err)
	}

	src, err := os.Open("./extra/testdata/src.png")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = src.Close() }()

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

	_ = writer.Close()

	req := httptest.NewRequest("POST", "/extra/resizeimg", bytes.NewReader(buf.Bytes()))
	req.Header.Add("Content-Type", writer.FormDataContentType())

	resp := httptest.NewRecorder()

	req.Header.Set("SB-PUBLIC-KEY", pubKey)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

	stdAuth := []middleware.Middleware{
		middleware.WithDB(backend.DB, backend.Cache, getStripePortalURL),
		middleware.RequireAuth(backend.DB, backend.Cache),
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

func TestSudoSendSMS(t *testing.T) {
	// get Twilio's AccountSID and AuthToken from env var
	// if not present, we skip this test
	aID, twiToken := config.Current.TwilioAccountID, config.Current.TwilioAuthToken
	if len(aID) == 0 || len(twiToken) == 0 {
		t.Skip("missing Twilio AccountSID and/or AuthToken")
	}

	to := config.Current.TwilioTestCellNumber
	from := config.Current.TwilioNumber

	data := sms.SMSData{
		AccountSID: aID,
		AuthToken:  twiToken,
		ToNumber:   to,
		FromNumber: from,
		Body:       "from unit test of StaticBackend",
	}

	resp := dbReq(t, extexec.sudoSendSMS, "POST", "/extra/sms", data, true)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode > 299 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		t.Log(string(b))
		t.Errorf("expected status 200 got %s", resp.Status)
	}
}

func TestHtmlToPDF(t *testing.T) {
	// TODO: this is intermitant and when it failes it's with that
	// error line:128: context deadline exceeded

	data := ConvertParams{
		ToPDF: true,
		URL:   htmlToXTestURL(t),
	}

	resp := dbReq(t, extexec.htmlToX, "POST", "/extras/htmltox", data)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode > 299 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		t.Log(string(b))
		t.Errorf("expected status 200 got %s", resp.Status)
	}
}

func TestHtmlToPNG(t *testing.T) {
	// TODO: this test failed intermitently:
	// error: extras_test.go:157: context deadline exceeded
	//
	// we need to determine why it's doing this and remove the Skip

	data := ConvertParams{
		ToPDF:    false,
		URL:      htmlToXTestURL(t),
		FullPage: true,
	}

	resp := dbReq(t, extexec.htmlToX, "POST", "/extras/htmltox", data)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode > 299 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		t.Log(string(b))
		t.Errorf("expected status 200 got %s", resp.Status)
	}
}

func htmlToXTestURL(t *testing.T) string {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, `<!doctype html>
<html>
<head><title>StaticBackend HTML conversion test</title></head>
<body>
<main>
<h1>StaticBackend HTML conversion test</h1>
<p>This local page keeps PDF and PNG conversion tests independent from external network availability.</p>
</main>
</body>
</html>`)
	}))
	t.Cleanup(srv.Close)

	return srv.URL
}
