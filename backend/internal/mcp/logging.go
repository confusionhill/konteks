package mcp

import (
	"context"
	"log/slog"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// loggerKey carries the slog.Logger into the request context so tool and
// resource handlers can emit scoped log records.
type loggerKey struct{}

// WithRequestLogger attaches the server logger to a request context.
func WithRequestLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// RequestLoggerFromContext returns the request-scoped logger, if any.
func RequestLoggerFromContext(ctx context.Context) (*slog.Logger, bool) {
	l, ok := ctx.Value(loggerKey{}).(*slog.Logger)
	return l, ok && l != nil
}

// logToolMiddleware emits one structured log line per tool invocation with the
// authenticated project scope, outcome, and duration. It complements the
// built-in mcp-go request logger (mcp.method / mcp.tool.name / outcome /
// duration_s) with the project that owns the call.
func logToolMiddleware(logger *slog.Logger, defaultAuth domain.AuthContext) server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			start := time.Now()

			projectID := ""
			if auth, ok := AuthFromContext(ctx); ok {
				projectID = auth.ProjectID
			} else if defaultAuth.Valid() {
				projectID = defaultAuth.ProjectID
			}

			result, handlerErr := next(ctx, req)
			duration := time.Since(start).Seconds()

			outcome := "ok"
			var errMsg string
			switch {
			case handlerErr != nil:
				outcome = "error"
				errMsg = handlerErr.Error()
			case result != nil && result.IsError:
				outcome = "error_result"
				if text := toolResultText(result); text != "" {
					errMsg = truncate(text, 500)
				}
			}

			attrs := []any{
				slog.String("tool", req.Params.Name),
				slog.String("project_id", projectID),
				slog.String("outcome", outcome),
				slog.Float64("duration_s", duration),
			}
			if errMsg != "" {
				attrs = append(attrs, slog.String("error", errMsg))
			}

			logger.Log(ctx, slog.LevelInfo, "mcp.tool.scoped", attrs...)
			return result, handlerErr
		}
	}
}

// toolResultText extracts the first text content payload, if any.
func toolResultText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}