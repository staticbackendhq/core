package model

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"github.com/staticbackendhq/core/config"
)

type DatabaseConfig struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"customerId"`
	Name             string    `json:"name"`
	AllowedDomain    []string  `json:"whitelist"`
	IsActive         bool      `json:"-"`
	MonthlySentEmail int       `json:"-"`
	Created          time.Time `json:"created"`
}

type PagedResult struct {
	Page    int64                    `json:"page"`
	Size    int64                    `json:"size"`
	Total   int64                    `json:"total"`
	Results []map[string]interface{} `json:"results"`
}

type ListParams struct {
	Page           int64
	Size           int64
	SortBy         string
	SortDescending bool
}

var (
	HashSecret *jwt.HMACSHA
)

func init() {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) == 0 {
		secret = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	HashSecret = jwt.NewHS256([]byte(secret))
}

const (
	SystemID = "sb"

	MsgTypeError     = "error"
	MsgTypeOk        = "ok"
	MsgTypeEcho      = "echo"
	MsgTypeInit      = "init"
	MsgTypeAuth      = "auth"
	MsgTypeToken     = "token"
	MsgTypeJoin      = "join"
	MsgTypeJoined    = "joined"
	MsgTypePresence  = "presence"
	MsgTypeChanIn    = "chan_in"
	MsgTypeChanOut   = "chan_out"
	MsgTypeDBCreated = "db_created"
	MsgTypeDBUpdated = "db_updated"
	MsgTypeDBDeleted = "db_deleted"
)

type Command struct {
	SID           string `json:"sid"`
	Type          string `json:"type"`
	Data          string `json:"data"`
	Channel       string `json:"channel"`
	Token         string `json:"token"`
	IsSystemEvent bool   `json:"-"`
}

func (msg Command) IsDBEvent() bool {
	switch msg.Type {
	case MsgTypeDBCreated, MsgTypeDBUpdated, MsgTypeDBDeleted:
		return true
	}
	return false
}

func CleanCollectionName(col string) string {
	if strings.EqualFold(config.Current.KeepPermissionInName, "yes") {
		return col
	}

	re := regexp.MustCompile(`_\d\d\d_`)
	return re.ReplaceAllString(col, "")

}
