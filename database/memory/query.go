package memory

import (
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/model"
)

func (m *Memory) ParseQuery(clauses [][]interface{}) (filter map[string]any, err error) {
	filter = make(map[string]any)

	for i, clause := range clauses {
		if len(clause) != 3 {
			err = fmt.Errorf("the %d query clause did not contains the required 3 parameters (field, operator, value)", i+1)
			return
		}

		field, ok := clause[0].(string)
		if !ok {
			err = fmt.Errorf("the %d query clause's field parameter must be a string: %v", i+1, clause[0])
			return
		}

		op, ok := clause[1].(string)
		if !ok {
			err = fmt.Errorf("the %d query clause's operator must be a string: %v", i+1, clause[1])
			return
		}

		switch op {
		case "=", "==":
			filter["= "+field] = clause[2]
		case "!=", "<>":
			filter["!= "+field] = clause[2]
		case ">":
			filter["> "+field] = clause[2]
		case "<":
			filter["< "+field] = clause[2]
		case ">=":
			filter[">= "+field] = clause[2]
		case "<=":
			filter["<= "+field] = clause[2]
		case "in":
			filter["in "+field] = fmt.Sprintf("in %s", clause[2])
		case "!in", "nin":
			filter[field] = clause[2]
		default:
			err = fmt.Errorf("the %d query clause's operator: %s is not supported at the moment", i+1, op)
		}
	}

	return
}

func secureRead(auth model.Auth, col string, list []map[string]any) []map[string]any {
	var filtered []map[string]any

	filter := make(map[string]string)

	// if they're not root and repo is not public
	if !strings.HasPrefix(col, "pub_") && auth.Role < 100 {
		switch internal.ReadPermission(col) {
		case internal.PermGroup:
			filter[FieldAccountID] = auth.AccountID
		case internal.PermOwner:
			filter[FieldAccountID] = auth.AccountID
			filter[FieldOwnerID] = auth.UserID
		}
	}

	for _, doc := range list {
		matches := 0
		for k, v := range filter {
			if doc[k] == v {
				matches++
			}
		}

		if matches == len(filter) {
			filtered = append(filtered, doc)
		}

	}

	return filtered
}

func canWrite(auth model.Auth, col string, doc map[string]any) bool {
	// if they are not "root", we use permission
	if auth.Role < 100 {
		switch internal.WritePermission(col) {
		case internal.PermGroup:
			return doc[FieldAccountID] == auth.AccountID
		case internal.PermOwner:
			return doc[FieldAccountID] == auth.AccountID && doc[FieldOwnerID] == auth.UserID
		}
	}

	return true
}
