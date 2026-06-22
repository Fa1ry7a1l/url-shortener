package handler

import (
	"net/http"
	pprof "net/http/pprof"

	"github.com/Fa1ry7a1l/url-shortener/internal/auth"
	"github.com/go-chi/chi/v5"

	appLogger "github.com/Fa1ry7a1l/url-shortener/internal/logger"
)

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
	r.Get("/debug/pprof/", pprof.Index)
	r.Get("/debug/pprof/cmdline", pprof.Cmdline)
	r.Get("/debug/pprof/profile", pprof.Profile)
	r.Get("/debug/pprof/symbol", pprof.Symbol)
	r.Post("/debug/pprof/symbol", pprof.Symbol)
	r.Get("/debug/pprof/trace", pprof.Trace)
	r.Get("/debug/pprof/{profile}", pprof.Index)
	r.Get("/{id}", rt.resolve)

	return r
}
