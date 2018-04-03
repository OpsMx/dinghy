package dinghyfile

import (
	"strings"
	"testing"

	"encoding/json"

	"github.com/armory-io/dinghy/pkg/cache"
	"github.com/armory-io/dinghy/pkg/git/dummy"
	"github.com/stretchr/testify/assert"
)

var fileService = dummy.FileService{
	"simpleTempl": `{
		"stages": [
			{{ module "wait.stage.module" "waitTime" 10 "refId" { "c": "d" } "requisiteStageRefIds" ["1", "2", "3"] }}
		]
	}`,
	"df": `{
		"stages": [
			{{ module "mod1" }},
			{{ module "mod2" }}
		]
	}`,
	"mod1": `{
		"foo": "bar",
		"type": "deploy"
	}`,
	"mod2": `{
		"type": "jenkins"
	}`,
	"wait.stage.module": `{
		"name": "Wait",
		"refId": {},
		"requisiteStageRefIds": [],
		"type": "wait",
		"waitTime": 12044
	}`,
}

func TestSimpleWaitStage(t *testing.T) {
	builder := &PipelineBuilder{
		Downloader: fileService,
		Depman:     cache.NewMemoryCache(),
	}

	buf := builder.Render("org", "repo", "simpleTempl", nil)

	const expected = `{
		"stages": [
			{
				"name": "Wait",
				"refId": { "c": "d" },
				"requisiteStageRefIds": ["1", "2", "3"],
				"type": "wait",
				"waitTime": 10
			}
		]
	}`

	// strip whitespace from both strings for assertion
	exp := strings.Join(strings.Fields(expected), "")
	actual := strings.Join(strings.Fields(buf.String()), "")
	assert.Equal(t, exp, actual)
}

func TestSpillover(t *testing.T) {
	builder := &PipelineBuilder{
		Downloader: fileService,
		Depman:     cache.NewMemoryCache(),
	}

	buf := builder.Render("org", "repo", "df", nil)

	const expected = `{
		"stages": [
			{"foo":"bar","type":"deploy"},
			{"type":"jenkins"}
		]
	}`

	// strip whitespace from both strings for assertion
	exp := strings.Join(strings.Fields(expected), "")
	actual := strings.Join(strings.Fields(buf.String()), "")
	assert.Equal(t, exp, actual)
}

var multilevelFileService = dummy.FileService{
	"dinghyfile": `{{ module "wait.stage.module" "foo" "baz" "waitTime" 100 }}`,

	"wait.stage.module": `{
		"foo": "{{ var "foo" "baz" }}",
		"a": "{{ var "nonexistent" "b" }}",
		"nested": {{ module "wait.dep.module" }}
	}`,

	"wait.dep.module": `{
		"waitTime": {{ var "waitTime" 1000 }}
	}`,
}

type testStruct struct {
	Foo    string `json:"foo"`
	A      string `json:"a"`
	Nested struct {
		WaitTime int `json:"waitTime"`
	} `json:"nested"`
}

func TestModuleVariableSubstitution(t *testing.T) {
	builder := &PipelineBuilder{
		Depman:     cache.NewMemoryCache(),
		Downloader: multilevelFileService,
	}

	ts := testStruct{}
	ret := builder.Render("org", "repo", "dinghyfile", nil)

	err := json.Unmarshal(ret.Bytes(), &ts)
	assert.Equal(t, nil, err)

	assert.Equal(t, "baz", ts.Foo)
	assert.Equal(t, "b", ts.A)
	assert.Equal(t, 100, ts.Nested.WaitTime)
}

/*
func TestPipelineID(t *testing.T) {
	id := pipelineIDFunc("armoryspinnaker", "fake-echo-test")
	assert.Equal(t, "f9c05bd0-5a50-4540-9e15-b44740abfb10", id)
}
*/
