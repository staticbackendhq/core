package model

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"github.com/staticbackendhq/core/config"
)

// Auth represents an authenticated user.
type Auth struct {
	AccountID string `json:"accountId"`
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	Role      int    `json:"role"`
	Token     string `json:"-"`
	Plan      int    `json:"-"`
}

func (auth Auth) ReconstructToken() string {
	if strings.HasPrefix(auth.Token, "__tmp__experimental_public") {
		return auth.Token
	}
	return fmt.Sprintf("%s|%s", auth.UserID, auth.Token)
}

// JWTPayload contains the current user token
type JWTPayload struct {
	jwt.Payload
	Token string `json:"token,omitempty"`
}

var (
	ctx = context.Background()
)

type Account struct {
	ID      string    `json:"id"`
	Email   string    `json:"email"`
	Created time.Time `json:"created"`
}

type User struct {
	ID        string    `json:"id"`
	AccountID string    `json:"accountId"`
	Token     string    `json:"token"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Role      int       `json:"role"`
	ResetCode string    `json:"-"`
	Created   time.Time `json:"created"`
}

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

const (
	PlanFree = iota
	PlanIdea
	PleanLaunch
	PlanTraction
	PlanGrowth
)

type Tenant struct {
	ID               string    `bson:"_id" json:"id"`
	Email            string    `bson:"email" json:"email"`
	StripeID         string    `bson:"stripeId" json:"stripeId"`
	SubscriptionID   string    `bson:"subId" json:"subId"`
	Plan             int       `json:"plan"`
	IsActive         bool      `bson:"active" json:"-"`
	MonthlyEmailSent int       `bson:"mes" json:"-"`
	Created          time.Time `bson:"created" json:"created"`
	ExternalLogins   []byte    `json:"-"`
}

func EncryptExternalLogins(tokens map[string]OAuthConfig) ([]byte, error) {
	key := []byte(config.Current.AppSecret)

	b, err := json.Marshal(tokens)
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, b, nil), nil
}

func (cus *Tenant) GetExternalLogins() (map[string]OAuthConfig, error) {
	key := []byte(config.Current.AppSecret)

	ciphertext := cus.ExternalLogins
	if len(ciphertext) == 0 {
		return make(map[string]OAuthConfig), nil
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	b, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	m := make(map[string]OAuthConfig)
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	return m, nil
}

func (cus *Tenant) GetProvider(provider string) (cfg OAuthConfig, ok bool) {
	logins, err := cus.GetExternalLogins()
	if err != nil {
		return
	}

	cfg, ok = logins[provider]
	return
}

type OAuthConfig struct {
	ConsumerKey    string
	ConsumerSecret string
}
