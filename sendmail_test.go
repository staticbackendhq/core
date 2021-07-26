package staticbackend

import "testing"

func TestSendingEmail(t *testing.T) {
	t.Skip()

	if err := sendMail("dominicstpierre@gmail.com", "Dominic", "support@staticbackend.com", "StaticBackend", "unit test", "working", ""); err != nil {
		t.Error(err)
	}
}
