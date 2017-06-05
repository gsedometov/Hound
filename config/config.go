package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"io/ioutil"
)

const (
	defaultMsBetweenPoll         = 30000
	defaultMaxConcurrentIndexers = 2
	defaultPushEnabled           = false
	defaultPollEnabled           = true
	defaultVcs                   = "git"
	defaultBaseUrl               = "{url}/blob/master/{path}{anchor}"
	defaultAnchor                = "#L{line}"
)

type UrlPattern struct {
	BaseUrl string `json:"base-url"`
	Anchor  string `json:"anchor"`
}

func newUrlPattern () (*UrlPattern){
	return &UrlPattern{
		defaultBaseUrl,
		defaultAnchor,
	}
}

type Repo struct {
	Url               string         `json:"url"`
	MsBetweenPolls    int            `json:"ms-between-poll"`
	Vcs               string         `json:"vcs"`
	VcsConfigMessage  *SecretMessage `json:"vcs-config"`
	UrlPattern        *UrlPattern    `json:"url-pattern"`
	ExcludeDotFiles   bool           `json:"exclude-dot-files"`
	PollUpdatesEnabled bool          `json:"enable-poll-updates"`
	PushUpdatesEnabled bool          `json:"enable-push-updates"`
}

func newRepo () *Repo {
	return &Repo{
		"",
		defaultMsBetweenPoll,
		defaultVcs,
		new(SecretMessage),
		newUrlPattern(),
		true,
		defaultPollEnabled,
		defaultPushEnabled,
	}
}

type Config struct {
	DbPath                string           `json:"dbpath"`
	Repos                 map[string]*Repo `json:"repos"`
	MaxConcurrentIndexers int              `json:"max-concurrent-indexers"`
}

func NewConfig() (*Config){
	return &Config{
		"",
		make(map[string]*Repo, 1),
		defaultMaxConcurrentIndexers,
	}
}

// SecretMessage is just like json.RawMessage but it will not
// marshal its value as JSON. This is to ensure that vcs-config
// is not marshalled into JSON and send to the UI.
type SecretMessage []byte

// This always marshals to an empty object.
func (s *SecretMessage) MarshalJSON() ([]byte, error) {
	return []byte("{}"), nil
}

// See http://golang.org/pkg/encoding/json/#RawMessage.UnmarshalJSON
func (s *SecretMessage) UnmarshalJSON(b []byte) error {
	if b == nil {
		return errors.New("SecretMessage: UnmarshalJSON on nil pointer")
	}
	*s = append((*s)[0:0], b...)
	return nil
}

// Get the JSON encode vcs-config for this repo. This returns nil if
// the repo doesn't declare a vcs-config.
func (r *Repo) VcsConfig() []byte {
	if r.VcsConfigMessage == nil {
		return nil
	}
	return *r.VcsConfigMessage
}

func (r *Repo) UnmarshalJSON(b []byte) error {
	type xrepo Repo
	rep := xrepo(*newRepo())
	if err := json.Unmarshal(b, &rep); err != nil {
		return err
	}
	*r = Repo(rep)
	return nil
}

func LoadFromFile(filename string) (*Config, error) {
	c := NewConfig()
	r, err := os.Open(filename)
	defer r.Close()
	if err != nil {
		return nil, err
	}

	data, _ := ioutil.ReadAll(r)
	if err := json.Unmarshal(data, c); err != nil {
		return nil, err
	}

	c.DbPath, err = filepath.Abs(c.DbPath); if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) ToJsonString() (string, error) {
	b, err := json.Marshal(c.Repos)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
