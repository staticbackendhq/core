package function

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
	"github.com/staticbackendhq/core/search"

	"github.com/dop251/goja"
)

type ExecutionEnvironment struct {
	Auth      model.Auth
	BaseName  string
	DataStore database.Persister
	Volatile  cache.Volatilizer
	Email     email.Mailer
	Search    *search.Search
	Data      model.ExecData

	CurrentRun model.ExecHistory
	Log        *logger.Logger
}

type Result struct {
	OK      bool        `json:"ok"`
	Content interface{} `json:"content"`
}

func (env *ExecutionEnvironment) Execute(data interface{}) error {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	if err := env.addHelpers(vm); err != nil {
		return err
	}
	if err := env.addDatabaseFunctions(vm); err != nil {
		return err
	}
	if err := env.addVolatileFunctions(vm); err != nil {
		return err
	}
	if err := env.addSearch(vm); err != nil {
		return err
	}
	if err := env.addSendMail(vm); err != nil {
		return err
	}

	if _, err := vm.RunString(env.Data.Code); err != nil {
		return err
	}

	handler, ok := goja.AssertFunction(vm.Get("handle"))
	if !ok {
		return errors.New(`unable to find function "handle"`)
	}

	args, err := env.prepareArguments(vm, data)
	if err != nil {
		return fmt.Errorf("error preparing argument: %v", err)
	}

	env.CurrentRun = model.ExecHistory{
		Version: env.Data.Version,
		Started: time.Now(),
		Output:  make([]string, 0),
	}

	env.CurrentRun.Output = append(env.CurrentRun.Output, "Function started")

	_, err = handler(goja.Undefined(), args...)
	go env.complete(err)
	if err != nil {
		return fmt.Errorf("error executing your function: %v", err)
	}

	return nil
}

func (env *ExecutionEnvironment) prepareArguments(vm *goja.Runtime, data interface{}) ([]goja.Value, error) {
	var args []goja.Value

	// for "web" trigger we prepare the body, query string and headers
	r, ok := data.(*http.Request)
	if ok {
		defer r.Body.Close()

		// let's ready the HTTP body
		if strings.EqualFold(r.Header.Get("Content-Type"), "application/json") {
			var v interface{}
			if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
				return nil, err
			}

			args = append(args, vm.ToValue(v))
		} else if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			if err := r.ParseForm(); err != nil {
				return nil, err
			}

			val := make(map[string]interface{})
			for k, v := range r.Form {
				val[k] = strings.Join(v, ", ")
			}
			args = append(args, vm.ToValue(val))
		}

		args = append(args, vm.ToValue(r.URL.Query()))
		args = append(args, vm.ToValue(r.Header))

		return args, nil
	}

	msg, ok := data.(model.Command)
	if ok {
		var v any
		if err := json.Unmarshal([]byte(msg.Data), &v); err != nil {
			return args, err
		}

		args = append(args, vm.ToValue(msg.Channel))
		args = append(args, vm.ToValue(msg.Type))
		args = append(args, vm.ToValue(v))

		return args, nil
	}

	// system or custom event/topic, we send only the 1st argument (body)
	args = append(args, vm.ToValue(data))
	return args, nil
}

