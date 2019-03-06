// Package settings is a single place to put all of the application settings.
package settings

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"io/ioutil"
	"os"
	"strings"

	"github.com/armory-io/dinghy/pkg/util"
	"github.com/armory/go-yaml-tools/pkg/spring"
	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
)

func NewDefaultSettings() Settings {
	return Settings{
		DinghyFilename:    "dinghyfile",
		TemplateRepo:      "dinghy-templates",
		AutoLockPipelines: "true",
		GitHubCredsPath:   util.GetenvOrDefault("GITHUB_TOKEN_PATH", os.Getenv("HOME")+"/.armory/cache/github-creds.txt"),
		GithubEndpoint:    "https://api.github.com",
		StashCredsPath:    util.GetenvOrDefault("STASH_TOKEN_PATH", os.Getenv("HOME")+"/.armory/cache/stash-creds.txt"),
		StashEndpoint:     "http://localhost:7990/rest/api/1.0",
		Logging: Logging{
			File:  "",
			Level: "INFO",
		},
		spinnakerSupplied: spinnakerSupplied{
			Orca: spinnakerService{
				Enabled: "true",
				BaseURL: util.GetenvOrDefault("ORCA_BASE_URL", "http://orca:8083"),
			},
			Front50: spinnakerService{
				Enabled: "true",
				BaseURL: util.GetenvOrDefault("FRONT50_BASE_URL", "http://front50:8080"),
			},
			Fiat: fiat{
				spinnakerService: spinnakerService{
					Enabled: "false",
					BaseURL: util.GetenvOrDefault("FIAT_BASE_URL", "http://fiat:7003"),
				},
				AuthUser: "",
			},
			Redis: Redis{
				Host:     util.GetenvOrDefault("REDIS_HOST", "redis"),
				Port:     util.GetenvOrDefault("REDIS_PORT", "6379"),
				Password: util.GetenvOrDefault("REDIS_PASSWORD", ""),
			},
		},
	}
}

// LoadSettings loads the Spring config from the default Spinnaker paths
// and merges default settings with the loaded settings
func LoadSettings() (*Settings, error) {
	springConfig, err := loadProfiles()
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	settings, err := ConfigureSettings(NewDefaultSettings(), springConfig)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	return settings, nil
}

func ConfigureSettings(defaultSettings, overrides Settings) (*Settings, error) {

	if err := mergo.Merge(&defaultSettings, overrides, mergo.WithOverride); err != nil {
		log.Errorf("failed to merge custom config with default: %s", err.Error())
		return nil, err
	}

	// If Github token not passed directly
	// Required for backwards compatibility
	if defaultSettings.GitHubToken == "" {
		// load github api token
		if _, err := os.Stat(defaultSettings.GitHubCredsPath); err == nil {
			creds, err := ioutil.ReadFile(defaultSettings.GitHubCredsPath)
			if err != nil {
				panic(err)
			}
			c := strings.Split(strings.TrimSpace(string(creds)), ":")
			if len(c) < 2 {
				panic("github creds file should have format 'username:token'")
			}
			defaultSettings.GitHubToken = c[1]
			log.Info("Successfully loaded github api creds")
		}
	}

	// If Stash token not passed directly
	// Required for backwards compatibility
	if defaultSettings.StashToken == "" || defaultSettings.StashUsername == "" {
		// load stash api creds
		if _, err := os.Stat(defaultSettings.StashCredsPath); err == nil {
			creds, err := ioutil.ReadFile(defaultSettings.StashCredsPath)
			if err != nil {
				panic(err)
			}
			c := strings.Split(strings.TrimSpace(string(creds)), ":")
			if len(c) < 2 {
				panic("stash creds file should have format 'username:token'")
			}
			defaultSettings.StashUsername = c[0]
			defaultSettings.StashToken = c[1]
			log.Info("Successfully loaded stash api creds")
		}
	}

	// Required for backwards compatibility
	if defaultSettings.Deck.BaseURL == "" && defaultSettings.SpinnakerUIURL != "" {
		log.Warn("Spinnaker UI URL should be set with ${services.deck.baseUrl}")
		defaultSettings.Deck.BaseURL = defaultSettings.SpinnakerUIURL
	}

	// Take the FiatUser setting if fiat is enabled (coming from hal settings)
	if defaultSettings.Fiat.Enabled == "true" && defaultSettings.FiatUser != "" {
		defaultSettings.Fiat.AuthUser = defaultSettings.FiatUser
	}

	c, _ := json.Marshal(defaultSettings)
	log.Infof("The following settings have been loaded: %v", string(c))

	return &defaultSettings, nil
}

func decodeProfilesToSettings(profiles map[string]interface{}, s *Settings) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           s,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(profiles)
}

func loadProfiles() (Settings, error) {
	// var s Settings
	var config Settings
	propNames := []string{"spinnaker", "dinghy"}
	c, err := spring.LoadDefault(propNames)
	if err != nil {
		log.Errorf("Could not load yaml configs - %v", err)
		return config, err
	}

	if err := decodeProfilesToSettings(c, &config); err != nil {
		return config, err
	}

	return config, nil
}
