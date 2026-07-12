package grpcserver

import (
	"context"
	"errors"
	"strings"

	shortenerpb "github.com/Fa1ry7a1l/url-shortener/api"
	"github.com/Fa1ry7a1l/url-shortener/internal/audit"
	"github.com/Fa1ry7a1l/url-shortener/internal/auth"
	"github.com/Fa1ry7a1l/url-shortener/internal/repository"
	"github.com/Fa1ry7a1l/url-shortener/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ShortenerService defines the business operations exposed over gRPC.
type ShortenerService interface {
	// Shorten validates and stores one original URL.
	Shorten(ctx context.Context, original string) (string, error)
	// Resolve returns the original URL for a short ID.
	Resolve(ctx context.Context, id string) (string, error)
	// UserURLs returns links created by the authenticated user.
	UserURLs(ctx context.Context) ([]service.UserURL, error)
}

// AuditPublisher receives audit events produced by gRPC handlers.
type AuditPublisher interface {
	// Notify publishes one audit event.
	Notify(ctx context.Context, event audit.Event) error
}

// Server exposes the shortener business service through gRPC.
type Server struct {
	shortenerpb.UnimplementedShortenerServiceServer
	svc     ShortenerService
	auditor AuditPublisher
}

// NewServer creates a gRPC shortener server.
func NewServer(svc ShortenerService, auditors ...AuditPublisher) *Server {
	s := &Server{svc: svc}
	if len(auditors) > 0 {
		s.auditor = auditors[0]
	}
	return s
}

// ShortenURL handles POST /api/shorten equivalent requests over gRPC.
func (s *Server) ShortenURL(ctx context.Context, req *shortenerpb.URLShortenRequest) (*shortenerpb.URLShortenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}

	shortURL, err := s.svc.Shorten(serviceContext(ctx), req.GetUrl())
	if err != nil {
		var conflictErr *service.ConflictError
		if errors.As(err, &conflictErr) {
			return nil, status.Error(codes.AlreadyExists, conflictErr.ShortURL)
		}
		return nil, status.Error(codes.InvalidArgument, "bad request")
	}

	publishAudit(ctx, s.auditor, audit.ActionShorten, req.GetUrl())
	return &shortenerpb.URLShortenResponse{Result: shortURL}, nil
}

// ExpandURL handles GET /{id} equivalent requests over gRPC.
func (s *Server) ExpandURL(ctx context.Context, req *shortenerpb.URLExpandRequest) (*shortenerpb.URLExpandResponse, error) {
	if req == nil || req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing id")
	}

	originalURL, err := s.svc.Resolve(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, repository.ErrDeleted) {
			return nil, status.Error(codes.FailedPrecondition, "deleted")
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "not found")
		}
		return nil, status.Error(codes.Internal, "internal server error")
	}

	publishAudit(ctx, s.auditor, audit.ActionFollow, originalURL)
	return &shortenerpb.URLExpandResponse{Result: originalURL}, nil
}

// ListUserURLs handles GET /api/user/urls equivalent requests over gRPC.
func (s *Server) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*shortenerpb.UserURLsResponse, error) {
	items, err := s.svc.UserURLs(serviceContext(ctx))
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}
		return nil, status.Error(codes.Internal, "internal server error")
	}

	resp := &shortenerpb.UserURLsResponse{
		Url: make([]*shortenerpb.URLData, 0, len(items)),
	}
	for _, item := range items {
		resp.Url = append(resp.Url, &shortenerpb.URLData{
			ShortUrl:    item.ShortURL,
			OriginalUrl: item.OriginalURL,
		})
	}
	return resp, nil
}

func serviceContext(ctx context.Context) context.Context {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return ctx
	}
	return service.ContextWithUserID(ctx, userID)
}

func publishAudit(ctx context.Context, publisher AuditPublisher, action string, originalURL string) {
	if publisher == nil {
		return
	}

	userID, _ := auth.UserIDFromContext(ctx)
	event := audit.NewEvent(action, userID, strings.TrimSpace(originalURL))
	_ = publisher.Notify(ctx, event)
}
