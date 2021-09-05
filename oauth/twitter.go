package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Twitter struct{}

func (*Twitter) Get(client *http.Client, accessToken string) (UserInfo, error) {
	var info UserInfo

	url := "https://api.twitter.com/1.1/account/verify_credentials.json?skip_status=true&include_email=true"

	resp, err := client.Get(url)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 299 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return info, err
		}

		return info, fmt.Errorf("error from Twitter: %s", string(b))
	}

	data := new(struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		ImageURL string `json:"profile_image_url_https"`
	})
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return info, err
	}

	info.Email = data.Email
	info.Name = data.Name
	info.Avatar = data.ImageURL

	return info, nil
}
