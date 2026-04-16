package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Fa1ry7a1l/url-shortener/internal/auth"
	"github.com/Fa1ry7a1l/url-shortener/internal/config"
	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
	appLogger "github.com/Fa1ry7a1l/url-shortener/internal/logger"
	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
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

	store, Close, err := initStorage(cfg)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to initialize storage, e=%s", err.Error()))
	}
	defer Close()

	idgen := service.NewRandomID(cfg.IDLength)
	svc := service.NewShortener(store, idgen, cfg.BaseURL)

	shortenH := handler.NewShortenHandler(svc)
	shortenJSONH := handler.NewShortenJSONHandler(svc)
	shortenBatchH := handler.NewShortenBatchHandler(svc)
	resolveH := handler.NewResolveHandler(svc)
	userURLsH := handler.NewUserURLsHandler(svc)
	pingH := handler.NewPingHandler(store)
	authManager := auth.NewManager(cfg.AuthSecret, auth.DefaultTTL)

	router := handler.NewRouter(
		shortenH.Handle,
		shortenJSONH.Handle,
		shortenBatchH.Handle,
		resolveH.Handle,
		userURLsH.Handle,
		pingH.Handle,
		log,
		authManager,
	)
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

func initStorage(cfg *config.Config) (repository.Store, func(), error) {
	if cfg.DatabaseDSN != "" {
		pg, err := repository.NewPostgresStore(cfg.DatabaseDSN)
		if err != nil {
			return nil, func() {}, err
		}

		return pg, func() {
			_ = pg.Close()
		}, nil
	}
	if cfg.FileStoragePath != "" {
		fs, err := repository.NewFileStore(cfg.FileStoragePath)
		if err != nil {
			return nil, nil, err
		}

		return fs, func() {}, nil
	}
	ms := repository.NewMemStore()
	return ms, func() {}, nil
}
