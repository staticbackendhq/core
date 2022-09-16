package mongo

import (
	"time"

	"github.com/staticbackendhq/core/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LocalExecData struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	FunctionName string             `bson:"name" json:"name"`
	TriggerTopic string             `bson:"tr" json:"trigger"`
	Code         string             `bson:"code" json:"code"`
	Version      int                `bson:"v" json:"version"`
	LastUpdated  time.Time          `bson:"lu" json:"lastUpdated"`
	LastRun      time.Time          `bson:"lr" json:"lastRun"`
	History      []LocalExecHistory `bson:"h" json:"history"`
}

type LocalExecHistory struct {
	ID        string    `bson:"id" json:"id"`
	Version   int       `bson:"v" json:"version"`
	Started   time.Time `bson:"s" json:"started"`
	Completed time.Time `bson:"c" json:"completed"`
	Success   bool      `bson:"ok" json:"success"`
	Output    []string  `bson:"out" json:"output"`
}

func toLocalExecData(ex model.ExecData) LocalExecData {
	oid, err := primitive.ObjectIDFromHex(ex.ID)
	if err != nil {
		//TODO: fin a way to handle this error properly
		return LocalExecData{}
	}

	return LocalExecData{
		ID:           oid,
		FunctionName: ex.FunctionName,
		TriggerTopic: ex.TriggerTopic,
		Code:         ex.Code,
		Version:      ex.Version,
		LastUpdated:  ex.LastUpdated,
		LastRun:      ex.LastRun,
		History:      toLocalExecHistory(ex.History),
	}
}

func toLocalExecHistory(h []model.ExecHistory) []LocalExecHistory {
	lh := make([]LocalExecHistory, 0)

	for _, exh := range h {
		lh = append(lh, LocalExecHistory{
			ID:        exh.ID,
			Version:   exh.Version,
			Started:   exh.Started,
			Completed: exh.Completed,
			Success:   exh.Success,
			Output:    exh.Output,
		})
	}

	return lh
}

func fromLocalExecData(lex LocalExecData) model.ExecData {
	return model.ExecData{
		ID:           lex.ID.Hex(),
		FunctionName: lex.FunctionName,
		TriggerTopic: lex.TriggerTopic,
		Code:         lex.Code,
		Version:      lex.Version,
		LastUpdated:  lex.LastUpdated,
		LastRun:      lex.LastRun,
		History:      fromLocalExecHistory(lex.History),
	}
}

func fromLocalExecHistory(lh []LocalExecHistory) []model.ExecHistory {
	var h []model.ExecHistory
	for _, exh := range lh {
		h = append(h, model.ExecHistory{
			ID:        exh.ID,
			Version:   exh.Version,
			Started:   exh.Started,
			Completed: exh.Completed,
			Success:   exh.Success,
			Output:    exh.Output,
		})
	}

	return h
}

func (mg *Mongo) AddFunction(dbName string, data model.ExecData) (string, error) {
	db := mg.Client.Database(dbName)

	data.ID = primitive.NewObjectID().Hex()
	data.Version = 1
	data.LastUpdated = time.Now()
	data.History = make([]model.ExecHistory, 0)

	lex := toLocalExecData(data)

	_, err := db.Collection("sb_functions").InsertOne(mg.Ctx, lex)
	if err != nil {
		return "", err
	}

	/*oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", nil
	}*/

	return data.ID, nil
}

func (mg *Mongo) UpdateFunction(dbName, id, code, trigger string) error {
	db := mg.Client.Database(dbName)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{"code": code, "lu": time.Now(), "tr": trigger},
		"$inc": bson.M{"v": 1},
	}
	filter := bson.M{FieldID: oid}

	res := db.Collection("sb_functions").FindOneAndUpdate(mg.Ctx, filter, update)
	if err := res.Err(); err != nil {
		return err
	}

	return nil
}

func (mg *Mongo) GetFunctionForExecution(dbName, name string) (result model.ExecData, err error) {
	db := mg.Client.Database(dbName)

	filter := bson.M{"name": name}

	opt := &options.FindOneOptions{}
	opt.SetProjection(bson.M{"h": false})

	var lex LocalExecData
	sr := db.Collection("sb_functions").FindOne(mg.Ctx, filter, opt)
	if err := sr.Decode(&lex); err != nil {
		return result, err
	} else if err := sr.Err(); err != nil {
		return result, err
	}

	result = fromLocalExecData(lex)
	return result, nil
}

func (mg *Mongo) GetFunctionByID(dbName, id string) (result model.ExecData, err error) {
	db := mg.Client.Database(dbName)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return
	}

	filter := bson.M{FieldID: oid}

	var lex LocalExecData
	sr := db.Collection("sb_functions").FindOne(mg.Ctx, filter)
	if err = sr.Decode(&lex); err != nil {
		return
	} else if err = sr.Err(); err != nil {
		return
	}

	result = fromLocalExecData(lex)
	return
}

func (mg *Mongo) GetFunctionByName(dbName, name string) (result model.ExecData, err error) {
	db := mg.Client.Database(dbName)

	filter := bson.M{"name": name}

	var lex LocalExecData
	sr := db.Collection("sb_functions").FindOne(mg.Ctx, filter)
	if err := sr.Decode(&lex); err != nil {
		return result, err
	} else if err := sr.Err(); err != nil {
		return result, err
	}
	result = fromLocalExecData(lex)
	return
}

func (mg *Mongo) ListFunctions(dbName string) (results []model.ExecData, err error) {
	db := mg.Client.Database(dbName)

	opt := &options.FindOptions{}
	opt.SetProjection(bson.M{"h": 0})

	cur, err := db.Collection("sb_functions").Find(mg.Ctx, bson.M{}, opt)
	if err != nil {
		return
	}
	defer cur.Close(mg.Ctx)

	for cur.Next(mg.Ctx) {
		var lex LocalExecData
		err = cur.Decode(&lex)
		if err != nil {
			return
		}

		results = append(results, fromLocalExecData(lex))
	}

	return
}

func (mg *Mongo) ListFunctionsByTrigger(dbName, trigger string) (results []model.ExecData, err error) {
	db := mg.Client.Database(dbName)

	opt := &options.FindOptions{}
	opt.SetProjection(bson.M{"h": 0})

	filter := bson.M{"tr": trigger}

	cur, err := db.Collection("sb_functions").Find(mg.Ctx, filter, opt)
	if err != nil {
		return
	}
	defer cur.Close(mg.Ctx)

	for cur.Next(mg.Ctx) {
		var lex LocalExecData
		if err = cur.Decode(&lex); err != nil {
			return
		}

		results = append(results, fromLocalExecData(lex))
	}

	return
}

func (mg *Mongo) DeleteFunction(dbName, name string) error {
	db := mg.Client.Database(dbName)

	filter := bson.M{"name": name}

	if _, err := db.Collection("sb_functions").DeleteOne(mg.Ctx, filter); err != nil {
		return err
	}
	return nil
}

func (mg *Mongo) RanFunction(dbName, id string, rh model.ExecHistory) error {
	db := mg.Client.Database(dbName)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	rh.ID = primitive.NewObjectID().Hex()
	leh := toLocalExecHistory([]model.ExecHistory{rh})[0]

	filter := bson.M{FieldID: oid}
	update := bson.M{
		"$set":  bson.M{"lr": time.Now()},
		"$push": bson.M{"h": leh},
	}
	res := db.Collection("sb_functions").FindOneAndUpdate(mg.Ctx, filter, update)
	if err := res.Err(); err != nil {
		return err
	}
	return nil
}
