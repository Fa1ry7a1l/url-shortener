package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/Fa1ry7a1l/url-shortener/internal/config"
	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

func main() {

	cfg := config.New()
	store := repository.NewMemStore()
	idgen := service.NewRandomID(cfg.IDLength)
	svc := service.NewShortener(store, idgen, cfg.BaseURL)

	shortenH := handler.NewShortenHandler(svc)
	resolveH := handler.NewResolveHandler(svc)
	router := handler.NewRouter(shortenH.Handle, resolveH.Handle)

	srv := &http.Server{
		Addr:              cfg.Addr,
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
