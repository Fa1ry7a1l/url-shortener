package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	appLogger "github.com/Fa1ry7a1l/url-shortener/internal/logger"
)

type Router struct {
	shorten     http.HandlerFunc
	shortenJSON http.HandlerFunc
	resolve     http.HandlerFunc
	ping        http.HandlerFunc
	logger      appLogger.Logger
}

func NewRouter(
	shorten http.HandlerFunc,
	shortenJSON http.HandlerFunc,
	resolve http.HandlerFunc,
	ping http.HandlerFunc,
	logger appLogger.Logger,
) *Router {
	return &Router{
		shorten:     shorten,
		shortenJSON: shortenJSON,
		resolve:     resolve,
		ping:        ping,
		logger:      logger,
	}
}

func (rt *Router) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(GzipHandle)

	if rt.logger != nil {
		r.Use(appLogger.RequestLogger(rt.logger))
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})

	r.Post("/", rt.shorten)
	r.Post("/api/shorten", rt.shortenJSON)
	r.Get("/ping", rt.ping)
	r.Get("/{id}", rt.resolve)

	return r
}
