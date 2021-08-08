package internal

import (
	"os"

	"github.com/gbrlsnchs/jwt/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BaseConfig struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	SBID      primitive.ObjectID `bson:"accountId" json:"-"`
	Name      string             `bson:"name" json:"name"`
	Whitelist []string           `bson:"whitelist" json:"whitelist"`
	IsActive  bool               `bson:"active" json:"-"`
}

var (
	Tokens     map[string]Auth       = make(map[string]Auth)
	Bases      map[string]BaseConfig = make(map[string]BaseConfig)
	HashSecret                       = jwt.NewHS256([]byte(os.Getenv("JWT_SECRET")))
)

const (
	FieldID        = "_id"
	FieldAccountID = "accountId"
	FieldOwnerID   = "sb_owner"
	FieldToken     = "token"
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
	MsgTypeChanIn    = "chan_in"
	MsgTypeChanOut   = "chan_out"
	MsgTypeDBCreated = "db_created"
	MsgTypeDBUpdated = "db_updated"
	MsgTypeDBDeleted = "db_deleted"
)

type Command struct {
	SID     string `json:"sid"`
	Type    string `json:"type"`
	Data    string `json:"data"`
	Channel string `json:"channel"`
	Token   string `json:"token"`
}

func (msg Command) IsDBEvent() bool {
	switch msg.Type {
	case MsgTypeDBCreated, MsgTypeDBUpdated, MsgTypeDBDeleted:
		return true
	}
	return false
}
