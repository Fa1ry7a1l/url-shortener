package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Router struct {
	shorten http.HandlerFunc
	resolve http.HandlerFunc
}

func NewRouter(shorten http.HandlerFunc, resolve http.HandlerFunc) *Router {
	return &Router{shorten: shorten, resolve: resolve}
}

func (rt *Router) Handler() http.Handler {
	r := chi.NewRouter()

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})

	r.Post("/", rt.shorten)
	r.Get("/{id}", rt.resolve)

	return r
}
