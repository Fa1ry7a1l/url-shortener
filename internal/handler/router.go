package handler

import (
	"net/http"
)

type Router struct {
	shorten http.HandlerFunc
	resolve http.HandlerFunc
}

func NewRouter(shorten http.HandlerFunc, resolve http.HandlerFunc) *Router {
	return &Router{shorten: shorten, resolve: resolve}
}

// NewServeMux-friendly: один mux, внутри разведение по Path/Method.
func (rt *Router) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", rt.root)
	return mux
}

func (rt *Router) root(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/":
		rt.shorten(w, r)
		return
	case r.Method == http.MethodGet && r.URL.Path != "/" && isSingleSegment(r.URL.Path):
		rt.resolve(w, r)
		return
	default:
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
}

func isSingleSegment(path string) bool {
	// "/abc" OK, "/a/b" not OK
	if path == "" || path[0] != '/' {
		return false
	}
	p := path[1:]
	return p != "" && !containsSlash(p)
}

func containsSlash(s string) bool {
	for _, ch := range s {
		if ch == '/' {
			return true
		}
	}
	return false
}
