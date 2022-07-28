package staticbackend

import (
	"strings"
	"testing"

	"github.com/staticbackendhq/core/internal"
)

func TestGetCurrentAuthUser(t *testing.T) {
	resp := dbReq(t, mship.me, "GET", "/me", nil)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var me internal.Auth
	if err := parseBody(resp.Body, &me); err != nil {
		t.Fatal(err)
	} else if !strings.EqualFold(me.Email, admEmail) {
		t.Errorf("expected email to be %s got %s", admEmail, me.Email)
	} else if me.Role != 100 {
		t.Errorf("expected role to be 100 got %d", me.Role)
	}
}
