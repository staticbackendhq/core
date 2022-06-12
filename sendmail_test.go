package staticbackend

import (
	"testing"

	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/internal"
)

func Test_Sendmail(t *testing.T) {
	data := internal.SendMailData{
		FromName: config.Current.FromName,
		From:     config.Current.FromEmail,
		To:       "dominicstpierre+unittest@gmail.com",
		ToName:   "Dominic St-Pierre",
		Subject:  "From unit test",
		HTMLBody: "<h1>hello</h1><p>working</p>",
		TextBody: "Hello\nworking",
		ReplyTo:  config.Current.FromEmail,
	}
	if err := emailer.Send(data); err != nil {
		t.Error(err)
	}
}