func (env *ExecutionEnvironment) addHelpers(vm *goja.Runtime) error {
	err := vm.Set("log", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return goja.Undefined()
		}

		var params []interface{}
		for _, v := range call.Arguments {
			params = append(params, v.Export())
		}
		env.CurrentRun.Output = append(env.CurrentRun.Output, fmt.Sprint(params...))
		return goja.Undefined()
	})
	if err != nil {
		return err
	}
	err = vm.Set("fetch", func(call goja.FunctionCall) goja.Value {
		url := ""
		fetchOptions := NewJSFetcthOptionArg()
		if len(call.Arguments) == 0 {
			return goja.Undefined()
		} else if len(call.Arguments) == 1 {
			url = call.Argument(0).Export().(string)
		} else {
			url = call.Argument(0).Export().(string)
			if err := vm.ExportTo(call.Argument(1), &fetchOptions); err != nil {
				return vm.ToValue(Result{Content: "the second argument should be an object"})
			}
		}
		if len(url) == 0 {
			return vm.ToValue(Result{Content: "the url should not be blank"})
		}

		responseChan := make(chan interface{})
		go func() {
			client := http.Client{Timeout: time.Duration(30) * time.Second}
			var request *http.Request
			var err error
			bodyReader := strings.NewReader(fetchOptions.Body)
			switch fetchOptions.Method {
			case "GET":
				request, err = http.NewRequest(http.MethodGet, url, nil)
			case "POST":
				request, err = http.NewRequest(http.MethodPost, url, bodyReader)
			case "PUT":
				request, err = http.NewRequest(http.MethodPut, url, bodyReader)
			case "DELETE":
				request, err = http.NewRequest(http.MethodDelete, url, bodyReader)
			case "PATCH":
				request, err = http.NewRequest(http.MethodPatch, url, bodyReader)
			}
			if err != nil {
				responseChan <- err
			}
			for headerKey, headerValue := range fetchOptions.Headers {
				if len(headerKey) > 0 && len(headerValue) > 0 {
					request.Header.Set(headerKey, headerValue)
				}
			}
			res, err := client.Do(request)
			if err != nil {
				responseChan <- err
			}
			responseChan <- res
		}()

		output := <-responseChan

		if err, ok := output.(error); ok {
			return vm.ToValue(Result{OK: false, Content: fmt.Sprintf("error calling fetch(): %s", err.Error())})
		} else if response, ok := output.(*http.Response); ok {
			bodyBytes, err := io.ReadAll(response.Body)
			if err != nil {
				return vm.ToValue(Result{OK: false, Content: fmt.Sprintf("error calling fetch(): %s", err.Error())})
			}
			response.Body.Close()

			return vm.ToValue(Result{OK: true, Content: HTTPResponse{Status: response.StatusCode, Body: string(bodyBytes)}})
		}
		return goja.Undefined()
	})
	if err != nil {
		return err
	}
	return nil
}

func (env *ExecutionEnvironment) addDatabaseFunctions(vm *goja.Runtime) error {
	err := vm.Set("create", func(call goja.FunctionCall) goja.Value {
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

		doc, err := env.DataStore.CreateDocument(env.Auth, env.BaseName, col, doc)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error calling create(): %s", err.Error())})
		}

		if err := env.clean(doc); err != nil {
			return vm.ToValue(Result{Content: err.Error()})
		}
		return vm.ToValue(Result{OK: true, Content: doc})
	})
	if err != nil {
		return err
	}

	err = vm.Set("list", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return vm.ToValue(Result{Content: "argument missmatch: your need at least 1 argument for list(col, [params])"})
		}

		var col string
		if err := vm.ExportTo(call.Argument(0), &col); err != nil {
			return vm.ToValue(Result{Content: "the first agrument should be a string"})
		}

		var params model.ListParams
		if len(call.Arguments) >= 2 {
			v := call.Argument(1)
			if !goja.IsNull(v) && !goja.IsUndefined(v) {
				if err := vm.ExportTo(v, &params); err != nil {
					return vm.ToValue(Result{Content: "the second argument should be an object"})
				}
			}
		}

		result, err := env.DataStore.ListDocuments(env.Auth, env.BaseName, col, params)
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
	if err != nil {
		return err
	}

	err = vm.Set("getById", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 2 {
			return vm.ToValue(Result{Content: "argument missmatch: you need 2 arguments for get(col, id)"})
		}
		var col, id string
		if err := vm.ExportTo(call.Argument(0), &col); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		}
		if err := vm.ExportTo(call.Argument(1), &id); err != nil {
			return vm.ToValue(Result{Content: "the second argument should be a string"})
		}

		doc, err := env.DataStore.GetDocumentByID(env.Auth, env.BaseName, col, id)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error calling get(): %s", err.Error())})
		}

		if err := env.clean(doc); err != nil {
			return vm.ToValue(Result{Content: err.Error()})
		}

		return vm.ToValue(Result{OK: true, Content: doc})
	})
	if err != nil {
		return err
	}

	err = vm.Set("query", func(call goja.FunctionCall) goja.Value {
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

		filter, err := env.DataStore.ParseQuery(clauses)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error parsing query filter: %v", err)})
		}

		var params model.ListParams
		if len(call.Arguments) >= 3 {
			v := call.Argument(2)
			if !goja.IsNull(v) && !goja.IsUndefined(v) {
				if err := vm.ExportTo(v, &params); err != nil {
					return vm.ToValue(Result{Content: "the second argument should be an object"})
				}
			}
		}

		// apply default page and limit
		if params.Size == 0 {
			params.Size = 25
			params.Page = 1
		}

		result, err := env.DataStore.QueryDocuments(env.Auth, env.BaseName, col, filter, params)
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
	if err != nil {
		return err
	}

	err = vm.Set("update", func(call goja.FunctionCall) goja.Value {
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

		updated, err := env.DataStore.UpdateDocument(env.Auth, env.BaseName, col, id, doc)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error executing update: %v", err)})
		}

		if err := env.clean(updated); err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error cleaning doc: %v", err)})
		}

		return vm.ToValue(Result{OK: true, Content: updated})
	})
	if err != nil {
		return err
	}

	err = vm.Set("del", func(call goja.FunctionCall) goja.Value {
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

		deleted, err := env.DataStore.DeleteDocument(env.Auth, env.BaseName, col, id)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error executing del: %v", err)})
		}

		return vm.ToValue(Result{OK: true, Content: deleted})
	})
	if err != nil {
		return err
	}
	return nil
}

