package main

import (
	"testing"
)

func TestHasPermission(t *testing.T) {
	reads := make(map[string]permissionLevel)
	reads["tbl_740_"] = permGroup
	reads["tbl_600_"] = permOwner
	reads["tbl"] = permGroup
	reads["tbl_226_"] = permEveryone

	for k, v := range reads {
		if p := readPermission(k); v != p {
			t.Errorf("%s expected read to be %v got %v", k, v, p)
		}
	}

	writes := make(map[string]permissionLevel)
	writes["tbl"] = permOwner
	writes["tbl_760_"] = permGroup
	writes["tbl_662_"] = permEveryone
	writes["tbl_244_"] = permOwner

	for k, v := range writes {
		if p := writePermission(k); v != p {
			t.Errorf("%s expected write to be %v got %v", k, v, p)
		}
	}
}
