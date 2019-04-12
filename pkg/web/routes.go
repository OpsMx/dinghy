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

package web

import (
	"bytes"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/armory-io/dinghy/pkg/events"
	"github.com/armory-io/dinghy/pkg/git/bbcloud"
	"github.com/armory/plank"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/armory-io/dinghy/pkg/cache"
	"github.com/armory-io/dinghy/pkg/dinghyfile"
	"github.com/armory-io/dinghy/pkg/git"
	"github.com/armory-io/dinghy/pkg/git/dummy"
	"github.com/armory-io/dinghy/pkg/git/github"
	"github.com/armory-io/dinghy/pkg/git/stash"
	"github.com/armory-io/dinghy/pkg/settings"
	"github.com/armory-io/dinghy/pkg/util"
)

// Push represents a push notification from a git service.
type Push interface {
	ContainsFile(file string) bool
	Files() []string
	Repo() string
	Org() string
	IsMaster() bool
	SetCommitStatus(s git.Status)
}

type WebAPI struct {
	Config      *settings.Settings
	Client      *plank.Client
	Cache       dinghyfile.DependencyManager
	EventClient *events.Client
}

func NewWebAPI(s *settings.Settings, r dinghyfile.DependencyManager, c *plank.Client, e *events.Client) *WebAPI {
	return &WebAPI{
		Config:      s,
		Client:      c,
		Cache:       r,
		EventClient: e,
	}
}

// Router defines the routes for the application.
func (wa *WebAPI) Router() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", wa.healthcheck)
	r.HandleFunc("/health", wa.healthcheck)
	r.HandleFunc("/healthcheck", wa.healthcheck)
	r.HandleFunc("/v1/webhooks/github", wa.githubWebhookHandler).Methods("POST")
	r.HandleFunc("/v1/webhooks/stash", wa.stashWebhookHandler).Methods("POST")
	r.HandleFunc("/v1/webhooks/bitbucket", wa.bitbucketServerWebhookHandler).Methods("POST")
	r.HandleFunc("/v1/webhooks/bitbucket-cloud", wa.bitbucketCloudWebhookHandler).Methods("POST")
	r.HandleFunc("/v1/updatePipeline", wa.manualUpdateHandler).Methods("POST")
	r.Use(RequestLoggingMiddleware)
	return r
}

// ==============
// route handlers
// ==============

func (wa *WebAPI) healthcheck(w http.ResponseWriter, r *http.Request) {
	log.Debug(r.RemoteAddr, " Requested ", r.RequestURI)
	w.Write([]byte(`{"status":"ok"}`))
}

