package spot

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

type node struct {
	Class              string `json:"_class"`
	DisplayName        string `json:"displayName"`
	Offline            bool   `json:"offline"`
	OfflineCauseReason string `json:"offlineCauseReason"`
}

type jenkinsResponse struct {
	Computers []node `json:"computer"`
}

// JenkinsOfflineAgentDetector is an OfflineAgentDetector for watching
// Jenkins agents. If a Username a
type JenkinsOfflineAgentDetector struct {
	APIEndpoint string
	Username    string
	Password    string

	api *http.Client
	log *logrus.Entry
}

// NewJenkinsDetector constructs a JenkinsOfflineAgentDetector
func NewJenkinsDetector(endpoint, un, pw string) *JenkinsOfflineAgentDetector {
	if strings.HasSuffix(endpoint, "/") {
		endpoint = strings.TrimSuffix(endpoint, "/")
	}

	result := &JenkinsOfflineAgentDetector{
		APIEndpoint: endpoint,
		Username:    un,
		Password:    pw,

		api: &http.Client{},
	}

	result.log = logrus.WithField("detector", result.Name())
	return result
}

func (j *JenkinsOfflineAgentDetector) queryAPI() ([]node, error) {
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

func (j *JenkinsOfflineAgentDetector) Name() string {
	return fmt.Sprintf("[jenkins] %s", j.APIEndpoint)
}

func (j *JenkinsOfflineAgentDetector) FindOfflineAgents() ([]string, error) {
	if j.api == nil {
		return nil, fmt.Errorf("Use spot.NewJenkinsDetector(...) to construct a JenkinsOfflineAgentDetector")
	}

	l := j.log.WithField("detector", j.Name())

	offline := []string{}
	nodes, err := j.queryAPI()
	if err != nil {
		return nil, err
	}

	if len(nodes) == 0 {
		l.Warn("No agents found")
	}

	for _, node := range nodes {
		if node.Offline {
			l.WithFields(logrus.Fields{
				"agent":  node.DisplayName,
				"reason": node.OfflineCauseReason,
			}).Warn("Found an offline agent")
			offline = append(offline, node.DisplayName)
		} else {
			l.WithField("agent", node.DisplayName).Debug("Node is online")
		}
	}

	return offline, nil
}
