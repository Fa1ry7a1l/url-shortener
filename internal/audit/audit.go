package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// ActionShorten marks an audit event created when a short URL is registered.
	ActionShorten = "shorten"
	// ActionFollow marks an audit event created when a short URL is resolved.
	ActionFollow = "follow"
)

// Event describes one user-visible action that can be sent to audit observers.
type Event struct {
	// TS is the Unix timestamp when the event was created.
	TS int64 `json:"ts"`
	// Action is the event kind.
	Action string `json:"action"`
	// UserID is the optional authenticated user identifier.
	UserID string `json:"user_id,omitempty"`
	// URL is the original URL associated with the action.
	URL string `json:"url"`
}

// NewEvent creates an audit event with the current Unix timestamp.
func NewEvent(action string, userID string, url string) Event {
	return Event{
		TS:     time.Now().Unix(),
		Action: action,
		UserID: userID,
		URL:    url,
	}
}

// Observer receives audit events from a Subject.
type Observer interface {
	// Notify handles a single audit event.
	Notify(ctx context.Context, event Event) error
}

// Subject publishes audit events to attached observers.
type Subject struct {
	mu        sync.RWMutex
	observers []Observer
}

// NewSubject creates a Subject and attaches the provided observers.
func NewSubject(observers ...Observer) *Subject {
	s := &Subject{}
	for _, observer := range observers {
		s.Attach(observer)
	}
	return s
}

// Attach subscribes an observer to future events.
func (s *Subject) Attach(observer Observer) {
	if observer == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.observers = append(s.observers, observer)
}

// Notify sends an event to every attached observer and joins returned errors.
func (s *Subject) Notify(ctx context.Context, event Event) error {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	observers := append([]Observer(nil), s.observers...)
	s.mu.RUnlock()

	var errs []error
	for _, observer := range observers {
		if err := observer.Notify(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// FileObserver appends audit events as JSON lines to a local file.
type FileObserver struct {
	mu   sync.Mutex
	path string
}

// NewFileObserver creates a file-backed audit observer.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

// Notify writes the event to the configured audit file.
func (o *FileObserver) Notify(_ context.Context, event Event) error {
	if o == nil || o.path == "" {
		return nil
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	o.mu.Lock()
	defer o.mu.Unlock()

	dir := filepath.Dir(o.path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	f, err := os.OpenFile(o.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

// HTTPObserver posts audit events to a remote HTTP endpoint.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver creates an HTTP audit observer.
func NewHTTPObserver(url string, client *http.Client) *HTTPObserver {
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	return &HTTPObserver{
		url:    url,
		client: client,
	}
}

// Notify sends the event as JSON and treats non-2xx responses as errors.
func (o *HTTPObserver) Notify(ctx context.Context, event Event) error {
	if o == nil || o.url == "" {
		return nil
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return errors.New("audit receiver returned non-2xx status")
	}

	return nil
}