func (env *ExecutionEnvironment) addSendMail(vm *goja.Runtime) error {
	smf := func(call goja.FunctionCall) goja.Value {

		if len(call.Arguments) != 1 {
			return vm.ToValue(Result{Content: "argument missmatch: you need only one arguments(object) for sendMail"})
		}

		sma := JSSendMailArg{}

		if err := vm.ExportTo(call.Argument(0), &sma); err != nil {
			return vm.ToValue(Result{Content: "argument should be an object"})
		}

		data := email.SendMailData{
			FromName: "",
			From:     sma.From,
			To:       sma.To,
			ToName:   "",
			Subject:  sma.Subject,
			HTMLBody: sma.HTMLBody,
			TextBody: sma.TextBody,
			ReplyTo:  "",
			Body:     "",
		}

		err := env.Email.Send(data)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("send mail error: %v", err)})
		}
		return vm.ToValue(Result{OK: true})
	}

	err := vm.Set("sendMail", smf)
	if err != nil {
		return err
	}
	return nil
}

func (*ExecutionEnvironment) clean(doc map[string]interface{}) error {
	//TODONOW: not sure what was the exact used for this clean-up
	/*
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
		}*/
	return nil
}

func (env *ExecutionEnvironment) addVolatileFunctions(vm *goja.Runtime) error {
	err := vm.Set("publish", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 3 {
			return vm.ToValue(Result{Content: "argument missmatch: you need 3 arguments for send(channel, type, data)"})
		}

		var typ, channel string
		if err := vm.ExportTo(call.Argument(0), &channel); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		} else if err := vm.ExportTo(call.Argument(1), &typ); err != nil {
			return vm.ToValue(Result{Content: "the third argument should be a string"})
		}

		b, err := json.Marshal(call.Argument(2).Export())
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error converting your data: %v", err)})
		}

		msg := model.Command{
			SID:     env.Data.ID,
			Type:    typ,
			Data:    string(b),
			Channel: channel,
			Token:   env.Auth.ReconstructToken(),
			Auth:    env.Auth,
			Base:    env.BaseName,
		}

		if err := env.Volatile.Publish(msg); err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error publishing your message: %v", err)})
		}

		return vm.ToValue(Result{OK: true})
	})
	if err != nil {
		return err
	}

	err = vm.Set("cacheGet", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 1 {
			return vm.ToValue(Result{Content: "argument missmatch: you need 1 argument for cacheGet(key)"})
		}

		var key string
		if err := vm.ExportTo(call.Argument(0), &key); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		}

		val, _ := env.Volatile.Get(key)

		return vm.ToValue(Result{OK: true, Content: val})
	})
	if err != nil {
		return err
	}

	err = vm.Set("cacheSet", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 2 {
			return vm.ToValue(Result{Content: "argument missmatch: you need 2 arguments for cacheSet(key, value)"})
		}

		var key, value string
		if err := vm.ExportTo(call.Argument(0), &key); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		} else if err := vm.ExportTo(call.Argument(1), &value); err != nil {
			return vm.ToValue(Result{Content: "the 2nd argument should be a string"})
		}

		if err := env.Volatile.Set(key, value); err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error while setting cache value: %v", err)})
		}

		return vm.ToValue(Result{OK: true})
	})
	if err != nil {
		return err
	}

	err = vm.Set("inc", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 2 {
			return vm.ToValue(Result{Content: "argument missmatch: you need 2 arguments for inc(key, n)"})
		}

		var key string
		var n int64
		if err := vm.ExportTo(call.Argument(0), &key); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		} else if err := vm.ExportTo(call.Argument(1), &n); err != nil {
			return vm.ToValue(Result{Content: "the 2nd argument should be a number"})
		}

		total, err := env.Volatile.Inc(key, n)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error while incrementing cache value: %v", err)})
		}

		return vm.ToValue(Result{OK: true, Content: total})
	})
	if err != nil {
		return err
	}

	err = vm.Set("dec", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 2 {
			return vm.ToValue(Result{Content: "argument missmatch: you need 2 arguments for dec(key, n)"})
		}

		var key string
		var n int64
		if err := vm.ExportTo(call.Argument(0), &key); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		} else if err := vm.ExportTo(call.Argument(1), &n); err != nil {
			return vm.ToValue(Result{Content: "the 2nd argument should be a number"})
		}

		total, err := env.Volatile.Dec(key, n)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error while decrementing cache value: %v", err)})
		}

		return vm.ToValue(Result{OK: true, Content: total})
	})
	if err != nil {
		return err
	}
	return nil
}

