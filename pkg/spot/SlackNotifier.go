package spot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

const defaultMessageTemplate = `
:warning: One or more build agents are offline! :warning:


{{- range $system,$agents := . }}
* {{ $system }}
    {{- range $agent := $agents }}
    * {{ $agent }}
    {{- end }}
{{- end }}
`

type slackPayload struct {
	Text     string `json:"text"`
	Username string `json:"username,omitempty"`
	IconURL  string `json:"icon_url,omitempty"`
}

// SlackNotifier is a Notifier for posting to slack-compatible webhooks
type SlackNotifier struct {
	Endpoint string

	api             *http.Client
	log             *logrus.Entry
	messageTemplate *template.Template
}

// NewSlackNotifier creates an instance of spot.SlackNotifier for
// a given webhook endpoint
func NewSlackNotifier(endpoint, templatePath string) (*SlackNotifier, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("Cannot create a notifier for an empty endpoint")
	}

	if _, err := os.Stat(templatePath); templatePath != "" && os.IsNotExist(err) {
		return nil, fmt.Errorf("Could not locate the message template at '%s'", templatePath)
	}

	var t *template.Template
	var err error

	if templatePath != "" {
		t, err = template.ParseFiles(templatePath)
	} else {
		t, err = template.New("message").Parse(strings.TrimSpace(defaultMessageTemplate))
	}

	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(endpoint, "/") {
		endpoint = strings.TrimSuffix(endpoint, "/")
	}

	return &SlackNotifier{
		Endpoint:        endpoint,
		api:             &http.Client{},
		log:             logrus.WithFields(logrus.Fields{"type": "slack", "endpoint": endpoint}),
		messageTemplate: t,
	}, nil
}

func (s *SlackNotifier) buildMessage(agents map[string][]string) string {
	buff := &bytes.Buffer{}

	if err := s.messageTemplate.Execute(buff, agents); err != nil {
		panic(err)
	} else {
		return buff.String()
	}
}

// Notify implements spot.Notifier.Notify by posting a message
// to a slack-compatible webhook
func (s *SlackNotifier) Notify(agents map[string][]string) error {
	if s.api == nil {
		return fmt.Errorf("Use spot.NewSlackNotifier(...) to construct a SlackNotifier")
	}

	if len(agents) == 0 {
		s.log.Debug("No agents are offline, not sending a notification")
		return nil
	}

	s.log.WithField("offlineCount", len(agents)).Debug("Sending Notification")
	payload := &slackPayload{
		Text:     s.buildMessage(agents),
		Username: "spot",
		IconURL:  "",
	}

	buff := &bytes.Buffer{}
	if err := json.NewEncoder(buff).Encode(payload); err != nil {
		return err
	}

	resp, err := s.api.Post(s.Endpoint, "application/json", buff)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Failed to notify: %s", resp.Status)
	}

	return nil
}
