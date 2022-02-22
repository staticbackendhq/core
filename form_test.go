package staticbackend

import (
	"net/url"
	"testing"
)

func TestFormSubmission(t *testing.T) {
	val := url.Values{}
	val.Add("name", "unit test")
	val.Add("email", "unit@test.com")

	resp := dbReq(t, submitForm, "POST", "/postform/testform", val, false, true)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	resp2 := dbReq(t, listForm, "GET", "/form?name=testform", nil, true)
	defer resp2.Body.Close()

	var results []map[string]interface{}
	if err := parseBody(resp2.Body, &results); err != nil {
		t.Fatal(err)
	} else if len(results) == 0 {
		t.Errorf("expected to get at least one form submission got 0")
	} else if results[0]["name"] != "unit test" {
		t.Log(results)
		t.Errorf("expected name to be unit test got %v", results[0]["name"])
	} else if results[0]["email"] != "unit@test.com" {
		t.Log(results)
		t.Errorf("expected email to be unit@test.com got %v", results[0]["email"])
	}
}
