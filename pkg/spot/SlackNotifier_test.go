package spot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
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
	n, _ := NewSlackNotifier(s.URL, "")

	return &mockSlackServer{
		mux:    m,
		server: s,
		teardown: func() {
			s.Close()
		},
	}, n
}

func TestNew_UsesDefaultTemplate(t *testing.T) {
	sut, _ := NewSlackNotifier("http://endpoint", "")
	buff := &bytes.Buffer{}

	err := sut.messageTemplate.Execute(buff, map[string][]string{"a": []string{"b", "c"}})

	require.NoError(t, err)
	require.Equal(t, ":warning: One or more build agents are offline! :warning:\n* a\n    * b\n    * c", buff.String())
}

func TestNew_CanUseCustomTemplate(t *testing.T) {
	tpl, err := ioutil.TempFile("", "template")
	require.NoError(t, err)
	defer os.Remove(tpl.Name())

	_, err = tpl.Write([]byte("foo"))
	require.NoError(t, err)

	tpl.Close()

	buff := &bytes.Buffer{}
	sut, err := NewSlackNotifier("http://endpoint", tpl.Name())
	require.NoError(t, err)

	err = sut.messageTemplate.Execute(buff, map[string][]string{"a": []string{"b", "c"}})

	require.Equal(t, "foo", buff.String())
}

func TestNew_ErrorForEmptyEndpoint(t *testing.T) {
	sut, err := NewSlackNotifier("", "")

	require.Nil(t, sut)
	require.EqualError(t, err, "Cannot create a notifier for an empty endpoint")
}

func TestNew_ErrorForTemplateNotFound(t *testing.T) {
	tpl, err := ioutil.TempFile("", "template")
	tpl.Close()
	require.NoError(t, err)
	require.NoError(t, os.Remove(tpl.Name()))

	sut, err := NewSlackNotifier("http://foo", tpl.Name())

	require.Nil(t, sut)
	require.EqualError(t, err, fmt.Sprintf("Could not locate the message template at '%s'", tpl.Name()))
}

func TestNew_ErrorForTemplateError(t *testing.T) {
	tpl, err := ioutil.TempFile("", "template")
	require.NoError(t, err)
	defer os.Remove(tpl.Name())

	_, err = tpl.Write([]byte("{{ foo"))
	require.NoError(t, err)

	tpl.Close()

	sut, err := NewSlackNotifier("http://endpoint", tpl.Name())

	require.Nil(t, sut)
	require.Error(t, err)
}

func TestNew_StripsTrailingSlash(t *testing.T) {
	sut, err := NewSlackNotifier("http://foo/", "")

	require.NoError(t, err)
	require.Equal(t, "http://foo", sut.Endpoint)
}

func TestNotify_ErrorForNilClient(t *testing.T) {
	sut := &SlackNotifier{}

	err := sut.Notify(map[string][]string{"a": []string{"b,c"}, "d": []string{"e", "f"}})

	require.EqualError(t, err, "Use spot.NewSlackNotifier(...) to construct a SlackNotifier")
}

func TestNotify_NoAgents(t *testing.T) {
	slack, sut := mockSlack()

	called := false
	slack.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	err := sut.Notify(map[string][]string{})

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
	err := sut.Notify(map[string][]string{"a": []string{"b,c"}, "d": []string{"e", "f"}})

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

	err := sut.Notify(map[string][]string{"a": []string{"b,c"}, "d": []string{"e", "f"}})

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

	err := sut.Notify(map[string][]string{"a": []string{"b,c"}, "d": []string{"e", "f"}})

	require.NoError(t, err)
	require.NotNil(t, payload)
	require.Equal(t, ":warning: One or more build agents are offline! :warning:\n* a\n    * b,c\n* d\n    * e\n    * f", payload.Text)
	require.Equal(t, "spot", payload.Username)
}
