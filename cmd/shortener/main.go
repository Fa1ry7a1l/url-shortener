package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Fa1ry7a1l/url-shortener/internal/config"
	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
	appLogger "github.com/Fa1ry7a1l/url-shortener/internal/logger"
	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		panic(err)
	}

	log, err := appLogger.New()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()

	store := repository.NewMemStore()
	idgen := service.NewRandomID(cfg.IDLength)
	svc := service.NewShortener(store, idgen, cfg.BaseURL)

	shortenH := handler.NewShortenHandler(svc)
	resolveH := handler.NewResolveHandler(svc)
	router := handler.NewRouter(shortenH.Handle, resolveH.Handle, log)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		panic(err)
	}

	log.Info("server started",
		zap.String("addr", cfg.Addr),
		zap.String("base_url", cfg.BaseURL),
	)

	fmt.Printf("listening on http://%s\n", srv.Addr)
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
