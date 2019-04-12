/*
* Copyright 2019 Armory, Inc.

* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at

*    http://www.apache.org/licenses/LICENSE-2.0

* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/

package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/armory-io/dinghy/pkg/git"
)

func TestSetCommitStatusSuccessfully(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "")
	}))
	defer ts.Close()
	// TODO: Do not use global variable. This will lead to side-effects.
	p := Push{
		GitHub: GitHub{
			Endpoint: ts.URL,
		},
		Repository: Repository{Organization: "armory-io", Name: "dinghy"},
		Commits: []Commit{
			{
				ID: "ABC",
			},
		},
	}

	// This shouldn't throw exceptions/panics
	p.SetCommitStatus(git.StatusPending)
}

func TestSetCommitStatusFails(t *testing.T) {
	// TODO: Do not use global variable. This will lead to side-effects.
	p := Push{
		GitHub: GitHub{
			Endpoint: "invalid-url",
		},
		Repository: Repository{Organization: "armory-io", Name: "dinghy"},
		Commits: []Commit{
			{
				ID: "ABC",
			},
		},
	}

	// TODO: this doesn't actually test anything
	// This shouldn't throw exceptions/panics
	p.SetCommitStatus(git.StatusPending)
}