func (wa *WebAPI) manualUpdateHandler(w http.ResponseWriter, r *http.Request) {
	var fileService = dummy.FileService{}

	builder := &dinghyfile.PipelineBuilder{
		Depman:               cache.NewMemoryCache(),
		Downloader:           fileService,
		Client:               wa.Client,
		DeleteStalePipelines: false,
		AutolockPipelines:    wa.Config.AutoLockPipelines,
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	fileService["dinghyfile"] = buf.String()
	log.Info("Received payload: ", fileService["dinghyfile"])

	if err := builder.ProcessDinghyfile("", "", "dinghyfile"); err != nil {
		util.WriteHTTPError(w, http.StatusInternalServerError, err)
	}
}

func (wa *WebAPI) githubWebhookHandler(w http.ResponseWriter, r *http.Request) {
	p := github.Push{}
	// TODO: stop using readbody, then we can return proper 422 status codes
	if err := readBody(r, &p); err != nil {
		util.WriteHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	if p.Ref == "" {
		// Unmarshal failed, might be a non-Push notification. Log event and return
		log.Debug("Possibly a non-Push notification received")
		return
	}

	// TODO: we're assigning config in two places here, we should refactor build pipelines so that
	// it takes file contents as a parameter and then we can create the github client, do the push
	// and download from here
	gh := github.GitHub{Endpoint: wa.Config.GithubEndpoint, Token: wa.Config.GitHubToken}
	p.GitHub = gh
	p.DeckBaseURL = wa.Config.Deck.BaseURL
	fileService := github.FileService{GitHub: &gh}

	wa.buildPipelines(&p, &fileService, w)
}

func (wa *WebAPI) stashWebhookHandler(w http.ResponseWriter, r *http.Request) {
	payload := stash.WebhookPayload{}
	// TODO: stop using readbody, then we can return proper status codes here
	if err := readBody(r, &payload); err != nil {
		util.WriteHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	payload.IsOldStash = true
	stashConfig := stash.StashConfig{
		Endpoint: wa.Config.StashEndpoint,
		Username: wa.Config.StashUsername,
		Token:    wa.Config.StashToken,
	}
	p, err := stash.NewPush(payload, stashConfig)
	if err != nil {
		util.WriteHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	// TODO: WebAPI already has the fields that are being assigned here and it's
	// the receiver on the buildPipelines. We don't need to reassign the values to
	// fileService here.
	fileService := stash.FileService{
		StashToken:    wa.Config.StashToken,
		StashUsername: wa.Config.StashUsername,
		StashEndpoint: wa.Config.StashEndpoint,
	}
	wa.buildPipelines(p, &fileService, w)
}

func (wa *WebAPI) bitbucketServerWebhookHandler(w http.ResponseWriter, r *http.Request) {
	payload := stash.WebhookPayload{}
	if err := readBody(r, &payload); err != nil {
		util.WriteHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	if payload.EventKey != "" && payload.EventKey != "repo:refs_changed" {
		w.WriteHeader(200)
		return
	}

	payload.IsOldStash = false
	stashConfig := stash.StashConfig{
		Endpoint: wa.Config.StashEndpoint,
		Username: wa.Config.StashUsername,
		Token:    wa.Config.StashToken,
	}
	p, err := stash.NewPush(payload, stashConfig)
	if err != nil {
		util.WriteHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	// TODO: WebAPI already has the fields that are being assigned here and it's
	// the receiver on the buildPipelines. We don't need to reassign the values to
	// fileService here.
	fileService := stash.FileService{
		StashToken:    wa.Config.StashToken,
		StashUsername: wa.Config.StashUsername,
		StashEndpoint: wa.Config.StashEndpoint,
	}
	wa.buildPipelines(p, &fileService, w)
}

func (wa *WebAPI) bitbucketCloudWebhookHandler(w http.ResponseWriter, r *http.Request) {
	payload := bbcloud.WebhookPayload{}
	if err := readBody(r, &payload); err != nil {
		util.WriteHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	bbcloudConfig := bbcloud.Config{
		Endpoint: wa.Config.StashEndpoint,
		Username: wa.Config.StashUsername,
		Token:    wa.Config.StashToken,
	}
	p, err := bbcloud.NewPush(payload, bbcloudConfig)
	if err != nil {
		util.WriteHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	// TODO: WebAPI already has the fields that are being assigned here and it's
	// the receiver on the buildPipelines. We don't need to reassign the values to
	// fileService here.
	fileService := bbcloud.FileService{
		BbcloudEndpoint: wa.Config.StashEndpoint,
		BbcloudUsername: wa.Config.StashUsername,
		BbcloudToken:    wa.Config.StashToken,
	}
	wa.buildPipelines(p, &fileService, w)
}

// =========
// utilities
// =========

// ProcessPush processes a push using a pipeline builder
func (wa *WebAPI) ProcessPush(p Push, b *dinghyfile.PipelineBuilder) error {
	// Ensure dinghyfile was changed.
	if !p.ContainsFile(wa.Config.DinghyFilename) {
		return nil
	}

	// Ensure we're on the master branch.
	if !p.IsMaster() {
		log.Info("Skipping Spinnaker pipeline update because this is not master")
		return nil
	}

	log.Info("Dinghyfile found in commit for repo " + p.Repo())

	// Set commit status to the pending yellow dot.
	p.SetCommitStatus(git.StatusPending)

	for _, filePath := range p.Files() {
		components := strings.Split(filePath, "/")
		if components[len(components)-1] == wa.Config.DinghyFilename {
			// Process the dinghyfile.
			err := b.ProcessDinghyfile(p.Org(), p.Repo(), filePath)
			// Set commit status based on result of processing.
			if err != nil {
				if err == dinghyfile.ErrMalformedJSON {
					p.SetCommitStatus(git.StatusFailure)
				} else {
					p.SetCommitStatus(git.StatusError)
				}
				return err
			}
			p.SetCommitStatus(git.StatusSuccess)
		}
	}
	return nil
}

// TODO: this func should return an error and allow the handlers to return the http response. Additionally,
// it probably doesn't belong in this file once refactored.
func (wa *WebAPI) buildPipelines(p Push, f dinghyfile.Downloader, w http.ResponseWriter) {
	// Construct a pipeline builder using provided downloader
	builder := &dinghyfile.PipelineBuilder{
		Downloader:           f,
		Depman:               wa.Cache,
		TemplateRepo:         wa.Config.TemplateRepo,
		TemplateOrg:          wa.Config.TemplateOrg,
		DinghyfileName:       wa.Config.DinghyFilename,
		DeleteStalePipelines: false,
		AutolockPipelines:    wa.Config.AutoLockPipelines,
		Client:               wa.Client,
		EventClient:          wa.EventClient,
	}

	// Process the push.
	err := wa.ProcessPush(p, builder)
	if err == dinghyfile.ErrMalformedJSON {
		util.WriteHTTPError(w, http.StatusUnprocessableEntity, err)
		return
	} else if err != nil {
		util.WriteHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	// Check if we're in a template repo
	if p.Repo() == wa.Config.TemplateRepo {
		// Set status to pending while we process modules
		p.SetCommitStatus(git.StatusPending)

		// For each module pushed, rebuild dependent dinghyfiles
		for _, file := range p.Files() {
			if err := builder.RebuildModuleRoots(p.Org(), p.Repo(), file); err != nil {
				switch err.(type) {
				case *util.GitHubFileNotFoundErr:
					util.WriteHTTPError(w, http.StatusNotFound, err)
				default:
					util.WriteHTTPError(w, http.StatusInternalServerError, err)
				}
				p.SetCommitStatus(git.StatusError)
				return
			}
		}
		p.SetCommitStatus(git.StatusSuccess)
	}

	w.Write([]byte(`{"status":"accepted"}`))
}

// TODO: get rid of this function and unmarshal JSON in the handlers so that we can
// return proper status codes
func readBody(r *http.Request, dest interface{}) error {
	body, err := httputil.DumpRequest(r, true)
	if err != nil {
		return err
	}
	log.Info("Received payload: ", string(body))

	util.ReadJSON(r.Body, &dest)
	return nil
}
