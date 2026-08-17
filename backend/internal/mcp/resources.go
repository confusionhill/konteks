package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/dika/contextkit/backend/internal/domain"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mserver "github.com/mark3labs/mcp-go/server"
)

// RegisterResources exposes ContextKit rooms as dynamic MCP resources. URIs:
//
//	/contextkit/room/{room_id}            -> room + state
//	/contextkit/room/{room_id}/context    -> assembled context package
//	/contextkit/room/{room_id}/messages   -> message history
//	/contextkit/room/{room_id}/memories   -> memory index
func RegisterResources(s *mserver.MCPServer, d toolDeps) {
	s.AddResourceTemplate(
		mcpgo.NewResourceTemplate("/contextkit/room/{room_id}", "Room"),
		func(ctx context.Context, req mcpgo.ReadResourceRequest) ([]mcpgo.ResourceContents, error) {
			auth, err := d.resolveAuth(ctx)
			if err != nil {
				return nil, err
			}
			roomID, ok := roomIDFromResourceURI(req.Params.URI)
			if !ok {
				return nil, errors.New("invalid resource URI: missing room_id")
			}
			room, state, err := d.roomService.GetRoom(ctx, auth, roomID)
			if err != nil {
				return nil, err
			}
			return jsonResourceContents(req.Params.URI, map[string]any{"room": room, "state": state})
		},
	)

	s.AddResourceTemplate(
		mcpgo.NewResourceTemplate("/contextkit/room/{room_id}/context", "Room Context"),
		func(ctx context.Context, req mcpgo.ReadResourceRequest) ([]mcpgo.ResourceContents, error) {
			auth, err := d.resolveAuth(ctx)
			if err != nil {
				return nil, err
			}
			roomID, ok := roomIDFromResourceURI(req.Params.URI)
			if !ok {
				return nil, errors.New("invalid resource URI: missing room_id")
			}
			pkg, err := d.contextService.GetContextPackage(ctx, auth, roomID, domain.GetContextRequest{})
			if err != nil {
				return nil, err
			}
			return jsonResourceContents(req.Params.URI, pkg)
		},
	)

	s.AddResourceTemplate(
		mcpgo.NewResourceTemplate("/contextkit/room/{room_id}/messages", "Room Messages"),
		func(ctx context.Context, req mcpgo.ReadResourceRequest) ([]mcpgo.ResourceContents, error) {
			auth, err := d.resolveAuth(ctx)
			if err != nil {
				return nil, err
			}
			roomID, ok := roomIDFromResourceURI(req.Params.URI)
			if !ok {
				return nil, errors.New("invalid resource URI: missing room_id")
			}
			messages, total, err := d.contextService.ListMessages(ctx, auth, roomID, 200, 0)
			if err != nil {
				return nil, err
			}
			return jsonResourceContents(req.Params.URI, map[string]any{"total": total, "messages": messages})
		},
	)

	s.AddResourceTemplate(
		mcpgo.NewResourceTemplate("/contextkit/room/{room_id}/memories", "Room Memories"),
		func(ctx context.Context, req mcpgo.ReadResourceRequest) ([]mcpgo.ResourceContents, error) {
			auth, err := d.resolveAuth(ctx)
			if err != nil {
				return nil, err
			}
			roomID, ok := roomIDFromResourceURI(req.Params.URI)
			if !ok {
				return nil, errors.New("invalid resource URI: missing room_id")
			}
			memories, total, err := d.memoryService.ListMemories(ctx, auth, roomID, 200, 0)
			if err != nil {
				return nil, err
			}
			return jsonResourceContents(req.Params.URI, map[string]any{"total": total, "memories": memories})
		},
	)
}

func roomIDFromResourceURI(uri string) (string, bool) {
	trimmed := strings.TrimPrefix(uri, "/contextkit/room/")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" || strings.Contains(trimmed, "/") {
		return "", false
	}
	return trimmed, true
}

func jsonResourceContents(uri string, data any) ([]mcpgo.ResourceContents, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}
	return []mcpgo.ResourceContents{
		mcpgo.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(bytes),
		},
	}, nil
}