package mongo

import (
	"github.com/staticbackendhq/core/model"
)

func (mg *Mongo) Count(auth model.Auth, dbName, col string, filter map[string]interface{}) (count int64, err error) {
	db := mg.Client.Database(dbName)

	acctID, userID, err := parseObjectID(auth)
	if err != nil {
		return
	}

	secureRead(acctID, userID, auth.Role, col, filter)

	count, err = db.Collection(model.CleanCollectionName(col)).CountDocuments(mg.Ctx, filter)
	if err != nil {
		return -1, err
	}

	return count, nil
}
