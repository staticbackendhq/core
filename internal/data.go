package internal

import (
	"os"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
)

type BaseConfig struct {
	ID            string    `bson:"_id" json:"id"`
	CustomerID    string    `bson:"accountId" json:"-"`
	Name          string    `bson:"name" json:"name"`
	AllowedDomain []string  `bson:"whitelist" json:"whitelist"`
	IsActive      bool      `bson:"active" json:"-"`
	Created       time.Time `json:"created"`
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
	//Tokens     map[string]Auth       = make(map[string]Auth)
	//Bases      map[string]BaseConfig = make(map[string]BaseConfig)
	HashSecret = jwt.NewHS256([]byte(os.Getenv("JWT_SECRET")))
)

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
