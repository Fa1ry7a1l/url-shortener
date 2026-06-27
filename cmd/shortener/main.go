package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Fa1ry7a1l/url-shortener/internal/audit"
	"github.com/Fa1ry7a1l/url-shortener/internal/auth"
	"github.com/Fa1ry7a1l/url-shortener/internal/config"
	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
	appLogger "github.com/Fa1ry7a1l/url-shortener/internal/logger"
	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()

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
	workerCtx, stopWorker := context.WithCancel(context.Background())
	defer stopWorker()
	go svc.RunDeleteWorker(workerCtx)

	auditor := initAuditor(cfg)
	shortenH := handler.NewShortenHandler(svc, auditor)
	shortenJSONH := handler.NewShortenJSONHandler(svc, auditor)
	shortenBatchH := handler.NewShortenBatchHandler(svc)
	resolveH := handler.NewResolveHandler(svc, auditor)
	userURLsH := handler.NewUserURLsHandler(svc)
	deleteUserURLsH := handler.NewDeleteUserURLsHandler(svc)
	pingH := handler.NewPingHandler(store)
	authManager := auth.NewManager(cfg.AuthSecret, auth.DefaultTTL)

	router := handler.NewRouter(
		shortenH.Handle,
		shortenJSONH.Handle,
		shortenBatchH.Handle,
		resolveH.Handle,
		userURLsH.Handle,
		deleteUserURLsH.Handle,
		pingH.Handle,
		log,
		authManager,
	)
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	pprofSrv := &http.Server{
		Addr:              cfg.PprofAddr,
		Handler:           handler.NewPprofHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		panic(err)
	}
	pprofLn, err := net.Listen("tcp", pprofSrv.Addr)
	if err != nil {
		_ = ln.Close()
		panic(err)
	}

	go func() {
		fmt.Printf("pprof listening on http://%s/debug/pprof/\n", pprofSrv.Addr)
		if serveErr := pprofSrv.Serve(pprofLn); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			panic(serveErr)
		}
	}()

	fmt.Printf("listening on http://%s\n", srv.Addr)
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func printBuildInfo() {
	fmt.Printf(
		"Build version: %s\nBuild date: %s\nBuild commit: %s\n",
		valueOrNA(buildVersion),
		valueOrNA(buildDate),
		valueOrNA(buildCommit),
	)
}

func valueOrNA(value string) string {
	if value == "" {
		return "N/A"
	}

	return value
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

func initAuditor(cfg *config.Config) *audit.Subject {
	auditor := audit.NewSubject()
	if cfg.AuditFile != "" {
		auditor.Attach(audit.NewFileObserver(cfg.AuditFile))
	}
	if cfg.AuditURL != "" {
		auditor.Attach(audit.NewHTTPObserver(cfg.AuditURL, nil))
	}

	return auditor
}
