package mcp

import (
	"context"
	"log/slog"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/dika/contextkit/backend/internal/service"
	"github.com/dika/contextkit/backend/internal/worker"
	"github.com/mark3labs/mcp-go/server"
)

// Deps bundles the dependencies required to build the ContextKit MCP server.
type Deps struct {
	AuthProvider *AuthProvider
	// DefaultAuth is resolved at startup from CONTEXTKIT_API_KEY and used as
	// the fallback scope for transports that carry no per-request credentials
	// (stdio).
	DefaultAuth domain.AuthContext

	RoomService    *service.MCPRoomService
	ContextService *service.MCPContextService
	MemoryService  *service.MCPMemoryService

	EmbeddingWorker *worker.MCPEmbeddingWorker

	// Logger is optional. When set it is installed on the server so every
	// JSON-RPC request and tool/resource invocation emits a structured log
	// record, and is attached to request contexts for handlers to use.
	Logger *slog.Logger
}

// NewServer builds the MCP MCPServer exposing ContextKit tools and resources.
func NewServer(d Deps) *server.MCPServer {
	opts := []server.ServerOption{
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	}
	if d.Logger != nil {
		opts = append(opts,
			server.WithLogger(d.Logger),
			server.WithToolHandlerMiddleware(logToolMiddleware(d.Logger, d.DefaultAuth)),
		)
	}

	s := server.NewMCPServer("contextkit", "1.0.0", opts...)

	if d.RoomService == nil || d.ContextService == nil || d.MemoryService == nil {
		panic("contextkit mcp: service dependencies are required")
	}

	deps := toolDeps{
		auth:           d.AuthProvider,
		defaultAuth:    d.DefaultAuth,
		roomService:    d.RoomService,
		contextService: d.ContextService,
		memoryService:  d.MemoryService,
	}

	RegisterTools(s, deps)
	RegisterResources(s, deps)
	return s
}

// authKey is the context key carrying the authenticated project scope.
type authKey struct{}

// authErrorKey carries an authentication failure from the transport boundary.
type authErrorKey struct{}

// WithAuth returns a context carrying the resolved AuthContext.
func WithAuth(ctx context.Context, auth domain.AuthContext) context.Context {
	return context.WithValue(ctx, authKey{}, auth)
}

// AuthFromContext returns the AuthContext attached to this request context.
func AuthFromContext(ctx context.Context) (domain.AuthContext, bool) {
	auth, ok := ctx.Value(authKey{}).(domain.AuthContext)
	return auth, ok && auth.Valid()
}

// WithAuthError marks the request as failed authentication with the given error.
func WithAuthError(ctx context.Context, err error) context.Context {
	return context.WithValue(ctx, authErrorKey{}, err)
}

// AuthErrorFromContext returns a transport-level auth failure, if any.
func AuthErrorFromContext(ctx context.Context) (error, bool) {
	err, ok := ctx.Value(authErrorKey{}).(error)
	return err, ok && err != nil
}