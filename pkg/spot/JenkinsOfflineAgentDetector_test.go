package spot

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockServer struct {
	mux      *http.ServeMux
	server   *httptest.Server
	teardown func()
}

func mockJenkins(un, pw string) (*mockServer, *JenkinsOfflineAgentDetector) {
	m := http.NewServeMux()
	s := httptest.NewServer(m)

	return &mockServer{
		mux:    m,
		server: s,
		teardown: func() {
			s.Close()
		},
	}, NewJenkinsDetector(s.URL, un, pw)
}

func TestName(t *testing.T) {
	sut := NewJenkinsDetector("http://foo/bar/", "fizz", "buzz")

	require.Equal(t, "[jenkins] http://foo/bar", sut.Name())
}

func TestFindOfflineAgents_ErrorForNilClient(t *testing.T) {
	sut := &JenkinsOfflineAgentDetector{}

	_, err := sut.FindOfflineAgents()

	require.EqualError(t, err, "Use spot.NewJenkinsDetector(...) to construct a JenkinsOfflineAgentDetector")
}

func TestFindOfflineAgents_Query_BadEndpoint(t *testing.T) {
	sut := NewJenkinsDetector("://foo", "fizz", "buzz")

	_, err := sut.FindOfflineAgents()

	require.EqualError(t, err, "parse ://foo/computer/api/json?tree=computer[displayName,offline,offlineCauseReason]: missing protocol scheme")
}

func TestFindOfflineAgents_Query_NonSuccess(t *testing.T) {
	jenkins, sut := mockJenkins("fizz", "buzz")
	defer jenkins.teardown()

	jenkins.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	_, err := sut.FindOfflineAgents()

	require.EqualError(t, err, "Request failed: 400 Bad Request")
}

func TestFindOfflineAgents_Query_NoResponse(t *testing.T) {
	jenkins, sut := mockJenkins("fizz", "buzz")
	jenkins.teardown()

	_, err := sut.FindOfflineAgents()

	require.Error(t, err)
	require.Regexp(t, `dial tcp.*refused`, err.Error())
}

func TestFindOfflineAgents_Query_NotJson(t *testing.T) {
	jenkins, sut := mockJenkins("fizz", "buzz")
	defer jenkins.teardown()

	jenkins.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	_, err := sut.FindOfflineAgents()

	require.Error(t, err, "Empty Body")
}

func TestFindOfflineAgents_NoErrorForNoAgents(t *testing.T) {
	jenkins, sut := mockJenkins("fizz", "buzz")
	defer jenkins.teardown()

	jenkins.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `
			{
				"_class":"hudson.model.ComputerSet",
				"computer":[]
			}
		`)
	})

	result, err := sut.FindOfflineAgents()
	require.NoError(t, err)
	require.Empty(t, result)
}

func TestFindOfflineAgents_MarksOfflineAgents(t *testing.T) {
	jenkins, sut := mockJenkins("fizz", "buzz")
	defer jenkins.teardown()

	jenkins.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `
			{
				"_class":"hudson.model.ComputerSet",
				"computer":[
					{
						"_class":"hudson.model.Hudson$MasterComputer",
						"displayName":"master",
						"offline":false,
						"offlineCauseReason":""
					},
					{
						"_class":"hudson.slaves.SlaveComputer",
						"displayName":"agent1",
						"offline":false,
						"offlineCauseReason":""
					},
					{
						"_class":"hudson.slaves.SlaveComputer",
						"displayName":"agent2",
						"offline":true,
						"offlineCauseReason":"testing"
					},
					{
						"_class":"hudson.slaves.SlaveComputer",
						"displayName":"agent3",
						"offline":true,
						"offlineCauseReason":"testing2"
					}
				]
			}
		`)
	})

	result, err := sut.FindOfflineAgents()

	require.NoError(t, err)
	require.Contains(t, result, "agent2")
	require.Contains(t, result, "agent3")
	require.NotContains(t, result, "agent1")
}
