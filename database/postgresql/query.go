package postgresql

import (
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/model"
)

func (mg *PostgreSQL) ParseQuery(clauses [][]interface{}) (map[string]interface{}, error) {
	filter := make(map[string]interface{})

	for i, clause := range clauses {
		if len(clause) != 3 {
			return filter, fmt.Errorf("the %d query clause did not contains the required 3 parameters (field, operator, value)", i+1)
		}

		field, ok := clause[0].(string)
		if !ok {
			return filter, fmt.Errorf("The %d query clause's field parameter must be a string: %v", i+1, clause[0])
		}

		field = fmt.Sprintf(`data->>'%s'`, field)

		op, ok := clause[1].(string)
		if !ok {
			return filter, fmt.Errorf("The %d query clause's operator must be a string: %v", i+1, clause[1])
		}

		switch op {
		case "=", "==":
			filter[field+" = "] = clause[2]
		case "!=", "<>":
			filter[field+" != "] = clause[2]
		case ">", "<", ">=", "<=":
			filter[field+" "+op+" "] = clause[2]
		case "in", "!in":
			//TODO: Implement the value in array or value not in array
			return filter, fmt.Errorf("array lookup is not implemented yet for PostgreSQL at %d op: %s.", i+1, op)
		default:
			return filter, fmt.Errorf("The %d query clause's operator: %s is not supported at the moment.", i+1, op)
		}
	}

	return filter, nil
}

func applyFilter(where string, filters map[string]interface{}) string {
	for field, val := range filters {
		where += fmt.Sprintf(" AND %s '%v'", field, val)
	}
	return where
}

func secureRead(auth model.Auth, col string) string {
	if strings.HasPrefix(col, "pub_") || auth.Role == 100 {
		return "WHERE $1=$1 AND $2=$2 "
	}

	switch internal.ReadPermission(col) {
	case internal.PermGroup:
		return "WHERE account_id = $1 AND $2=$2 "
	case internal.PermOwner:
		return "WHERE account_id = $1 AND owner_id = $2 "
	default:
		//for read permission to everyone i.e. col-name_774_
		return "WHERE $1=$1 AND $2=$2 "
	}
}

func secureWrite(auth model.Auth, col string) string {
	if strings.HasPrefix(col, "pub_") || auth.Role == 100 {
		return "WHERE $1=$1 AND $2=$2 "
	}

	switch internal.WritePermission(col) {
	case internal.PermGroup:
		return "WHERE account_id = $1 AND $2=$2 "
	case internal.PermOwner:
		return "WHERE account_id = $1 AND owner_id = $2 "
	default:
		//for write permission to everyone i.e. col-name_776_
		// This should probably get more warning in the doc.
		// All logged-in users can update/delete data.
		// There's use cases for that, and it's certainly opt-in
		// but it's not recommended.
		return "WHERE $1=$1 AND $2=$2 "
	}
}

func setPaging(params model.ListParams) string {
	if len(params.SortBy) == 0 {
		params.SortBy = "created"
	}

	direction := "ASC"
	if params.SortDescending {
		direction = "DESC"
	}

	orderBy := fmt.Sprintf("ORDER BY %s %s", params.SortBy, direction)

	offset := (params.Page - 1) * params.Size
	return fmt.Sprintf("%s\nLIMIT %d OFFSET %d", orderBy, params.Size, offset)
}
