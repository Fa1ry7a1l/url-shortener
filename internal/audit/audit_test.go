package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileObserver_Notify_AppendsJSONLine(t *testing.T) {
	path := t.TempDir() + "/audit/events.log"
	observer := NewFileObserver(path)

	event := Event{
		TS:     12345678,
		Action: ActionShorten,
		UserID: "user-1",
		URL:    "https://example.com/one",
	}

	require.NoError(t, observer.Notify(context.Background(), event))
	require.NoError(t, observer.Notify(context.Background(), event))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 2)

	var got Event
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &got))
	require.Equal(t, event, got)
}

func TestHTTPObserver_Notify_PostsJSON(t *testing.T) {
	var got Event
	var gotContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		gotContentType = r.Header.Get("Content-Type")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	event := Event{
		TS:     12345678,
		Action: ActionFollow,
		UserID: "user-1",
		URL:    "https://example.com/one",
	}

	observer := NewHTTPObserver(server.URL, server.Client())
	require.NoError(t, observer.Notify(context.Background(), event))
	require.Equal(t, "application/json", gotContentType)
	require.Equal(t, event, got)
}

type recordObserver struct {
	events []Event
}

func (r *recordObserver) Notify(_ context.Context, event Event) error {
	r.events = append(r.events, event)
	return nil
}

func TestSubject_Notify_SendsToAllObservers(t *testing.T) {
	first := &recordObserver{}
	second := &recordObserver{}
	subject := NewSubject(first, second)

	event := Event{TS: 12345678, Action: ActionShorten, URL: "https://example.com"}
	require.NoError(t, subject.Notify(context.Background(), event))

	require.Equal(t, []Event{event}, first.events)
	require.Equal(t, []Event{event}, second.events)
}
