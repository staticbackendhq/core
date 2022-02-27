package internal

import "testing"

func TestReadPermissions(t *testing.T) {
	tables := make(map[string]PermissionLevel)
	tables["normal"] = PermGroup
	tables["only-owner-read_700_"] = PermOwner
	tables["same-acct-users_770_"] = PermGroup
	tables["logged-in_774_"] = PermEveryone

	for col, perm := range tables {
		if p := ReadPermission(col); p != perm {
			t.Errorf("%s: expected read perm %d got %d", col, perm, p)
		}
	}
}

func TestWritePermissions(t *testing.T) {
	tables := make(map[string]PermissionLevel)
	tables["normal"] = PermOwner
	tables["same-acct-users_770_"] = PermGroup
	tables["only-owner-write_700_"] = PermOwner
	tables["logged-in_772_"] = PermEveryone

	for col, perm := range tables {
		if p := WritePermission(col); p != perm {
			t.Errorf("%s: expected write perm %d got %d", col, perm, p)
		}
	}
}
