package function

import (
	"encoding/json"
	"fmt"
	"staticbackend/db"
	"staticbackend/internal"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ExecutionEnvironment struct {
	Auth     internal.Auth
	DB       *mongo.Database
	Base     *db.Base
	Volatile internal.PubSuber
	Data     ExecData
}

type Result struct {
	OK      bool        `json:"ok"`
	Content interface{} `json:"content"`
}

func (env *ExecutionEnvironment) Execute() error {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	env.addHelpers(vm)
	env.addDatabaseFunctions(vm)
	env.addVolatileFunctions(vm)

	result, err := vm.RunString(env.Data.Code)
	if err != nil {
		return err
	}

	handler, ok := goja.AssertFunction(vm.Get("handle"))
	if !ok {
		return fmt.Errorf(`unable to find function "handle": %v`, err)
	}

	resp, err := handler(goja.Undefined())
	if err != nil {
		return fmt.Errorf("error executing your function: %v", err)
	}

	fmt.Println("resp", resp)
	fmt.Println("result", result)
	return nil
}

func (env *ExecutionEnvironment) addHelpers(vm *goja.Runtime) {
	vm.Set("log", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return goja.Undefined()
		}

		var params []interface{}
		for _, v := range call.Arguments {
			params = append(params, v.Export())
		}
		fmt.Println(params...)
		return goja.Undefined()
	})
}

func (env *ExecutionEnvironment) addDatabaseFunctions(vm *goja.Runtime) {
	vm.Set("create", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 2 {
			return vm.ToValue(Result{Content: "argument missmatch: you need 2 arguments for create(col, doc"})
		}
		var col string
		if err := vm.ExportTo(call.Argument(0), &col); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		}
		doc := make(map[string]interface{})
		if err := vm.ExportTo(call.Argument(1), &doc); err != nil {
			return vm.ToValue(Result{Content: "the second argument should be an object"})
		}

		doc, err := env.Base.Add(env.Auth, env.DB, col, doc)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error calling create(): %s", err.Error())})
		}

		if err := env.clean(doc); err != nil {
			return vm.ToValue(Result{Content: err.Error()})
		}
		return vm.ToValue(Result{OK: true, Content: doc})
	})
	vm.Set("list", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return vm.ToValue(Result{Content: "argument missmatch: your need at least 1 argument for list(col, [params])"})
		}

		var col string
		if err := vm.ExportTo(call.Argument(0), &col); err != nil {
			return vm.ToValue(Result{Content: "the first agrument should be a string"})
		}

		var params db.ListParams
		if len(call.Arguments) >= 2 {
			v := call.Argument(1)
			if !goja.IsNull(v) && !goja.IsUndefined(v) {
				if err := vm.ExportTo(v, &params); err != nil {
					return vm.ToValue(Result{Content: "the second argument should be an object"})
				}
			}
		}

		result, err := env.Base.List(env.Auth, env.DB, col, params)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error executing list: %v", err)})
		}

		for _, v := range result.Results {
			if err := env.clean(v); err != nil {
				return vm.ToValue(Result{Content: fmt.Sprintf("error cleaning doc: %v", err)})
			}
		}

		return vm.ToValue(Result{OK: true, Content: result})
	})
	vm.Set("getById", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 2 {
			return vm.ToValue(Result{Content: "argument missmatch: you need 2 arguments for get(repo, id)"})
		}
		var col, id string
		if err := vm.ExportTo(call.Argument(0), &col); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		}
		if err := vm.ExportTo(call.Argument(1), &id); err != nil {
			return vm.ToValue(Result{Content: "the second argument should be a string"})
		}

		doc, err := env.Base.GetByID(env.Auth, env.DB, col, id)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error calling get(): %s", err.Error())})
		}

		if err := env.clean(doc); err != nil {
			return vm.ToValue(Result{Content: err.Error()})
		}

		return vm.ToValue(Result{OK: true, Content: doc})
	})
	vm.Set("query", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return vm.ToValue(Result{Content: "argument missmatch: you need at least 2 arguments for query(col, filter, [params])"})
		}
		var col string
		if err := vm.ExportTo(call.Argument(0), &col); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		}
		var clauses [][]interface{}
		if err := vm.ExportTo(call.Argument(1), &clauses); err != nil {
			return vm.ToValue(Result{Content: "the second argument should be a query filter: [['field', '==', 'value'], ...]"})
		}

		filter, err := db.ParseQuery(clauses)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error parsing query filter: %v", err)})
		}

		var params db.ListParams
		if len(call.Arguments) >= 3 {
			v := call.Argument(2)
			if !goja.IsNull(v) && !goja.IsUndefined(v) {
				if err := vm.ExportTo(v, &params); err != nil {
					return vm.ToValue(Result{Content: "the second argument should be an object"})
				}
			}
		}

		result, err := env.Base.Query(env.Auth, env.DB, col, filter, params)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error executing query: %v", err)})
		}

		for _, v := range result.Results {
			if err := env.clean(v); err != nil {
				return vm.ToValue(Result{Content: fmt.Sprintf("error cleaning doc: %v", err)})
			}
		}

		return vm.ToValue(Result{OK: true, Content: result})
	})
	vm.Set("update", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 3 {
			return vm.ToValue(Result{Content: "argument missmatch: you need 3 arguments for update(col, id, doc)"})
		}

		var col, id string
		if err := vm.ExportTo(call.Argument(0), &col); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		}
		if err := vm.ExportTo(call.Argument(1), &id); err != nil {
			return vm.ToValue(Result{Content: "the second argument should be a string"})
		}

		doc := make(map[string]interface{})
		if err := vm.ExportTo(call.Argument(2), &doc); err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error executing update: %v", err)})
		}

		updated, err := env.Base.Update(env.Auth, env.DB, col, id, doc)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error executing update: %v", err)})
		}

		if err := env.clean(updated); err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error cleaning doc: %v", err)})
		}

		return vm.ToValue(Result{OK: true, Content: updated})
	})
	vm.Set("del", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 2 {
			return vm.ToValue(Result{Content: "argument missmatch: you need 3 arguments for del(col, id)"})
		}

		var col, id string
		if err := vm.ExportTo(call.Argument(0), &col); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		}
		if err := vm.ExportTo(call.Argument(1), &id); err != nil {
			return vm.ToValue(Result{Content: "the second argument should be a string"})
		}

		deleted, err := env.Base.Delete(env.Auth, env.DB, col, id)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error executing del: %v", err)})
		}

		return vm.ToValue(Result{OK: true, Content: deleted})
	})
}

