package bamboo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	bambooAgentAPICall = "rest/api/latest/agent"
)

type bambooAgent struct {
	ID      int64
	Name    string
	Type    string
	Active  bool
	Enabled bool
	Busy    bool
}

// OfflineAgentDetector is a spot.OfflineAgentDetector for watching
// Bamboo agents. If a Username and password are provided, API requests
// will use HTTP Basic authentication with the provided credentials.
type OfflineAgentDetector struct {
	APIEndpoint string
	Username    string
	Password    string

	api *http.Client
	log *logrus.Entry
}

// NewDetectorFromArg parses a configuration string into a
// BambooOfflineAgentDetector. The format of the string is one of
// the following:
//
// <url>: an http:// or https:// URL to a bamboo instance that does
//        not require authentication
//
// <url>,<un>,<pw>: an http:// or https:// URL to a bamboo instance.
//                  <un> and <pw> will be used to authenticate API
//                  requests. <pw> may be a password or access token.
func NewDetectorFromArg(arg string) (*OfflineAgentDetector, error) {
	if arg == "" {
		return nil, fmt.Errorf("No arg specified")
	}

	parts := strings.Split(arg, ",")
	switch len(parts) {
	case 1:
		return NewDetector(parts[0], "", ""), nil
	case 3:
		return NewDetector(parts[0], parts[1], parts[2]), nil
	default:
		return nil, fmt.Errorf("The format of the config string was not recognized: %s", arg)
	}
}

// NewDetector constructs a BambooOfflineAgentDetector
func NewDetector(endpoint, un, pw string) *OfflineAgentDetector {
	if strings.HasSuffix(endpoint, "/") {
		endpoint = strings.TrimSuffix(endpoint, "/")
	}

	result := &OfflineAgentDetector{
		APIEndpoint: endpoint,
		Username:    un,
		Password:    pw,

		api: &http.Client{},
	}

	result.log = logrus.WithField("detector", result.Name())
	return result
}

func (b *OfflineAgentDetector) queryAPI() ([]bambooAgent, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", b.APIEndpoint, bambooAgentAPICall), nil)
	if err != nil {
		return nil, err
	}

	if b.Username != "" && b.Password != "" {
		b.log.WithField("username", b.Username).WithField("uri", req.URL.String()).Debug("Using basic auth")
		req.SetBasicAuth(b.Username, b.Password)
	}

	resp, err := b.api.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Request failed: %s", resp.Status)
	}

	response := []bambooAgent{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response, nil
}

// Name implements spot.OfflineAgentDetector.Name by returning
// the name of the detector formatted as '[bamboo] {endpoint}'
func (b *OfflineAgentDetector) Name() string {
	return fmt.Sprintf("[bamboo] %s", b.APIEndpoint)
}

// FindOfflineAgents implements spot.OfflineAgentDetector.FindOfflineAgents
// by querying the bamboo agent API endpoint and returning any agents
// that have their Active property set to true.
func (b *OfflineAgentDetector) FindOfflineAgents() ([]string, error) {
	if b.api == nil {
		return nil, fmt.Errorf("Use spot.NewBambooDetector(...) to construct a BambooOfflineAgentDetector")
	}

	offline := []string{}
	nodes, err := b.queryAPI()
	if err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		b.log.Warn("No agents found")
	}

	for _, node := range nodes {
		if !node.Active {
			b.log.WithField("agent", node.Name).Warn("Found an offline agent")
			offline = append(offline, node.Name)
		} else {
			b.log.WithField("agent", node.Name).Debug("Node is online")
		}
	}

	return offline, nil
}
