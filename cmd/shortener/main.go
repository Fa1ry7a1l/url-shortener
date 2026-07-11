package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Fa1ry7a1l/url-shortener/internal/audit"
	"github.com/Fa1ry7a1l/url-shortener/internal/auth"
	"github.com/Fa1ry7a1l/url-shortener/internal/buildinfo"
	"github.com/Fa1ry7a1l/url-shortener/internal/config"
	"github.com/Fa1ry7a1l/url-shortener/internal/handler"
	appLogger "github.com/Fa1ry7a1l/url-shortener/internal/logger"
	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
)

var (
	buildVersion = buildinfo.NotAvailable
	buildDate    = buildinfo.NotAvailable
	buildCommit  = buildinfo.NotAvailable
)

const shutdownTimeout = 10 * time.Second

func main() {
	buildinfo.Print(buildinfo.Info{
		Version: buildVersion,
		Date:    buildDate,
		Commit:  buildCommit,
	})

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
	workerDone := make(chan struct{})
	defer func() {
		stopWorker()
		<-workerDone
	}()
	go func() {
		defer close(workerDone)
		svc.RunDeleteWorker(workerCtx)
	}()

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

	ln, scheme, err := listen(srv.Addr, cfg.EnableHTTPS)
	if err != nil {
		panic(err)
	}
	pprofLn, err := net.Listen("tcp", pprofSrv.Addr)
	if err != nil {
		_ = ln.Close()
		panic(err)
	}

	serveErrCh := make(chan error, 2)
	fmt.Printf("pprof listening on http://%s/debug/pprof/\n", pprofSrv.Addr)
	serveAsync(serveErrCh, "pprof", pprofSrv, pprofLn)
	fmt.Printf("listening on %s://%s\n", scheme, srv.Addr)
	serveAsync(serveErrCh, "main", srv, ln)

	signalCtx, stopSignals := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)
	defer stopSignals()

	select {
	case <-signalCtx.Done():
		stopSignals()
		log.Info("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := shutdownServers(shutdownCtx, srv, pprofSrv); err != nil {
			_ = srv.Close()
			_ = pprofSrv.Close()
			panic(err)
		}
		if err := waitServers(serveErrCh, 2); err != nil {
			panic(err)
		}
	case serveErr := <-serveErrCh:
		_ = srv.Close()
		_ = pprofSrv.Close()
		if serveErr != nil {
			panic(serveErr)
		}
		if err := waitServers(serveErrCh, 1); err != nil {
			panic(err)
		}
	}
}

func serveAsync(errCh chan<- error, name string, srv *http.Server, ln net.Listener) {
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("%s server: %w", name, err)
			return
		}
		errCh <- nil
	}()
}

func shutdownServers(ctx context.Context, servers ...*http.Server) error {
	errCh := make(chan error, len(servers))
	for _, srv := range servers {
		go func(s *http.Server) {
			errCh <- s.Shutdown(ctx)
		}(srv)
	}

	var errs []error
	for range servers {
		if err := <-errCh; err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func waitServers(errCh <-chan error, count int) error {
	var errs []error
	for i := 0; i < count; i++ {
		if err := <-errCh; err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func listen(addr string, enableHTTPS bool) (net.Listener, string, error) {
	if !enableHTTPS {
		ln, err := net.Listen("tcp", addr)
		return ln, "http", err
	}

	tlsConfig, err := newTLSConfig(addr)
	if err != nil {
		return nil, "", err
	}
	ln, err := tls.Listen("tcp", addr, tlsConfig)
	return ln, "https", err
}

func newTLSConfig(addr string) (*tls.Config, error) {
	cert, err := newSelfSignedCertificate(addr)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}, nil
}

func newSelfSignedCertificate(addr string) (tls.Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return tls.Certificate{}, err
	}

	certTemplate := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	for _, host := range certificateHosts(addr) {
		if ip := net.ParseIP(host); ip != nil {
			certTemplate.IPAddresses = append(certTemplate.IPAddresses, ip)
			continue
		}
		certTemplate.DNSNames = append(certTemplate.DNSNames, host)
	}

	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&certTemplate,
		&certTemplate,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		return tls.Certificate{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}

func certificateHosts(addr string) []string {
	hosts := []string{"localhost", "127.0.0.1", "::1"}
	host, _, err := net.SplitHostPort(addr)
	if err == nil && host != "" {
		hosts = append(hosts, host)
	}

	seen := make(map[string]struct{}, len(hosts))
	uniqueHosts := hosts[:0]
	for _, host := range hosts {
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		uniqueHosts = append(uniqueHosts, host)
	}

	return uniqueHosts
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
