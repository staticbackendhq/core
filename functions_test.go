package staticbackend

import (
	"io"
	"net/http"
	"testing"
)

func TestFunctionsExecuteGetByID(t *testing.T) {
	data := ExecData{FunctionName: "unittest"}
	resp := dbPost(t, funexec.exec, "", data)
	if resp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		t.Log(string(b))
		t.Errorf("expected status 200 got %s", resp.Status)
	}
}
