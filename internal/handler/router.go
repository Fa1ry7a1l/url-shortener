package handler

import (
	"net/http"

	"github.com/Fa1ry7a1l/url-shortener/internal/auth"
	"github.com/go-chi/chi/v5"

	appLogger "github.com/Fa1ry7a1l/url-shortener/internal/logger"
)

// Router wires all HTTP endpoints and middleware for the shortener API.
type Router struct {
	shorten        http.HandlerFunc
	shortenJSON    http.HandlerFunc
	shortenBatch   http.HandlerFunc
	resolve        http.HandlerFunc
	userURLs       http.HandlerFunc
	deleteUserURLs http.HandlerFunc
	ping           http.HandlerFunc
	logger         appLogger.Logger
	authManager    *auth.Manager
}

// NewRouter creates a Router from endpoint handlers and optional middleware dependencies.
func NewRouter(
	shorten http.HandlerFunc,
	shortenJSON http.HandlerFunc,
	shortenBatch http.HandlerFunc,
	resolve http.HandlerFunc,
	userURLs http.HandlerFunc,
	deleteUserURLs http.HandlerFunc,
	ping http.HandlerFunc,
	logger appLogger.Logger,
	authManager *auth.Manager,
) *Router {
	return &Router{
		shorten:        shorten,
		shortenJSON:    shortenJSON,
		shortenBatch:   shortenBatch,
		resolve:        resolve,
		userURLs:       userURLs,
		deleteUserURLs: deleteUserURLs,
		ping:           ping,
		logger:         logger,
		authManager:    authManager,
	}
}

// Handler builds the public HTTP handler tree with API routes, gzip, auth, and logging.
func (rt *Router) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(GzipHandle)

	if rt.logger != nil {
		r.Use(appLogger.RequestLogger(rt.logger))
	}
	if rt.authManager != nil {
		r.Use(rt.authManager.Middleware)
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})

	r.Post("/", rt.shorten)
	r.Post("/api/shorten", rt.shortenJSON)
	r.Post("/api/shorten/batch", rt.shortenBatch)
	r.Get("/api/user/urls", rt.userURLs)
	r.Delete("/api/user/urls", rt.deleteUserURLs)
	r.Get("/ping", rt.ping)
	r.Get("/{id}", rt.resolve)

	return r
}
