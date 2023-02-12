package internal

import "testing"

func TestRandStringRunes(t *testing.T) {
	lengths := []int{1, 1, 12, 30}
	for _, length := range lengths {
		if p := RandStringRunes(length); len(p) != length {
			t.Errorf("%s: expected length %d got %d", p, length, len(p))
		}
	}
}

func TestCleanUpFileName(t *testing.T) {
	tables := make(map[string]string)
	tables["dummy_1_.sample"] = "dummy__"
	tables["/someWhere/dummy_2_.sample"] = "someWheredummy__"
	tables[".Dummy3.sample"] = "Dummy"

	for name, cleanedName := range tables {
		if p := CleanUpFileName(name); p != cleanedName {
			t.Errorf("%s: expected file name %s got %s", name, cleanedName, p)
		}

	}
}
