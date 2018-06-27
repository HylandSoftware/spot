package bamboo

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockBambooServer struct {
	mux      *http.ServeMux
	server   *httptest.Server
	teardown func()
}

func mockBamboo(un, pw string) (*mockBambooServer, *OfflineAgentDetector) {
	m := http.NewServeMux()
	s := httptest.NewServer(m)

	return &mockBambooServer{
		mux:    m,
		server: s,
		teardown: func() {
			s.Close()
		},
	}, NewDetector(s.URL, un, pw)
}

func TestNewBambooDetectorFromArg_ErrorForEmpty(t *testing.T) {
	_, err := NewDetectorFromArg("")

	require.EqualError(t, err, "No arg specified")
}

func TestNewBambooDetectorFromArg_ErrorForMalformatted(t *testing.T) {
	_, err := NewDetectorFromArg("http://foo,bar,baz,fizz,buzz")

	require.EqualError(t, err, fmt.Sprintf("The format of the config string was not recognized: %s", "http://foo,bar,baz,fizz,buzz"))
}

func TestNewBambooDetectorFromArg_NoCredentials(t *testing.T) {
	sut, err := NewDetectorFromArg("http://foo/")

	require.NoError(t, err)
	require.Equal(t, "http://foo", sut.APIEndpoint)
	require.Empty(t, sut.Username)
	require.Empty(t, sut.Password)
}

func TestNewBambooDetectorFromArg_WithCredentials(t *testing.T) {
	sut, err := NewDetectorFromArg("http://foo/,un,pw")

	require.NoError(t, err)
	require.Equal(t, "http://foo", sut.APIEndpoint)
	require.Equal(t, "un", sut.Username)
	require.Equal(t, "pw", sut.Password)
}

func TestName(t *testing.T) {
	sut := NewDetector("http://foo/bar/", "fizz", "buzz")

	require.Equal(t, "[bamboo] http://foo/bar", sut.Name())
}

func TestFindOfflineAgents_ErrorForNilClient(t *testing.T) {
	sut := &OfflineAgentDetector{}

	_, err := sut.FindOfflineAgents()

	require.EqualError(t, err, "Use spot.NewBambooDetector(...) to construct a BambooOfflineAgentDetector")
}

func TestFindOfflineAgents_Query_BadEndpoint(t *testing.T) {
	sut := NewDetector("://foo", "fizz", "buzz")

	_, err := sut.FindOfflineAgents()

	require.EqualError(t, err, "parse ://foo/rest/api/latest/agent: missing protocol scheme")
}

func TestFindOfflineAgents_Query_NonSuccess(t *testing.T) {
	bamboo, sut := mockBamboo("fizz", "buzz")
	defer bamboo.teardown()

	bamboo.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	_, err := sut.FindOfflineAgents()

	require.EqualError(t, err, "Request failed: 400 Bad Request")
}

func TestFindOfflineAgents_Query_NoResponse(t *testing.T) {
	bamboo, sut := mockBamboo("fizz", "buzz")
	bamboo.teardown()

	_, err := sut.FindOfflineAgents()

	require.Error(t, err)
	require.Regexp(t, `dial tcp.*refused`, err.Error())
}

func TestFindOfflineAgents_Query_NotJson(t *testing.T) {
	bamboo, sut := mockBamboo("fizz", "buzz")
	defer bamboo.teardown()

	bamboo.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	_, err := sut.FindOfflineAgents()

	require.Error(t, err, "Empty Body")
}

func TestFindOfflineAgents_NoErrorForNoAgents(t *testing.T) {
	bamboo, sut := mockBamboo("fizz", "buzz")
	defer bamboo.teardown()

	bamboo.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, "[]")
	})

	result, err := sut.FindOfflineAgents()
	require.NoError(t, err)
	require.Empty(t, result)
}

func TestFindOfflineAgents_MarksOfflineAgents(t *testing.T) {
	bamboo, sut := mockBamboo("fizz", "buzz")
	defer bamboo.teardown()

	bamboo.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `
			[
				{
					"id": 1,
					"name": "agent1",
					"type": "REMOTE",
					"active": true,
					"enabled": true,
					"busy": false
				},
				{
					"id": 2,
					"name": "agent2",
					"type": "REMOTE",
					"active": false,
					"enabled": true,
					"busy": false
				},
				{
					"id": 3,
					"name": "agent3",
					"type": "REMOTE",
					"active": false,
					"enabled": true,
					"busy": false
				}
			]
		`)
	})

	result, err := sut.FindOfflineAgents()

	require.NoError(t, err)
	require.Contains(t, result, "agent2")
	require.Contains(t, result, "agent3")
	require.NotContains(t, result, "agent1")
}
