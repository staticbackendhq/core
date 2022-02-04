package staticbackend

import (
	"testing"

	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/internal"
)

func Test_Sendmail_AWS(t *testing.T) {
	emailer := &email.AWSSES{}

	data := internal.SendMailData{
		FromName: "My name here",
		From:     "delivery@tangara.io",
		To:       "dominicstpierre@gmail.com",
		ToName:   "Dominic St-Pierre",
		Subject:  "From unit test",
		HTMLBody: "<h1>hello</h1><p>working</p>",
		TextBody: "Hello\nworking",
		ReplyTo:  "dominic@focuscentric.com",
	}
	if err := emailer.Send(data); err != nil {
		t.Error(err)
	}
}
