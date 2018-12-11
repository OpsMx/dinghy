package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/armory-io/dinghy/pkg/git"
	"github.com/armory-io/dinghy/pkg/settings"
)

/* Example: POST /repos/:owner/:repo/statuses/:sha
{
  "state": "success",
  "target_url": "https://example.com/build/status",
  "description": "The build succeeded!",
  "context": "continuous-integration/jenkins"
} */

// Status is a payload that we send to the Github API to set commit status
type Status struct {
	State       string `json:"state"`
	TargetURL   string `json:"target_url"`
	Description string `json:"description"`
	Context     string `json:"context"`
}

// SetCommitStatus sets the commit status
func (p *Push) SetCommitStatus(s git.Status) {
	update := newStatus(s)
	for _, c := range p.Commits {
		sha := c.ID // not sure if this is right.
		url := fmt.Sprintf("%s/repos/%s/%s/statuses/%s",
			settings.S.GithubEndpoint,
			p.Org(),
			p.Repo(),
			sha)
		body, err := json.Marshal(update)
		if err != nil {
			log.Debug("Could not unmarshall ", update, ": ", err)
			return
		}
		log.Info(fmt.Sprintf("Updating commit %s for %s/%s to %s.", sha, p.Org(), p.Repo(), string(s)))
		log.Debug("POST ", url, " - ", string(body))
		req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
		req.Header.Add("Authorization", "token "+settings.S.GitHubToken)
		resp, err := http.DefaultClient.Do(req)
		if resp != nil {
			defer resp.Body.Close()
			httputil.DumpResponse(resp, true)
		}
		if err != nil {
			log.Error(err)
			return
		}
	}
}

func newStatus(s git.Status) Status {
	ret := Status{
		State:     string(s),
		TargetURL: settings.S.Deck.BaseURL,
		Context:   "continuous-deployment/dinghy",
	}
	switch s {
	case git.StatusSuccess:
		ret.Description = "Pipeline definitions updated!"
	case git.StatusError:
		ret.Description = "Error updating pipeline definitions!"
	case git.StatusFailure:
		ret.Description = "Failed to update pipeline definitions!"
	case git.StatusPending:
		ret.Description = "Updating pipeline definitions..."
	}
	return ret
}
