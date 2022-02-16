package staticbackend

import (
	"io"
	"net/http"
	"testing"

	"github.com/staticbackendhq/core/internal"
)

func TestFunctionsExecuteDBOperations(t *testing.T) {
	code := `
	log("works here");
	function handle(body) {
		var o = {
			desc: "yep", 
			done: false, 
			subobj: {
				yep: "working", 
				status: true
			}
		};
		var result = create("jsexec", o);
		if (!result.ok) {
			log("ERROR: creating doc");
			log(result.content);
			return;
		}
		var getRes = getById("jsexec", result.content.id)
		if (!getRes.ok) {
			log("ERROR: getting doc by id");
			log(getRes.content);
			return;
		}

		var updata = getRes.content;
		updata.done = true;
		var upres = update("jsexec", updata.id, updata);
		if (!upres.ok) {
			log("ERROR: updating doc");
			log(upres.content);
			return;
		}

		var qres = query("jsexec", [["done", "==", true]]);
		if (!qres.ok) {
			log("ERROR: querying documents");
			log(qres.content);
			return;
		}

		if (qres.content.results.length != 1) {
			log("ERROR");
			log("expected results to have 1 doc, got: " + qres.content.results.length);
			return;
		}

		if (upres.content.id != qres.content.results[0].id) {
			log("ERROR");
			log("expected updated doc's id to equal the query result");
			log("updated id: " + upres.content.id);
			log("query doc id: " + qres.content.results[0].id);
			return;
		}
		
	}`
	data := internal.ExecData{
		FunctionName: "unittest",
		Code:         code,
		TriggerTopic: "web",
	}
	addResp := dbReq(t, funexec.add, "POST", "/", data, true)
	if addResp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(addResp.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer addResp.Body.Close()

		t.Log(string(b))
		t.Errorf("add: expected status 200 got %s", addResp.Status)
	}

	execResp := dbReq(t, funexec.exec, "POST", "/", data, true)
	if execResp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(execResp.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer execResp.Body.Close()

		t.Log(string(b))
		t.Errorf("expected status 200 got %s", execResp.Status)
	}
}
