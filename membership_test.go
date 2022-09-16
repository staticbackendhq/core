package staticbackend

import (
	"fmt"
	"strings"
	"testing"

	"github.com/staticbackendhq/core/model"
)

func TestGetCurrentAuthUser(t *testing.T) {
	resp := dbReq(t, mship.me, "GET", "/me", nil)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var me model.Auth
	if err := parseBody(resp.Body, &me); err != nil {
		t.Fatal(err)
	} else if !strings.EqualFold(me.Email, admEmail) {
		t.Errorf("expected email to be %s got %s", admEmail, me.Email)
	} else if me.Role != 100 {
		t.Errorf("expected role to be 100 got %d", me.Role)
	}
}

func TestMagicLink(t *testing.T) {
	// note, even though the magic link route allow public (un-unthenticated) req
	// I'm using dbReq (which enforce the stdauth) for ease of re-using the
	// function.

	data := new(struct {
		FromEmail string `json:"fromEmail"`
		FromName  string `json:"fromName"`
		Email     string `json:"email"`
		Subject   string `json:"subject"`
		Body      string `json:"body"`
		MagicLink string `json:"link"`
	})

	data.FromEmail = "unit@test.com"
	data.FromName = "unit test"
	data.Email = admEmail
	data.Subject = "Magic link from unit test"
	data.Body = "<p>Hello</p><p>Please click on the following link to sign-in</p><p>[ink]</p>"
	data.MagicLink = "https://mycustom.link/with-code"

	resp := dbReq(t, mship.magicLink, "POST", "/login/magic", data)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	// in dev mode, the code is always 666333
	u := fmt.Sprintf("/login/magic?email=%s&code=666333", admEmail)
	resp2 := dbReq(t, mship.magicLink, "GET", u, nil)
	defer resp2.Body.Close()

	if resp2.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp2))
	}
}
