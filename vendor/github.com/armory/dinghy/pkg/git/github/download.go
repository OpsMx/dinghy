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
	"github.com/armory/dinghy/pkg/cache/local"
	"github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

// FileService is for working with repositories
type FileService struct {
	cache  local.Cache
	GitHub GitHubClient
	Logger logrus.FieldLogger
}

// Download a file from github
// note that "path" is the full path relative to the repo root
// eg: src/foo/bar/filename
func (f *FileService) Download(org, repo, path, branch string) (string, error) {
	// The endpoint used by the Github lib (https://raw.githubusercontent.com/) does not
	// accept branch names such as refs/heads/master, but only the name of the branch.
	// Need to strip that if it exists. Can't use split here either, because '/' is allowed
	// in branch names
	branch = strings.Replace(branch, "refs/heads/", "", 1)

	url := f.EncodeURL(org, repo, path, branch)
	body := f.cache.Get(url)
	if body != "" {
		return body, nil
	}

	contents, err := f.GitHub.DownloadContents(org, repo, path, branch)
	if err != nil {
		f.Logger.Error(err)
		return "", err
	}

	f.cache.Add(url, contents)

	return contents, nil
}

// EncodeURL returns the git url for a given org, repo, path and branch
func (f *FileService) EncodeURL(org, repo, path, branch string) string {
	// this is only used for caching purposes
	return fmt.Sprintf(`%s/repos/%s/%s/contents/%s?ref=%s`, f.GitHub.GetEndpoint(), org, repo, path, branch)
}

// DecodeURL takes a url and returns the org, repo, path and branch
func (f *FileService) DecodeURL(url string) (org, repo, path, branch string) {
	targetExpression := fmt.Sprintf(`%s/repos/(.+)/(.+)/contents/(.+)\?ref=(.+)`, f.GitHub.GetEndpoint())
	r, _ := regexp.Compile(targetExpression)
	match := r.FindStringSubmatch(url)
	org = match[1]
	repo = match[2]
	path = match[3]
	branch = match[4]
	return
}
