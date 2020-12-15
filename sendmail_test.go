package main

import "testing"

func TestSendingEmail(t *testing.T) {
	if err := sendMail("dominicstpierre@gmail.com", "Dominic", "support@staticbackend.com", "StaticBackend", "unit test", "working", ""); err != nil {
		t.Error(err)
	}
}
