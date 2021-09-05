package oauth

import (
	"context"
	"errors"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserInfo struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
}

type InfoGrabber interface {
	Get(client *http.Client, accessToken string) (UserInfo, error)
}

type ExternalLogin struct {
	Provider     string   `bson:"provider" json:"provider"`
	ClientID     string   `bson:"cid" json:"-"`
	ClientSecret string   `bson:"cs" json:"-"`
	Scopes       []string `bson:"scopes" json:"scopes"`
	AuthURL      string   `bson:"authUrl" json:"authUrl"`
	TokenURL     string   `bson:"tokenUrl" json:"tokenUrl"`
}

func SaveConfig(db *mongo.Database, data ExternalLogin) error {
	ctx := context.Background()

	opts := options.Update().SetUpsert(true)
	filter := bson.M{"provider": data.Provider}
	update := bson.M{"$set": bson.M{
		"provider": data.Provider,
		"cid":      data.ClientID,
		"cs":       data.ClientSecret,
		"scopes":   data.Scopes,
		"authUrl":  data.AuthURL,
		"tokenUrl": data.TokenURL,
	}}

	res, err := db.Collection("sb_extlogins").UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	} else if res.UpsertedCount != 1 {
		return errors.New("external login save failed.")
	}
	return nil
}

func GetConfig(db *mongo.Database, provider string) (ExternalLogin, error) {
	ctx := context.Background()

	var data ExternalLogin
	filter := bson.M{"provider": provider}

	sr := db.Collection("sb_extlogins").FindOne(ctx, filter)
	if err := sr.Decode(&data); err != nil {
		return data, err
	} else if err := sr.Err(); err != nil {
		return data, err
	}
	return data, nil

}

func GetInfoGrabber(provider string) InfoGrabber {
	switch provider {
	case "google":
		return &Google{}
	case "twitter":
		return &Twitter{}
	}
	return nil
}