func (*ExecutionEnvironment) clean(doc map[string]interface{}) error {
	if id, ok := doc["id"]; ok {
		oid, ok := id.(primitive.ObjectID)
		if !ok {
			return fmt.Errorf("unable to cast document id")
		}
		doc["id"] = oid.Hex()
	}

	if id, ok := doc[internal.FieldAccountID]; ok {
		oid, ok := id.(primitive.ObjectID)
		if !ok {
			return fmt.Errorf("unable to cast document accountId")
		}
		doc[internal.FieldAccountID] = oid.Hex()
	}

	return nil
}

func (env *ExecutionEnvironment) addVolatileFunctions(vm *goja.Runtime) {
	vm.Set("send", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 3 {
			return vm.ToValue(Result{Content: "argument missmatch: you need 3 arguments for send(type, data, channel)"})
		}

		var typ, channel string
		if err := vm.ExportTo(call.Argument(0), &typ); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		} else if err := vm.ExportTo(call.Argument(2), &channel); err != nil {
			return vm.ToValue(Result{Content: "the third argument should be a string"})
		}

		b, err := json.Marshal(call.Argument(1).Export())
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error converting your data: %v", err)})
		}

		msg := internal.Command{
			SID:     env.Data.ID.Hex(),
			Type:    typ,
			Data:    string(b),
			Channel: channel,
			Token:   env.Auth.ReconstructToken(),
		}

		if err := env.Volatile.Publish(msg); err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error publishing your message: %v", err)})
		}

		return vm.ToValue(Result{OK: true})
	})
}
