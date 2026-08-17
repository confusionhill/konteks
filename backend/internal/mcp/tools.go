package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/dika/contextkit/backend/internal/service"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mserver "github.com/mark3labs/mcp-go/server"
)

// toolDeps wires the fresh repository/service layers into the MCP tools.
type toolDeps struct {
	auth           *AuthProvider
	defaultAuth    domain.AuthContext
	roomService    *service.MCPRoomService
	contextService *service.MCPContextService
	memoryService  *service.MCPMemoryService
}

// resolveAuth returns the per-request auth context, falling back to the
// startup-resolved default (stdio mode) where no request scoping exists.
func (d toolDeps) resolveAuth(ctx context.Context) (domain.AuthContext, error) {
	if err, ok := AuthErrorFromContext(ctx); ok {
		return domain.AuthContext{}, fmt.Errorf("authentication failed: %w", err)
	}
	if auth, ok := AuthFromContext(ctx); ok {
		return auth, nil
	}
	if d.defaultAuth.Valid() {
		return d.defaultAuth, nil
	}
	return domain.AuthContext{}, errors.New("authentication required: provide an API key via CONTEXTKIT_API_KEY or the Authorization header")
}

func toolResult(data any) (*mcpgo.CallToolResult, error) {
	return mcpgo.NewToolResultJSON(data)
}

func toolErrorf(format string, a ...any) *mcpgo.CallToolResult {
	return mcpgo.NewToolResultError(fmt.Sprintf(format, a...))
}

