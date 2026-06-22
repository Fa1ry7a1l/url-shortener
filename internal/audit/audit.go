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
	ActionShorten = "shorten"
	ActionFollow  = "follow"
)

type Event struct {
	TS     int64  `json:"ts"`
	Action string `json:"action"`
	UserID string `json:"user_id,omitempty"`
	URL    string `json:"url"`
}

func NewEvent(action string, userID string, url string) Event {
	return Event{
		TS:     time.Now().Unix(),
		Action: action,
		UserID: userID,
		URL:    url,
	}
}

type Observer interface {
	Notify(ctx context.Context, event Event) error
}

type Subject struct {
	mu        sync.RWMutex
	observers []Observer
}

func NewSubject(observers ...Observer) *Subject {
	s := &Subject{}
	for _, observer := range observers {
		s.Attach(observer)
	}
	return s
}

func (s *Subject) Attach(observer Observer) {
	if observer == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.observers = append(s.observers, observer)
}

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

type FileObserver struct {
	mu   sync.Mutex
	path string
}

func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

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

type HTTPObserver struct {
	url    string
	client *http.Client
}

func NewHTTPObserver(url string, client *http.Client) *HTTPObserver {
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	return &HTTPObserver{
		url:    url,
		client: client,
	}
}

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
