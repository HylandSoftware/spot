package spot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockSlackServer struct {
	mux      *http.ServeMux
	server   *httptest.Server
	teardown func()
}

func mockSlack() (*mockSlackServer, *SlackNotifier) {
	m := http.NewServeMux()
	s := httptest.NewServer(m)
	n, _ := NewSlackNotifier(s.URL)

	return &mockSlackServer{
		mux:    m,
		server: s,
		teardown: func() {
			s.Close()
		},
	}, n
}

func TestNotify_ErrorForNilClient(t *testing.T) {
	sut := &SlackNotifier{}

	err := sut.Notify([]string{"a", "b", "c"})

	require.EqualError(t, err, "Use spot.NewSlackNotifier(...) to construct a SlackNotifier")
}

func TestNotify_NoAgents(t *testing.T) {
	slack, sut := mockSlack()

	called := false
	slack.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	err := sut.Notify([]string{})

	require.NoError(t, err)
	require.False(t, called, "Expected no API calls to be made")
}

func TestNotify_HttpErrorFailure(t *testing.T) {
	slack, sut := mockSlack()

	called := false
	slack.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	sut.Endpoint = "thisisnotaprotocol://foo"
	err := sut.Notify([]string{"a", "b", "c"})

	require.EqualError(t, err, "Post thisisnotaprotocol://foo: unsupported protocol scheme \"thisisnotaprotocol\"")
	require.False(t, called, "Expected no API calls to be made")
}

func TestNotify_NonSuccessResponse(t *testing.T) {
	slack, sut := mockSlack()

	called := false
	slack.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusBadRequest)
	})

	err := sut.Notify([]string{"a", "b", "c"})

	require.EqualError(t, err, "Failed to notify: 400 Bad Request")
	require.True(t, called, "Expected an API call to be made")
}

func TestNotify(t *testing.T) {
	slack, sut := mockSlack()

	var payload *slackPayload
	slack.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		payload = &slackPayload{}

		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(payload); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	})

	err := sut.Notify([]string{"a", "b", "c"})

	require.NoError(t, err)
	require.NotNil(t, payload)
	require.Equal(t, payload.Text, ":warning: One or more build agents are offline! :warning:\n\n* a\n* b\n* c")
	require.Equal(t, payload.Username, "spot")
}
