package preprocessor

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPreprocess(t *testing.T) {
	input := `{ "a": {{ module "myMod" "key" {"my": "value"} "foo" [1] }} }`
	expected := `{ "a": {{ module "myMod" "key" "{\"my\": \"value\"}" "foo" "[1]" }} }`
	actual := Preprocess(input)
	t.Log(actual)
	assert.Equal(t, expected, actual)
}
