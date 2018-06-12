package spot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

type slackPayload struct {
	Text     string `json:"text"`
	Username string `json:"username,omitempty"`
	IconURL  string `json:"icon_url,omitempty"`
}

// SlackNotifier is a Notifier for posting to slack-compatible webhooks
type SlackNotifier struct {
	Endpoint string

	api *http.Client
	log *logrus.Entry
}

// NewSlackNotifier creates an instance of spot.SlackNotifier for
// a given webhook endpoint
func NewSlackNotifier(endpoint string) (*SlackNotifier, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("Cannot create a notifier for an empty endpoint")
	}

	if strings.HasSuffix(endpoint, "/") {
		endpoint = strings.TrimSuffix(endpoint, "/")
	}

	return &SlackNotifier{
		Endpoint: endpoint,
		api:      &http.Client{},
		log:      logrus.WithFields(logrus.Fields{"type": "slack", "endpoint": endpoint}),
	}, nil
}

func buildMessage(agents []string) string {
	agentSlug := ""
	for _, v := range agents {
		agentSlug += fmt.Sprintf("* %s\n", v)
	}

	return fmt.Sprintf(":warning: One or more build agents are offline! :warning:\n\n%s", strings.TrimSpace(agentSlug))
}

// Notify implements spot.Notifier.Notify by posting a message
// to a slack-compatible webhook
func (s *SlackNotifier) Notify(agents []string) error {
	if s.api == nil {
		return fmt.Errorf("Use spot.NewSlackNotifier(...) to construct a SlackNotifier")
	}

	if len(agents) == 0 {
		s.log.Debug("No agents are offline, not sending a notification")
		return nil
	}

	s.log.WithField("offlineCount", len(agents)).Debug("Sending Notification")
	payload := &slackPayload{
		Text:     buildMessage(agents),
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
