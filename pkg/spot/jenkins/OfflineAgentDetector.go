package jenkins

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	nodeAPICall = "computer/api/json?tree=computer[displayName,offline,offlineCauseReason]"
)

var (
	classWhitelist = []string{
		"hudson.slaves.SlaveComputer",
	}
)

type node struct {
	Class              string `json:"_class"`
	DisplayName        string `json:"displayName"`
	Offline            bool   `json:"offline"`
	OfflineCauseReason string `json:"offlineCauseReason"`
}

type jenkinsResponse struct {
	Computers []node `json:"computer"`
}

// OfflineAgentDetector is a spot.OfflineAgentDetector for watching
// Jenkins agents. If a Username and password are provided, API requests
// will use HTTP Basic authentication with the provided credentials.
type OfflineAgentDetector struct {
	APIEndpoint string
	Username    string
	Password    string

	api *http.Client
	log *logrus.Entry
}

// UseClassWhitelist sets the class whitelist to filter offline agents by.
func UseClassWhitelist(whitelist []string) {
	logrus.WithField("whitelist", whitelist).Warning("The default class whitelist has been overridden")
	classWhitelist = whitelist
}

// NewDetectorFromArg parses a configuration string into a
// JenkinsOfflineAgentDetector. The format of the string is one of
// the following:
//
// <url>: an http:// or https:// URL to a jenkins instance that does
//        not require authentication
//
// <url>,<un>,<pw>: an http:// or https:// URL to a jenkins instance.
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

// NewDetector constructs an OfflineAgentDetector
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

func (j *OfflineAgentDetector) queryAPI() ([]node, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", j.APIEndpoint, nodeAPICall), nil)
	if err != nil {
		return nil, err
	}

	if j.Username != "" && j.Password != "" {
		req.SetBasicAuth(j.Username, j.Password)
	}

	resp, err := j.api.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Request failed: %s", resp.Status)
	}

	response := &jenkinsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return nil, err
	}

	return response.Computers, nil
}

// Name implements spot.OfflineAgentDetector.Name by returning
// the name of the detector formatted as '[jenkins] {endpoint}'
func (j *OfflineAgentDetector) Name() string {
	return fmt.Sprintf("[jenkins] %s", j.APIEndpoint)
}

// FindOfflineAgents implements spot.OfflineAgentDetector.FindOfflineAgents
// by querying the jenkins computer API endpoint and returning any nodes
// that have their Offline property set to true.
func (j *OfflineAgentDetector) FindOfflineAgents() ([]string, error) {
	if j.api == nil {
		return nil, fmt.Errorf("Use spot.NewJenkinsDetector(...) to construct a JenkinsOfflineAgentDetector")
	}

	offline := []string{}
	nodes, err := j.queryAPI()
	if err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		j.log.Warn("No agents found")
	}

	for _, node := range nodes {
		whitelisted := false
		for _, class := range classWhitelist {
			if node.Class == class {
				whitelisted = true
				break
			}
		}

		if !whitelisted {
			j.log.WithFields(logrus.Fields{
				"agent": node.DisplayName,
				"class": node.Class,
			}).Debug("Skipping agent (class not whitelisted)")
		} else if node.Offline {
			j.log.WithFields(logrus.Fields{
				"agent":  node.DisplayName,
				"reason": node.OfflineCauseReason,
			}).Warn("Found an offline agent")
			offline = append(offline, node.DisplayName)
		} else {
			j.log.WithField("agent", node.DisplayName).Debug("Node is online")
		}
	}

	return offline, nil
}
