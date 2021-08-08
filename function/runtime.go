package function

import (
	"fmt"
	"staticbackend/db"
	"staticbackend/internal"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ExecutionEnvironment struct {
	Auth internal.Auth
	DB   *mongo.Database
	Base *db.Base
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

	result, err := vm.RunString(`
		log("works here");
		function handle() {
			var o = {
				desc: "yep", 
				done: false, 
				subobj: {
					yep: "working", 
					status: true
				}
			};
			var result = create("jsexec", o);
			log(result);
			if (!result.ok) {
				log("ERROR");
				log(result.content);
				return;
			}
			var result = get("jsexec", result.content.id)
			log("result.ok", result.ok);
			log("result.content", result.content)
		}`)

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
	vm.Set("get", func(call goja.FunctionCall) goja.Value {
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
