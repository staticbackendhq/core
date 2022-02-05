package mongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

func (mg *Mongo) ParseQuery(clauses [][]interface{}) (map[string]interface{}, error) {
	filter := bson.M{}
	for i, clause := range clauses {
		if len(clause) != 3 {
			return filter, fmt.Errorf("The %d query clause did not contains the required 3 parameters (field, operator, value)", i+1)
		}

		field, ok := clause[0].(string)
		if !ok {
			return filter, fmt.Errorf("The %d query clause's field parameter must be a string: %v", i+1, clause[0])
		}

		op, ok := clause[1].(string)
		if !ok {
			return filter, fmt.Errorf("The %d query clause's operator must be a string: %v", i+1, clause[1])
		}

		switch op {
		case "=", "==":
			filter[field] = clause[2]
		case "!=", "<>":
			filter[field] = bson.M{"$ne": clause[2]}
		case ">":
			filter[field] = bson.M{"$gt": clause[2]}
		case "<":
			filter[field] = bson.M{"$lt": clause[2]}
		case ">=":
			filter[field] = bson.M{"$gte": clause[2]}
		case "<=":
			filter[field] = bson.M{"$lte": clause[2]}
		case "in":
			filter[field] = bson.M{"$in": clause[2]}
		case "!in", "nin":
			filter[field] = bson.M{"$nin": clause[2]}
		default:
			return filter, fmt.Errorf("The %d query clause's operator: %s is not supported at the moment.", i+1, op)
		}
	}

	return filter, nil
}
