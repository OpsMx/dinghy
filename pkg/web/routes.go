package web

import (
	"bytes"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/armory-io/dinghy/pkg/git/bbcloud"
	"github.com/armory-io/dinghy/pkg/spinnaker"

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

type SpinnakerServices struct {
	Front50API  spinnaker.Front50API
	PipelineAPI spinnaker.PipelineAPI
	OrcaAPI     spinnaker.OrcaAPI
}

type WebAPI struct {
	Config            settings.Settings
	SpinnakerServices SpinnakerServices
	Cache             dinghyfile.DependencyManager
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
		PipelineAPI:          wa.SpinnakerServices.PipelineAPI,
		DeleteStalePipelines: false,
		AutolockPipelines:    wa.Config.AutoLockPipelines,
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	df := buf.String()
	log.Info("Received payload: ", df)
	fileService["dinghyfile"] = df

	err := builder.ProcessDinghyfile("", "", "dinghyfile")
	if err != nil {
		util.WriteHTTPError(w, http.StatusInternalServerError, err)
		return
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

	// TODO: WebAPI already has the fields that are being assigned here and it's
	// the receiver on the buildPipelines. We don't need to reassign the values to
	// fileService here.
	p.GitHubToken = wa.Config.GitHubToken
	p.GitHubEndpoint = wa.Config.GithubEndpoint
	p.DeckBaseURL = wa.Config.Deck.BaseURL
	fileService := github.FileService{
		GitHubEndpoint: wa.Config.GithubEndpoint,
		GitHubToken:    wa.Config.GitHubToken,
	}

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
		PipelineAPI:          wa.SpinnakerServices.PipelineAPI,
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
