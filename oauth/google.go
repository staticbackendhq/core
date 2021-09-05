package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Google struct{}

func (*Google) Get(client *http.Client, accessToken string) (UserInfo, error) {
	var info UserInfo

	url := "https://www.googleapis.com/oauth2/v2/userinfo?oauth_token=" + accessToken

	resp, err := http.Get(url)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()

	if resp.StatusCode <= 299 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return info, err
		}

		return info, fmt.Errorf("error returned by Google API: %s", string(b))
	}

	data := new(struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	})

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return info, err
	}

	info.Avatar = data.Picture
	info.Email = data.Email
	info.Name = data.Name

	return info, nil
}