// asMap converts an arbitrary JSON argument value into a map, or nil.
func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// asStringSlice converts an argument into a []string, tolerating nil.
func asStringSlice(v any) []string {
	arr, _ := v.([]any)
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func parsePolicy(args map[string]any) domain.ContextPolicy {
	policy := domain.ContextPolicy{}
	if raw, ok := args["policy"]; ok && raw != nil {
		if fields, isObj := raw.(map[string]any); isObj {
			if v, ok := fields["strategy"].(string); ok {
				policy.Strategy = v
			}
			if v, ok := fields["max_tokens"].(float64); ok {
				policy.MaxTokens = int(v)
			}
			if v, ok := fields["max_history_depth"].(float64); ok {
				policy.MaxHistoryDepth = int(v)
			}
			if v, ok := fields["freshness_weight"].(float64); ok {
				policy.FreshnessWeight = v
			}
			if v, ok := fields["min_relevance_score"].(float64); ok {
				policy.MinRelevanceScore = v
			}
			if v, ok := fields["fast_mode"].(bool); ok {
				policy.FastMode = v
			}
		}
	}
	return policy
}

func RegisterTools(s *mserver.MCPServer, d toolDeps) {
	s.AddTool(mcpgo.NewTool(
		"create_room",
		mcpgo.WithDescription("Create a new persistent conversation room for an external user. Pass the agent's or customer's external user identifier (not an internal user id)."),
		mcpgo.WithString("external_user_id",
			mcpgo.Description("The agent or customer identifier the room belongs to (optional; defaults to the authenticated project)"),
		),
		mcpgo.WithString("description",
			mcpgo.Description("Human-readable description or purpose of the room"),
		),
		mcpgo.WithObject("state_schema",
			mcpgo.Description("Optional JSON schema that constrains the room state object, e.g. {\"budget\": \"number\"}"),
		),
		mcpgo.WithObject("default_policy",
			mcpgo.Description("Default context policy: {strategy, max_tokens, max_history_depth, freshness_weight, min_relevance_score}"),
		),
		mcpgo.WithString("parent_room_id",
			mcpgo.Description("Parent room ID for hierarchical agent orchestration"),
		),
		mcpgo.WithObject("initial_state",
			mcpgo.Description("Optional initial JSON state object"),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		args := req.GetArguments()
		externalUserID := req.GetString("external_user_id", "")
		description := req.GetString("description", "")

		var schemaJSON json.RawMessage
		if v := args["state_schema"]; v != nil {
			schemaJSON, _ = json.Marshal(v)
		}
		var policyJSON json.RawMessage
		var policy *domain.ContextPolicy
		if v := args["default_policy"]; v != nil {
			policyJSON, _ = json.Marshal(v)
			p := parsePolicy(args)
			policy = &p
		}
		var initialStateJSON json.RawMessage
		if v := args["initial_state"]; v != nil {
			initialStateJSON, _ = json.Marshal(v)
		}
		parentID := req.GetString("parent_room_id", "")
		var parentPtr *string
		if parentID != "" {
			parentPtr = &parentID
		}
		_ = policyJSON

		room, err := d.roomService.CreateRoom(ctx, auth, externalUserID, description, parentPtr, schemaJSON, policy, initialStateJSON)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(room)
	})

	s.AddTool(mcpgo.NewTool(
		"update_room",
		mcpgo.WithDescription("Update a room's metadata, parent, state schema, or default context policy."),
		mcpgo.WithString("room_id",
			mcpgo.Description("The room ID to update"),
			mcpgo.Required(),
		),
		mcpgo.WithString("description",
			mcpgo.Description("New room description"),
		),
		mcpgo.WithString("parent_room_id",
			mcpgo.Description("New parent room ID (empty string clears it)"),
		),
		mcpgo.WithObject("state_schema",
			mcpgo.Description("New room state JSON schema"),
		),
		mcpgo.WithObject("default_policy",
			mcpgo.Description("New default context policy object"),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		args := req.GetArguments()
		roomID := req.GetString("room_id", "")

		var descPtr *string
		if v, ok := args["description"]; ok && v != nil {
			s := v.(string)
			descPtr = &s
		}
		var parentPtr *string
		if v, ok := args["parent_room_id"]; ok && v != nil {
			s := v.(string)
			parentPtr = &s
		}
		var schemaJSON json.RawMessage
		if v := args["state_schema"]; v != nil {
			schemaJSON, _ = json.Marshal(v)
		}
		var policy *domain.ContextPolicy
		if v := args["default_policy"]; v != nil {
			p := parsePolicy(args)
			policy = &p
		}

		room, err := d.roomService.UpdateRoom(ctx, auth, roomID, descPtr, parentPtr, schemaJSON, policy)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(room)
	})

	s.AddTool(mcpgo.NewTool(
		"get_room",
		mcpgo.WithDescription("Get a room with its full state (summary, active topic, state object, references, entities)."),
		mcpgo.WithString("room_id",
			mcpgo.Description("The room ID to fetch"),
			mcpgo.Required(),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		roomID := req.GetString("room_id", "")
		room, state, err := d.roomService.GetRoom(ctx, auth, roomID)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(map[string]any{
			"room":  room,
			"state": state,
		})
	})

	s.AddTool(mcpgo.NewTool(
		"list_rooms",
		mcpgo.WithDescription("List rooms in the project, optionally filtered by external user, with pagination."),
		mcpgo.WithString("external_user_id",
			mcpgo.Description("Only return rooms owned by this external user"),
		),
		mcpgo.WithInteger("limit",
			mcpgo.Description("Maximum number of rooms to return"),
			mcpgo.DefaultNumber(20),
		),
		mcpgo.WithInteger("offset",
			mcpgo.Description("Number of rooms to skip for pagination"),
			mcpgo.DefaultNumber(0),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		externalUserID := req.GetString("external_user_id", "")
		limit := req.GetInt("limit", 20)
		offset := req.GetInt("offset", 0)

		rooms, total, err := d.roomService.ListRooms(ctx, auth, externalUserID, limit, offset)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(map[string]any{
			"total": total,
			"limit": limit,
			"offset": offset,
			"rooms": rooms,
		})
	})

	s.AddTool(mcpgo.NewTool(
		"get_room_state",
		mcpgo.WithDescription("Get a room's extracted state package: summary, active topic, state object, entities, and references."),
		mcpgo.WithString("room_id",
			mcpgo.Description("The room ID to fetch state for"),
			mcpgo.Required(),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		roomID := req.GetString("room_id", "")
		state, err := d.roomService.GetRoomState(ctx, auth, roomID)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(state)
	})

	s.AddTool(mcpgo.NewTool(
		"update_room_state",
		mcpgo.WithDescription("Patch a room's state object with key-value updates and/or key deletions."),
		mcpgo.WithString("room_id",
			mcpgo.Description("The room ID whose state to update"),
			mcpgo.Required(),
		),
		mcpgo.WithObject("state_updates",
			mcpgo.Description("Flat JSON object of key-value pairs to set on the room state"),
			mcpgo.Required(),
		),
		mcpgo.WithArray("deletes",
			mcpgo.Description("Optional list of state keys to remove"),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		roomID := req.GetString("room_id", "")
		updates := asMap(req.GetArguments()["state_updates"])
		if len(updates) == 0 {
			return toolErrorf("state_updates must be a non-empty object"), nil
		}
		deletes := asStringSlice(req.GetArguments()["deletes"])

		state, err := d.roomService.UpdateRoomState(ctx, auth, roomID, updates, deletes)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(state)
	})

	s.AddTool(mcpgo.NewTool(
		"archive_room",
		mcpgo.WithDescription("Mark a room as archived. Archived rooms retain a hot cache of their latest state and 3 most recent messages."),
		mcpgo.WithString("room_id",
			mcpgo.Description("The room ID to archive"),
			mcpgo.Required(),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		roomID := req.GetString("room_id", "")
		room, err := d.roomService.ArchiveRoom(ctx, auth, roomID)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(room)
	})

	s.AddTool(mcpgo.NewTool(
		"add_message",
		mcpgo.WithDescription("Append a message to a room's conversation history. Encryption is optional and purely client-side."),
		mcpgo.WithString("room_id",
			mcpgo.Description("The room ID to append to"),
			mcpgo.Required(),
		),
		mcpgo.WithString("role",
			mcpgo.Description("Message role"),
			mcpgo.Enum("user", "assistant", "system", "tool"),
			mcpgo.Required(),
		),
		mcpgo.WithString("content",
			mcpgo.Description("Message content"),
			mcpgo.Required(),
		),
		mcpgo.WithBoolean("is_encrypted",
			mcpgo.Description("Whether the content is client-encrypted (server stores it verbatim)"),
			mcpgo.DefaultBool(false),
		),
		mcpgo.WithString("encryption_metadata",
			mcpgo.Description("Optional JSON metadata describing the encryption scheme (alg, key version, nonce)"),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		roomID := req.GetString("room_id", "")
		role := req.GetString("role", "")
		content := req.GetString("content", "")
		isEncrypted := req.GetBool("is_encrypted", false)

		var encMeta *string
		if v := req.GetString("encryption_metadata", ""); v != "" {
			encMeta = &v
		}

		msg, err := d.contextService.AddMessage(ctx, auth, roomID, role, content, isEncrypted, encMeta)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(map[string]any{
			"message_id": msg.ID,
			"sequence":   msg.Sequence,
			"created_at": msg.CreatedAt,
		})
	})

	s.AddTool(mcpgo.NewTool(
		"get_messages",
		mcpgo.WithDescription("List a room's message history in chronological order with pagination."),
		mcpgo.WithString("room_id",
			mcpgo.Description("The room ID whose messages to fetch"),
			mcpgo.Required(),
		),
		mcpgo.WithInteger("limit",
			mcpgo.Description("Maximum number of messages to return"),
			mcpgo.DefaultNumber(50),
		),
		mcpgo.WithInteger("offset",
			mcpgo.Description("Number of messages to skip for pagination"),
			mcpgo.DefaultNumber(0),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		roomID := req.GetString("room_id", "")
		limit := req.GetInt("limit", 50)
		offset := req.GetInt("offset", 0)

		messages, total, err := d.contextService.ListMessages(ctx, auth, roomID, limit, offset)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(map[string]any{
			"total":    total,
			"limit":    limit,
			"offset":   offset,
			"messages": messages,
		})
	})

	s.AddTool(mcpgo.NewTool(
		"get_context_package",
		mcpgo.WithDescription("Assemble a token-budgeted context package for a room: summary, active topic, state, recent messages, relevant memories. Use before answering your agent to ground your response."),
		mcpgo.WithString("room_id",
			mcpgo.Description("The room ID to assemble context for"),
			mcpgo.Required(),
		),
		mcpgo.WithString("query",
			mcpgo.Description("The current user query, used to recall semantically relevant memories"),
		),
		mcpgo.WithObject("policy",
			mcpgo.Description("Context policy: {strategy: auto|recent_heavy|memory_heavy|hybrid, max_tokens, max_history_depth, freshness_weight, fast_mode}"),
		),
		mcpgo.WithObject("dynamic_schema",
			mcpgo.Description("Optional JSON schema to re-extract room state on the fly"),
		),
		mcpgo.WithBoolean("is_encrypted",
			mcpgo.Description("Mark the assembled package as client-encrypted (skips server-side LLM enrichment)"),
			mcpgo.DefaultBool(false),
		),
		mcpgo.WithBoolean("include_children",
			mcpgo.Description("Include child agent room summaries in the package"),
			mcpgo.DefaultBool(false),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		args := req.GetArguments()
		roomID := req.GetString("room_id", "")
		query := req.GetString("query", "")
		isEncrypted := req.GetBool("is_encrypted", false)
		includeChildren := req.GetBool("include_children", false)
		policy := parsePolicy(args)

		pkg, err := d.contextService.GetContextPackage(ctx, auth, roomID, domain.GetContextRequest{
			Query:           query,
			Policy:          policy,
			IsEncrypted:     isEncrypted,
			IncludeChildren: includeChildren,
		})
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(pkg)
	})

	s.AddTool(mcpgo.NewTool(
		"add_memory",
		mcpgo.WithDescription("Persist a long-term memory for a room. The embedding is generated asynchronously in the background, so the call returns immediately."),
		mcpgo.WithString("room_id",
			mcpgo.Description("The room ID to attach the memory to"),
			mcpgo.Required(),
		),
		mcpgo.WithString("content",
			mcpgo.Description("The factual content to remember (e.g. 'The user prefers window seats')"),
			mcpgo.Required(),
		),
		mcpgo.WithObject("metadata",
			mcpgo.Description("Optional JSON metadata, e.g. {\"type\": \"preference\"}"),
		),
		mcpgo.WithString("external_user_id",
			mcpgo.Description("Optional external user scope; defaults to the room owner"),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		args := req.GetArguments()
		roomID := req.GetString("room_id", "")
		content := req.GetString("content", "")

		var metadataJSON json.RawMessage
		if v := args["metadata"]; v != nil {
			metadataJSON, _ = json.Marshal(v)
		}
		extUser := req.GetString("external_user_id", "")
		var extPtr *string
		if extUser != "" {
			extPtr = &extUser
		}

		mem, err := d.memoryService.AddMemory(ctx, auth, roomID, content, metadataJSON, extPtr)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(map[string]any{
			"memory_id": mem.ID,
			"status":    "embedding_pending",
		})
	})

	s.AddTool(mcpgo.NewTool(
		"search_memories",
		mcpgo.WithDescription("Semantically search a room's memories using cosine similarity on the query's embedding."),
		mcpgo.WithString("room_id",
			mcpgo.Description("The room ID to search within"),
			mcpgo.Required(),
		),
		mcpgo.WithString("query",
			mcpgo.Description("Natural language query to match against memory content"),
			mcpgo.Required(),
		),
		mcpgo.WithInteger("limit",
			mcpgo.Description("Maximum number of memories to return"),
			mcpgo.DefaultNumber(10),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		roomID := req.GetString("room_id", "")
		query := req.GetString("query", "")
		limit := req.GetInt("limit", 10)

		memories, err := d.memoryService.SearchMemories(ctx, auth, roomID, query, limit)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(map[string]any{
			"memories": memories,
		})
	})

	s.AddTool(mcpgo.NewTool(
		"list_memories",
		mcpgo.WithDescription("List all memories for a room with pagination."),
		mcpgo.WithString("room_id",
			mcpgo.Description("The room ID whose memories to list"),
			mcpgo.Required(),
		),
		mcpgo.WithInteger("limit",
			mcpgo.Description("Maximum number of memories to return"),
			mcpgo.DefaultNumber(20),
		),
		mcpgo.WithInteger("offset",
			mcpgo.Description("Number of memories to skip for pagination"),
			mcpgo.DefaultNumber(0),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		roomID := req.GetString("room_id", "")
		limit := req.GetInt("limit", 20)
		offset := req.GetInt("offset", 0)

		memories, total, err := d.memoryService.ListMemories(ctx, auth, roomID, limit, offset)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(map[string]any{
			"total":    total,
			"limit":    limit,
			"offset":   offset,
			"memories": memories,
		})
	})

	s.AddTool(mcpgo.NewTool(
		"remove_memory",
		mcpgo.WithDescription("Permanently delete a memory by ID."),
		mcpgo.WithString("memory_id",
			mcpgo.Description("The memory ID to delete"),
			mcpgo.Required(),
		),
	), func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		auth, err := d.resolveAuth(ctx)
		if err != nil {
			return toolErrorf("%v", err), nil
		}
		memoryID := req.GetString("memory_id", "")
		if err := d.memoryService.RemoveMemory(ctx, auth, memoryID); err != nil {
			return toolErrorf("%v", err), nil
		}
		return toolResult(map[string]any{
			"deleted": true,
			"memory_id": memoryID,
		})
	})
}