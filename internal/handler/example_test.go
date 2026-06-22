package handler_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/Fa1ry7a1l/url-shortener/internal/auth"
	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

type exampleIDGenerator struct {
	ids  []string
	next int
}

func (g *exampleIDGenerator) NewID() (string, error) {
	if g.next >= len(g.ids) {
		id := fmt.Sprintf("id%d", g.next+1)
		g.next++
		return id, nil
	}

	id := g.ids[g.next]
	g.next++
	return id, nil
}

func newExampleServer(ids []string, withAuth bool) *httptest.Server {
	store := repository.NewMemStore()
	svc := service.NewShortener(store, &exampleIDGenerator{ids: ids}, "http://short.local")

	var authManager *auth.Manager
	if withAuth {
		authManager = auth.NewManager("example-secret", time.Hour)
	}

	router := handler.NewRouter(
		handler.NewShortenHandler(svc).Handle,
		handler.NewShortenJSONHandler(svc).Handle,
		handler.NewShortenBatchHandler(svc).Handle,
		handler.NewResolveHandler(svc).Handle,
		handler.NewUserURLsHandler(svc).Handle,
		handler.NewDeleteUserURLsHandler(svc).Handle,
		handler.NewPingHandler(store).Handle,
		nil,
		authManager,
	)

	return httptest.NewServer(router.Handler())
}

func addCookies(req *http.Request, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
}

func ExampleRouter_plainTextEndpoints() {
	server := newExampleServer([]string{"abc123"}, false)
	defer server.Close()

	resp, err := http.Post(server.URL+"/", "text/plain", strings.NewReader("https://practicum.yandex.ru"))
	if err != nil {
		panic(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	_ = resp.Body.Close()

	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))

	client := server.Client()
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err = client.Get(server.URL + "/abc123")
	if err != nil {
		panic(err)
	}
	_ = resp.Body.Close()

	fmt.Println(resp.StatusCode)
	fmt.Println(resp.Header.Get("Location"))

	// Output:
	// 201
	// http://short.local/abc123
	// 307
	// https://practicum.yandex.ru
}

func ExampleRouter_jsonEndpoints() {
	server := newExampleServer([]string{"json1", "batch1", "batch2"}, false)
	defer server.Close()

	resp, err := http.Post(
		server.URL+"/api/shorten",
		"application/json",
		strings.NewReader(`{"url":"https://example.com/article"}`),
	)
	if err != nil {
		panic(err)
	}
	var shortResp struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&shortResp); err != nil {
		panic(err)
	}
	_ = resp.Body.Close()

	fmt.Println(resp.StatusCode, shortResp.Result)

	resp, err = http.Post(
		server.URL+"/api/shorten/batch",
		"application/json",
		strings.NewReader(`[
			{"correlation_id":"first","original_url":"https://example.com/a"},
			{"correlation_id":"second","original_url":"https://example.com/b"}
		]`),
	)
	if err != nil {
		panic(err)
	}
	var batchResp []struct {
		CorrelationID string `json:"correlation_id"`
		ShortURL      string `json:"short_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		panic(err)
	}
	_ = resp.Body.Close()

	for _, item := range batchResp {
		fmt.Println(resp.StatusCode, item.CorrelationID, item.ShortURL)
	}

	// Output:
	// 201 http://short.local/json1
	// 201 first http://short.local/batch1
	// 201 second http://short.local/batch2
}

func ExampleRouter_userEndpoints() {
	server := newExampleServer([]string{"mine1"}, true)
	defer server.Close()

	client := server.Client()
	req, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/api/shorten",
		strings.NewReader(`{"url":"https://example.com/profile"}`),
	)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	cookies := resp.Cookies()
	_ = resp.Body.Close()

	req, err = http.NewRequest(http.MethodGet, server.URL+"/api/user/urls", nil)
	if err != nil {
		panic(err)
	}
	addCookies(req, cookies)

	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	var urls []struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&urls); err != nil {
		panic(err)
	}
	_ = resp.Body.Close()

	fmt.Println(resp.StatusCode, len(urls))
	fmt.Println(urls[0].OriginalURL)
	fmt.Println(urls[0].ShortURL)

	req, err = http.NewRequest(http.MethodDelete, server.URL+"/api/user/urls", strings.NewReader(`["mine1"]`))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	addCookies(req, cookies)

	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	_ = resp.Body.Close()

	fmt.Println(resp.StatusCode)

	// Output:
	// 200 1
	// https://example.com/profile
	// http://short.local/mine1
	// 202
}

func ExampleRouter_pingEndpoint() {
	server := newExampleServer([]string{"ping1"}, false)
	defer server.Close()

	resp, err := http.Get(server.URL + "/ping")
	if err != nil {
		panic(err)
	}
	_ = resp.Body.Close()

	fmt.Println(resp.StatusCode)

	// Output:
	// 200
}
