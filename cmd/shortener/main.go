package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

const (
	addr    = "localhost:8080"
	baseURL = "http://localhost:8080"
)

func main() {
	store := repository.NewMemStore()
	idgen := service.NewRandomID(6)
	svc := service.NewShortener(store, idgen, baseURL)

	shortenH := handler.NewShortenHandler(svc)
	resolveH := handler.NewResolveHandler(svc)
	router := handler.NewRouter(shortenH.Handle, resolveH.Handle)

	srv := &http.Server{
		Addr:              addr,
		Handler:           router.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("listening on http://%s\n", srv.Addr)
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
