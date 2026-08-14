# Konteks Project Handover & Architecture Document

## 1. Overview & Architecture Summary

**Konteks** is a high-performance **Context-as-a-Service (CaaS)** platform that powers AI agents with zero-latency stateful context assembly, vector memory recall, long-form core markdown memory artifacts, and cold storage room lifecycle management.

### Key Architectural Model
- **User -> Has Projects -> Project Has Rooms & Core Memory Artifacts**: Rooms, short-term memories, and markdown memory artifacts belong directly to a **Project**, not a raw user or global tenant.
- **Project Members & Invitations**: Fully prepared for team collaboration via `project_members` (roles: `owner`, `admin`, `member`) and `project_invitations` tables.
- **Dual Authentication System**:
  1. **User JWT Authentication** (`Authorization: Bearer <jwt_token>`): Used by the Web Dashboard. Claims contain `user_id`, `email`, and an array of authorized `projects: ["uuid-1", ...]`. Requests scope queries to a target project via the `X-Project-ID` header with **0ms database authorization latency**.
  2. **Project API Key Authentication** (`Authorization: Bearer ck_live_...` or `X-API-Key: ...`): Used by backend SDKs and AI agents. Directly scopes requests to the key's bound `project_id`.

---

## 2. Test Credentials & Quick Start

### Pre-seeded Admin Account & Test Data
- **Web Dashboard URL**: `http://localhost:5173/login` (or `/projects`)
- **Email**: `admin@admin.com`
- **Password**: `admin123`
- **Pre-seeded Demo Project**: `Demo AI Assistant Project` (`b0000000-0000-0000-0000-000000000001`)
- **Pre-seeded Test API Key**: `ck_live_dev_test_key_123`

---

## 3. Database Consolidation & Schema

Consolidated schema migrations in `backend/db/migrations/`:
- **`01_init.sql`**: Authoritative core schema (`users`, `projects`, `project_members`, `api_keys`, `rooms`, `messages`, `memories` with `pgvector`, `room_state`, `room_events`, `usage_records`, `context_evaluations`).
- **`02_add_encryption_fields.sql`**: Adds optional `is_encrypted BOOLEAN DEFAULT FALSE` and `encryption_metadata JSONB DEFAULT NULL` fields to `messages`, `room_state`, and `memories`.
- **`03_add_evaluation_encryption_fields.sql`**: Adds encryption fields to `context_evaluations`.
- **`04_add_room_description_and_parent.sql`**: Adds `description` and `parent_room_id` to `rooms` table for multi-agent sub-room hierarchies.
- **`05_add_artifacts.sql`**: Adds `artifact_groups`, `artifacts` (with `s3_key`), `artifact_chunks` (with HNSW vector index), `artifact_references` (Knowledge Graph Edges), and `room_artifact_blacklists`.

---

## 4. Artifact & Core Memory Infrastructure (.md)

Modern AI agents rely on durable, structured Markdown memory files (e.g. system rules, `.cursorrules`/`CLAUDE.md`, API specs, reference docs).

### Key Core Memory Pillars
1. **S3 Object Storage File Persistence**: Raw `.md` files are stored in S3 (`s3_key: "projects/{project_id}/artifacts/{artifact_id}.md"`). PostgreSQL stores metadata, `s3_key`, section headings, `pgvector` HNSW embeddings, and 1-hop graph edges.
2. **Header-Aware Chunking & Async Batch Indexing**: Markdown files are chunked by section headers (`#`, `##`, `###`). Embeddings are generated in bulk batch calls asynchronously, keeping the upload API non-blocking (**< 15ms API response**).
3. **Syntax-Based Graph Links**: Automatically parses standard markdown links (`[Auth Spec](docs/auth.md)`) and wikilinks (`[[docs/auth.md]]`) to build 1-hop knowledge graph edges (`artifact_references`).
4. **Room Memory Blacklisting**: Rooms maintain permanent room blacklists in DB + turn-level `blacklist_override` payload arrays. Excluded artifacts cost **0 tokens** and **0 LLM latency**.
5. **Policy-Driven Predictable Latency Tiers**:
   - **Fast Mode** (`include_artifacts: "false"`, default): Completely skips artifact search (**< 5-10ms**).
   - **Auto Mode** (`include_artifacts: "auto"`): Fast regex heuristic + vector similarity threshold gating (`> 0.78`) without LLM overhead (**~10-12ms**).
   - **Deep Artifact Mode** (`include_artifacts: "true"`): Deep vector search + 1-hop reference expansion (**~25-45ms**).

---

## 5. Web Dashboard & Routing Architecture

Built with **React 19**, **React Router v7**, and **TailwindCSS** adhering to the Corporate Trust design system (Indigo `#4F46E5`, Violet `#7C3AED`, Slate `#F8FAFC`).

### Route Structure
1. `/login`: Modern sign-in form pre-filled with test credentials.
2. `/register`: New user registration.
3. `/projects`: Intermediary **Projects Hub** page.
4. `/dashboard`: **Overview Tab** (real-time usage metrics, active rooms summary, evaluations).
5. `/dashboard/artifacts`: **Memory Artifacts Tab** featuring:
   - **Tab 1: Folder File Explorer & Table**: List project `.md` files, group tags, chunk counts.
   - **Tab 2: Interactive 2D Knowledge Graph Visualizer**: Built with SVG canvas (`ArtifactsGraphView.tsx`) mapping nodes, directed reference edges, and click-to-inspect drawers.
   - **Tab 3: Short Fact Memories Inspector**: Search extracted user facts.
   - **In-Dashboard Markdown Editor**: Edit `.md` content directly in the browser with automatic S3 and `pgvector` re-indexing (`PUT /v1/projects/:id/artifacts/:artifact_id`).
6. `/dashboard/logs`: **Logs Tab** (Main Logs & Context Intelligence Dashboard featuring evaluation metrics and dual-view log streams).
7. `/dashboard/logs/:roomId`: **Dedicated Log Inspection Page** featuring the new **"Memory Blacklist"** exclusion panel for managing room memory blacklists.
8. `/dashboard/evaluations/:evalId`: **Dedicated Context Evaluation Inspection Page**.
9. `/dashboard/api-keys`: **API Keys Tab**.
10. `/dashboard/settings`: **Settings Tab**.

---

## 6. How to Run, Build & Verify

### 1. Database Migrations (PostgreSQL Docker)
Ensure PostgreSQL with `pgvector` is running (e.g. via Docker):
```bash
# Docker PostgreSQL container
docker run -d --name contextkit-postgres \
  -e POSTGRES_DB=contextkit \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  pgvector/pgvector:pg16
```
Migrations `01_init.sql` through `05_add_artifacts.sql` apply automatically on backend server startup!

### 2. Backend Server (Go)
```bash
cd backend
go test -v ./...       # Run all backend unit tests
go run cmd/server/main.go
```

### 3. Frontend Web Dashboard (React + Vite)
```bash
cd frontend
npm run build       # Build production bundle
npm run dev         # Start local dev server at http://localhost:5173
```
