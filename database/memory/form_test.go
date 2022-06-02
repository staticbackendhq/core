package memory

import (
	"testing"
)

func TestForm(t *testing.T) {
	doc := make(map[string]interface{})
	doc["name"] = "unit"
	doc["email"] = "unit@test.com"

	if err := datastore.AddFormSubmission(confDBName, "test", doc); err != nil {
		t.Fatal(err)
	}

	results, err := datastore.ListFormSubmissions(confDBName, "test")
	if err != nil {
		t.Fatal(err)
	} else if len(results) != 1 {
		t.Errorf("expected results to have 1 item, got %d", len(results))
	} else if results[0]["name"] != "unit" || results[0]["email"] != "unit@test.com" {
		t.Errorf("forms data is not as expected %v", results)
	}

	forms, err := datastore.GetForms(confDBName)
	if err != nil {
		t.Fatal(err)
	} else if len(forms) != 1 {
		t.Errorf("expected forms to have len 1 got %d", len(forms))
	} else if forms[0] != "test" {
		t.Errorf("expected forms[0] to be test got %s", forms[0])
	}
}
