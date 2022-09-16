package staticbackend

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/staticbackendhq/core/model"
)

func TestFunctionsExecuteDBOperations(t *testing.T) {
	code := `
	function handle(body) {
		log(body);
		var o = {
			from: body.from,
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
			log("id:");
			log(getRes.content.id);
			log("end id");
			return;
		} else if (getRes.content.from != "val from unit test") {
			log("ERROR: asserting data from request body");
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
			log(qres);
			return;
		}

		if (upres.content.id != qres.content.results[0].id) {
			log("ERROR");
			log("expected updated doc's id to equal the query result");
			log("updated id: " + upres.content.id);
			log("query doc id: " + qres.content.results[0].id);
			return;
		}

		var getRes = fetch("https://run.mocky.io/v3/427873c5-4baa-4f68-b880-b6e3e45b3d4d");
		if (!getRes.ok) {
			log("ERROR: sending GET request");
			log(getRes.content);
			return;
		}

		var postRes = fetch("https://run.mocky.io/v3/427873c5-4baa-4f68-b880-b6e3e45b3d4d", {
			method: "POST",
			headers: {
				"Content-Type" : "application/json"
			}, 
			body: {
				"test": "test msg"
			}
		});
		if (!postRes.ok) {
			log("ERROR: sending POST request");
			log(postRes.content);
			return;
		}

		var putRes = fetch("https://run.mocky.io/v3/427873c5-4baa-4f68-b880-b6e3e45b3d4d", {
			method: "PUT",
			headers: {
				"Content-Type" : "application/json"
			}, 
			body: {
				"test": "test msg"
			}
		});
		if (!putRes.ok) {
			log("ERROR: sending PUT request");
			log(putRes.content);
			return;
		}
		var patchRes = fetch("https://run.mocky.io/v3/427873c5-4baa-4f68-b880-b6e3e45b3d4d", {
			method: "PATCH",
			headers: {
				"Content-Type" : "application/json"
			}, 
			body: {
				"test": "test msg"
			}
		});
		if (!patchRes.ok) {
			log("ERROR: sending PATCH request");
			log(patchRes.content);
			return;
		}
		var delRes = fetch("https://run.mocky.io/v3/427873c5-4baa-4f68-b880-b6e3e45b3d4d", {
			method: "DELETE",
			headers: {
				"Content-Type" : "application/json"
			}, 
			body: {
				"test": "test msg"
			}
		});
		if (!delRes.ok) {
			log("ERROR: sending DELETE request");
			log(delRes.content);
			return;
		}
	}`
	data := model.ExecData{
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

	val := url.Values{}
	val.Add("from", "val from unit test")

	execResp := dbReq(t, funexec.exec, "POST", "/fn/exec/unittest", val, false, true)
	if execResp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(execResp.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer execResp.Body.Close()

		t.Log(string(b))
		t.Errorf("expected status 200 got %s", execResp.Status)
	}

	infoResp := dbReq(t, funexec.info, "GET", "/fn/info/unittest", nil, true)

	var checkFn model.ExecData
	if err := parseBody(infoResp.Body, &checkFn); err != nil {
		t.Fatal(err)
	}
	defer infoResp.Body.Close()

	var errorLines []string
	foundError := false
	for _, h := range checkFn.History {
		for _, line := range h.Output {
			if strings.Index(line, "ERROR") > -1 {
				errorLines = h.Output
				foundError = true
				break
			}
		}

		if foundError {
			break
		}
	}

	if foundError {
		t.Errorf("found error in function exec log: %v", errorLines)
	}
}