func (env *ExecutionEnvironment) addSearch(vm *goja.Runtime) error {
	err := vm.Set("search", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 2 {
			return vm.ToValue(Result{Content: "argument missmatch: you need 2 arguments for search(col, keywords)"})
		}

		var col, keywords string
		if err := vm.ExportTo(call.Argument(0), &col); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		} else if err := vm.ExportTo(call.Argument(1), &keywords); err != nil {
			return vm.ToValue(Result{Content: "the second argument should be a string"})
		}

		results, err := env.Search.Search(env.BaseName, col, keywords)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error while executing search(): %v", err)})
		}

		docs, err := env.DataStore.GetDocumentsByIDs(env.Auth, env.BaseName, col, results.IDs)
		if err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error getting document by ids from search result: %v", err)})
		}

		for _, doc := range docs {
			if err := env.clean(doc); err != nil {
				return vm.ToValue(Result{Content: fmt.Sprintf("error cleaning doc: %v", err)})
			}
		}

		return vm.ToValue(Result{OK: true, Content: docs})
	})
	if err != nil {
		return err
	}

	err = vm.Set("indexDocument", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 3 {
			return vm.ToValue(Result{Content: "argument missmatch: you need 3 arguments for indexDocument(col, id, text)"})
		}

		var col, id, text string
		if err := vm.ExportTo(call.Argument(0), &col); err != nil {
			return vm.ToValue(Result{Content: "the first argument should be a string"})
		} else if err := vm.ExportTo(call.Argument(1), &id); err != nil {
			return vm.ToValue(Result{Content: "the second argument should be a string"})
		} else if err := vm.ExportTo(call.Argument(1), &text); err != nil {
			return vm.ToValue(Result{Content: "the third argument should be a string"})
		}

		if err := env.Search.Index(env.BaseName, col, id, text); err != nil {
			return vm.ToValue(Result{Content: fmt.Sprintf("error while trying to index the document: %v", err)})
		}

		return vm.ToValue(Result{OK: true})
	})
	if err != nil {
		return err
	}
	return nil
}

func (env *ExecutionEnvironment) complete(err error) {
	env.CurrentRun.Completed = time.Now()
	env.CurrentRun.Success = err == nil

	env.CurrentRun.Output = append(env.CurrentRun.Output, "Function completed")

	// add the error in the last output entry
	if err != nil {
		env.CurrentRun.Output = append(env.CurrentRun.Output, err.Error())
	}

	//TODO: this needs to be regrouped and ran un batch
	if err := env.DataStore.RanFunction(env.BaseName, env.Data.ID, env.CurrentRun); err != nil {
		env.Log.Error().Err(err).Msg("error logging function complete")
	}
}
