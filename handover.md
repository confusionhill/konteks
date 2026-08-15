# Konteks Project Handover & Architecture Document

## 1. Overview & Architecture Summary

**Konteks** is a high-performance **Context-as-a-Service (CaaS)** platform that powers AI agents with zero-latency stateful context assembly, vector memory recall, long-form core markdown memory artifacts, cold storage room lifecycle management, and an automated **Context Health Engine** for noise sanitization and stale fact lifecycle management.

### Key Architectural Model
- **User -> Has Projects -> Project Has Rooms, Core Memory Artifacts & Context Health Policies**: Rooms, short-term memories, core markdown artifacts, and context health policies belong directly to a **Project**, not a raw user or global tenant.
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
- **Live Test API Key**: `ck_live_3383e91ba399b71fb614c8b239a7629b`

---

## 3. Database Consolidation & Schema

Consolidated schema migrations in `backend/db/migrations/`:
- **`01_init.sql`**: Authoritative core schema (`users`, `projects`, `project_members`, `api_keys`, `rooms`, `messages`, `memories` with `pgvector`, `room_state`, `room_events`, `usage_records`, `context_evaluations`).
- **`02_add_encryption_fields.sql`**: Adds optional `is_encrypted BOOLEAN DEFAULT FALSE` and `encryption_metadata JSONB DEFAULT NULL` fields to `messages`, `room_state`, and `memories`.
- **`03_add_evaluation_encryption_fields.sql`**: Adds encryption fields to `context_evaluations`.
- **`04_add_room_description_and_parent.sql`**: Adds `description` and `parent_room_id` to `rooms` table for multi-agent sub-room hierarchies.
- **`05_add_artifacts.sql`**: Adds `artifact_groups`, `artifacts` (with `s3_key`), `artifact_chunks` (with HNSW vector index), `artifact_references` (Knowledge Graph Edges), and `room_artifact_blacklists`.
- **`07_add_context_health_settings.sql`**: Adds `context_health_policy` JSONB setting to `projects`, extends `memories` table with `status`, `superseded_by_id`, `shannon_entropy`, `is_whitelisted`, and creates `memory_revisions` for audit logging.

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

## 5. Context Health Engine (Pollution Mitigation & Stale Knowledge Lifecycle)

The **Context Health Engine** sanitizes memory noise, manages fact versioning, and resolves conflicting user state over extended 1-year+ room timelines.

### Key Context Health Pillars
1. **Fast-Path Deterministic Ingress Gate (`service/entropy.go`)**:
   - Calculates Shannon entropy for prose text.
   - Applies regex technical whitelisting for code blocks, API keys (`sk-...`, `ck_live_...`), UUIDs, Base64 strings, and CJK text to prevent false-positive entropy rejections.
2. **Version-Linked Supersedence DAG (`usecase/context_health_usecase.go`)**:
   - Runs LLM Entity-Attribute-Value (EAV) contradiction evaluation during room micro-sweeps ($N=50$ turns).
   - Outdated facts are updated to `status = 'superseded'` and linked to the new fact via `superseded_by_id`.
   - Immutable audit logs are stored in `memory_revisions`.
3. **Tiered SLA Context Retrieval**:
   - **Standard Active Query SLA**: $\le 25\text{ms}$ p95 (`WHERE status = 'active'`).
   - **Temporal Historical Query SLA**: $\le 250\text{ms}$ p95 (Deterministic temporal parser extracts relative expressions like *"8 months ago"* and traverses cold historical partitions).
4. **Distance-Preserving Vector Pseudonymization (DPVP)**:
   - Mathematically honest security model for encrypted rotated vectors ($\mathbf{R} \cdot \mathbf{v}$) with bucketed metadata (`token_length` ranges: `0-128`, `129-512`, `513-2048`).
