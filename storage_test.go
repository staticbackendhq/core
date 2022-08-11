package staticbackend

import (
	"testing"
)

func TestCleanUpFileName(t *testing.T) {
	fakeNames := make(map[string]string)
	fakeNames[""] = ""
	fakeNames["abc.def"] = "abc"
	fakeNames["ok!.test"] = "ok"
	fakeNames["@file-name_here!.ext"] = "file-name_here"

	for k, v := range fakeNames {
		if clean := cleanUpFileName(k); clean != v {
			t.Errorf("expected %s got %s", v, clean)
		}
	}
}
