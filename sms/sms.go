package sms

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type SMSData struct {
	AccountSID string `json:"accountSID"`
	AuthToken  string `json:"authToken"`
	ToNumber   string `json:"toNumber"`
	FromNumber string `json:"fromNumber"`
	Body       string `json:"body"`
}

// Send sends a text-message using Twilio
func Send(data SMSData) error {
	apiURL := "https://api.twilio.com/2010-04-01/Accounts/" + data.AccountSID + "/Messages.json"

	// Build out the data for the message
	v := url.Values{}
	v.Set("To", data.ToNumber)
	v.Set("From", data.FromNumber)
	v.Set("Body", data.Body)
	rb := strings.NewReader(v.Encode())

	client := &http.Client{}

	req, err := http.NewRequest("POST", apiURL, rb)
	if err != nil {
		return err
	}

	req.SetBasicAuth(data.AccountSID, data.AuthToken)

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("error returned by Twilio: %s", string(b))
	}
	return nil
}
