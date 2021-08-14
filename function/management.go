package function

import (
	"context"
	"staticbackend/internal"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ExecData represents a server-side function with its name, code and execution
// history
type ExecData struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	FunctionName string             `bson:"name" json:"name"`
	TriggerTopic string             `bson:"tr" json:"trigger"`
	Code         string             `bson:"code" json:"code"`
	Version      int                `bson:"v" json:"version"`
	LastUpdated  time.Time          `bson:"lu" json:"lastUpdated"`
	LastRun      time.Time          `bson:"lr" json:"lastRun"`
	History      []ExecHistory      `bson:"h" json:"history"`
}

// ExecHistory represents a function run ending result
type ExecHistory struct {
	ID        string    `bson:"id" json:"id"`
	Version   int       `bson:"v" json:"version"`
	Started   time.Time `bson:"s" json:"started"`
	Completed time.Time `bson:"c" json:"completed"`
	Success   bool      `bson:"ok" json:"success"`
	Output    []string  `bson:"out" json:"output"`
}

func Add(db *mongo.Database, data ExecData) (string, error) {
	data.ID = primitive.NewObjectID()
	data.Version = 1
	data.LastUpdated = time.Now()

	ctx := context.Background()
	res, err := db.Collection("sb_functions").InsertOne(ctx, data)
	if err != nil {
		return "", err
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", nil
	}
	return oid.Hex(), nil
}

func Update(db *mongo.Database, id, code, trigger string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{"code": code, "lu": time.Now(), "tr": trigger},
		"$inc": bson.M{"v": 1},
	}
	filter := bson.M{internal.FieldID: oid}

	ctx := context.Background()
	res := db.Collection("sb_functions").FindOneAndUpdate(ctx, filter, update)
	if err := res.Err(); err != nil {
		return err
	}

	return nil
}

func GetForExecution(db *mongo.Database, name string) (ExecData, error) {
	var result ExecData

	filter := bson.M{"name": name}

	ctx := context.Background()
	opt := &options.FindOneOptions{}
	opt.SetProjection(bson.M{"h": -1})

	sr := db.Collection("sb_functions").FindOne(ctx, filter, opt)
	if err := sr.Decode(&result); err != nil {
		return result, err
	} else if err := sr.Err(); err != nil {
		return result, err
	}

	return result, nil
}

func GetByID(db *mongo.Database, id string) (ExecData, error) {
	var result ExecData

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return result, err
	}

	filter := bson.M{internal.FieldID: oid}

	ctx := context.Background()
	sr := db.Collection("sb_functions").FindOne(ctx, filter)
	if err := sr.Decode(&result); err != nil {
		return result, err
	} else if err := sr.Err(); err != nil {
		return result, err
	}

	return result, nil
}

func List(db *mongo.Database) ([]ExecData, error) {
	opt := &options.FindOptions{}
	opt.SetProjection(bson.M{"h": 0})

	ctx := context.Background()
	cur, err := db.Collection("sb_functions").Find(ctx, bson.M{}, opt)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []ExecData
	for cur.Next(ctx) {
		var ed ExecData
		err := cur.Decode(&ed)
		if err != nil {
			return nil, err
		}

		results = append(results, ed)
	}

	return results, nil
}

func ListByTrigger(db *mongo.Database, trigger string) ([]ExecData, error) {
	opt := &options.FindOptions{}
	opt.SetProjection(bson.M{"h": 0})

	filter := bson.M{"tr": trigger}
	ctx := context.Background()
	cur, err := db.Collection("sb_functions").Find(ctx, filter, opt)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []ExecData
	for cur.Next(ctx) {
		var ed ExecData
		err := cur.Decode(&ed)
		if err != nil {
			return nil, err
		}

		results = append(results, ed)
	}

	return results, nil
}

func Delete(db *mongo.Database, name string) error {
	filter := bson.M{"name": name}

	ctx := context.Background()
	if _, err := db.Collection("sb_functions").DeleteOne(ctx, filter); err != nil {
		return err
	}

	return nil
}
