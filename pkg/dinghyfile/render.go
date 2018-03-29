package dinghyfile

import (
	"bytes"
	"encoding/json"
	"errors"

	"text/template"

	"github.com/armory-io/dinghy/pkg/preprocessor"
	log "github.com/sirupsen/logrus"
)

func parseValue(val interface{}) interface{} {
	log.Info("newval: ", val)

	if jsonStr, ok := val.(string); ok {
		if jsonStr[0] == '{' {
			json.Unmarshal([]byte(jsonStr), &val)
		}
		if jsonStr[0] == '[' {
			json.Unmarshal([]byte(jsonStr), &val)
		}
	}

	return val
}

func replaceFields(obj map[string]interface{}, vars []interface{}, mod string) {
	if len(vars)%2 != 0 {
		log.Fatal(errors.New("invalid number of args to module: " + mod))
	}

	for i := 0; i < len(vars); i += 2 {
		key, ok := vars[i].(string)
		if !ok {
			log.Fatal(errors.New("dict keys must be strings in module: " + mod))
		}

		val, exists := obj[key]
		if exists {
			obj[key] = parseValue(vars[i+1])
			log.Info(" ** variable substitution in ", mod, " for key: ", key, ", value ", val, " --> ", obj[key])
		}
	}
}

type templateFunc func(mod string, vars ...interface{}) string

func moduleFunc(b *PipelineBuilder, org string, deps map[string]bool, v []interface{}) templateFunc {
	return func(mod string, vars ...interface{}) string {
		// Record the dependency.
		child := b.Downloader.EncodeURL(org, b.TemplateRepo, mod)
		if _, exists := deps[child]; !exists {
			deps[child] = true
		}

		// Render the module recursively.
		rendered := b.Render(b.TemplateOrg, b.TemplateRepo, mod, vars)

		// Decode rendered JSON into a map.
		var decoded map[string]interface{}
		err := json.Unmarshal(rendered.Bytes(), &decoded)
		if err != nil {
			log.Fatal("could not unmarshal module after rendering: ", mod, " err: ", err)
		}

		// Replace fields inside map.
		replaceFields(decoded, append(vars, v...), mod)

		// Encode back into JSON.
		byt, err := json.Marshal(decoded)
		if err != nil {
			log.Fatal("could not marshal variable substituted json for module: ", mod, err)
		}

		return string(byt)
	}
}

// Render renders the template
func (b *PipelineBuilder) Render(org, repo, path string, v []interface{}) *bytes.Buffer {
	deps := make(map[string]bool)
	funcMap := template.FuncMap{
		"module": moduleFunc(b, org, deps, v),
	}

	// Download the template being rendered.
	contents, err := b.Downloader.Download(org, repo, path)
	if err != nil {
		log.Fatalf("could not download %s/%s/%s", org, repo, path)
	}

	// Preprocess to stringify any json args in calls to modules.
	contents = preprocessor.Preprocess(contents)

	// Parse the downloaded template.
	tmpl, err := template.New("moduleTest").Funcs(funcMap).Parse(contents)
	if err != nil {
		log.Fatalf("template parsing: %s", err)
	}

	// Run the template to verify the output.
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, "")
	if err != nil {
		log.Fatalf("template execution: %s", err)
	}

	// Record the dependencies we ran into.
	depUrls := make([]string, 0)
	for dep := range deps {
		depUrls = append(depUrls, dep)
	}
	b.Depman.SetDeps(b.Downloader.EncodeURL(org, repo, path), depUrls)

	return buf
}