5. **Zero Redis Infrastructure**:
   - Runs natively on Docker Compose (`docker-compose.yml`) via PostgreSQL 15 `pgvector:pg15`, Go `chan messageTask` worker queues, and goroutines. Redis migration specs for multi-replica scaling are detailed in `backlog/cache.md`.

---

## 6. Client Sidecar SDK & Zero-Trust Proxy (`sdk/python/` & `sidecar/`)

The **Client Sidecar SDK** runs in the customer's VPC, providing zero-trust E2EE envelope encryption, vector matrix rotation, and transparent HTTP proxying.

### Key Sidecar SDK Components
1. **Python Package (`sdk/python/contextkit_sdk/`)**:
   - `crypto.py`: HKDF key derivation ($\mathbf{K}_{room} = \text{HKDF}(\mathbf{K}_{master})$) and AES-256-GCM envelope encryption/decryption.
   - `vector_rotation.py`: Orthonormal Matrix generator ($\mathbf{R} \in \mathbb{R}^{d \times d}$) to transform embeddings ($\mathbf{v}_{enc} = \mathbf{R} \cdot \mathbf{v}$) while preserving exact cosine distance in `pgvector`.
   - `health_worker.py`: Client E2EE background worker that decrypts candidate memories in client memory, evaluates contradictions via local/private LLMs (Ollama, vLLM, private OpenAI endpoints), and submits signed supersedence directives.
   - `client.py` & `setup.py`: High-level Python package (`pip install contextkit-sdk`).
2. **Transparent Sidecar Proxy Container (`sidecar/`)**:
   - `sidecar/proxy.py`: AsyncIO / FastAPI HTTP proxy listening at `http://localhost:8085`.
   - `Dockerfile` & `docker-compose.sidecar.yml`: Containerized proxy allowing legacy AI agents to communicate with ContextKit using standard API calls while the sidecar handles encryption, vector rotation, and context health micro-sweeps invisibly.
3. **Outbound-Only VPC Connectivity**:
   - Requires **0 open inbound ports** in client VPC firewalls. All communication uses outbound HTTPS (Port 443 TLS 1.3).
   - Project-scoped connection multiplexing ($O(1)$ connection per project) prevents connection storms across millions of rooms.

---

## 7. Web Dashboard & Routing Architecture

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
7. `/dashboard/rooms/:roomId`: **Dedicated Log & Room Inspection Page** featuring:
   - **Memory Blacklist Exclusion Panel** (`RoomBlacklistPanel.tsx`).
   - **Context Memory Audit & Supersedence Panel** (`RoomMemoriesPanel.tsx`) with status badges (`active`, `superseded`, `flagged`), entropy scores, and a **Restore Fact** button.
8. `/dashboard/evaluations/:evalId`: **Dedicated Context Evaluation Inspection Page**.
9. `/dashboard/api-keys`: **API Keys Tab**.
10. `/dashboard/settings`: **Project Settings Tab** featuring the **Context Health Engine Policy Card** (Master toggle, mode selectors `balanced`/`strict`/`cost_optimized`/`off`, capability warning banners, recency decay slider, turn eval interval, max entropy threshold, and Save Policy button).

---

## 8. How to Run, Build & Verify

### 1. Full Docker Compose Deployment (Recommended)
```bash
# Build backend and start PostgreSQL pgvector + MinIO + Go server
docker compose up -d --build

# Verify container logs & migration 07 status
docker logs konteks-app --tail 30
```

### 2. Backend Server Unit & Integration Tests (Go)
```bash
cd backend
go test -v ./...       # Run all backend unit tests (0.585s)
```

### 3. Client Python SDK Unit Tests (Python)
```bash
./venv/bin/python -m unittest discover -s sdk/python/tests -p "test_*.py"
```

### 4. Frontend Web Dashboard (React + Vite)
```bash
cd frontend
npx tsc --noEmit    # Verify zero TypeScript compilation errors
npm run build       # Build production bundle
npm run dev         # Start local dev server at http://localhost:5173
```
