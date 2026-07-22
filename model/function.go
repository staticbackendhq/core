package model

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/staticbackendhq/core/config"
)

var ErrInvalidFunctionSecretKey = errors.New("APP_SECRET must be set to a 16, 24, or 32 byte value before using function secrets")

// ExecData represents a server-side function with its name, code and execution
// history
type ExecData struct {
	ID           string        `json:"id"`
	AccountID    string        `json:"accountId"`
	FunctionName string        `json:"name"`
	TriggerTopic string        `json:"trigger"`
	Code         string        `json:"code"`
	Secrets      []byte        `json:"-"`
	Version      int           `json:"version"`
	LastUpdated  time.Time     `json:"lastUpdated"`
	LastRun      time.Time     `json:"lastRun"`
	History      []ExecHistory `json:"history"`
}

type FunctionUpdate struct {
	ID            string
	Code          string
	TriggerTopic  string
	Secrets       []byte
	UpdateSecrets bool
}

func EncryptFunctionSecrets(raw string) ([]byte, error) {
	secrets, err := ParseFunctionSecrets(raw)
	if err != nil {
		return nil, err
	}
	if len(secrets) == 0 {
		return nil, nil
	}

	b, err := json.Marshal(secrets)
	if err != nil {
		return nil, err
	}

	c, err := newFunctionSecretsCipher()
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

func ParseFunctionSecrets(raw string) (map[string]string, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, err
	}

	secrets := make(map[string]string, len(values))
	for key := range values {
		if key == "" {
			continue
		}
		secrets[key] = values.Get(key)
	}
	return secrets, nil
}

func (ex ExecData) GetSecrets() (map[string]string, error) {
	if len(ex.Secrets) == 0 {
		return make(map[string]string), nil
	}

	c, err := newFunctionSecretsCipher()
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ex.Secrets) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ex.Secrets[:nonceSize], ex.Secrets[nonceSize:]
	b, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	secrets := make(map[string]string)
	if err := json.Unmarshal(b, &secrets); err != nil {
		return nil, err
	}
	return secrets, nil
}

func FunctionHistoryRetentionCutoff() time.Time {
	return time.Now().AddDate(0, 0, -config.Current.FunctionHistoryRetentionDays)
}

func newFunctionSecretsCipher() (cipher.Block, error) {
	key := []byte(config.Current.AppSecret)
	switch len(key) {
	case 16, 24, 32:
	default:
		return nil, fmt.Errorf("%w; got %d bytes", ErrInvalidFunctionSecretKey, len(key))
	}

	return aes.NewCipher(key)
}

// ExecHistory represents a function run ending result
type ExecHistory struct {
	ID         string    `json:"id"`
	FunctionID string    `json:"functionId"`
	Version    int       `json:"version"`
	Started    time.Time `json:"started"`
	Completed  time.Time `json:"completed"`
	Success    bool      `json:"success"`
	Output     []string  `json:"output"`
}

const (
	TaskTypeFunction = "function"
	TaskTypeMessage  = "message"
	TaskTypeHTTP     = "http"
)

type Task struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Value    string    `json:"value"`
	Meta     string    `json:"meta"`
	Interval string    `json:"interval"`
	LastRun  time.Time `json:"last"`

	BaseName string `json:"base"`
}

type MetaMessage struct {
	Data        string `json:"data"`
	Channel     string `json:"channel"`
	HTTPMethod  string `json:"method"`
	ContentType string `json:"ct"`
	HTTPHeaders string `jsno:"headers"`
}
