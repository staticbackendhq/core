package staticbackend

import (
	"os"
	"testing"

	"github.com/staticbackendhq/core/internal"
)

func Test_Sendmail(t *testing.T) {
	data := internal.SendMailData{
		FromName: os.Getenv("FROM_NAME"),
		From:     os.Getenv("FROM_EMAIL"),
		To:       "dominicstpierre+unittest@gmail.com",
		ToName:   "Dominic St-Pierre",
		Subject:  "From unit test",
		HTMLBody: "<h1>hello</h1><p>working</p>",
		TextBody: "Hello\nworking",
		ReplyTo:  os.Getenv("FROM_EMAIL"),
	}
	if err := emailer.Send(data); err != nil {
		t.Error(err)
	}
}
