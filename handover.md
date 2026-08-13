# Konteks Project Handover & Architecture Document

## 1. Overview & Architecture Summary

**Konteks** is a high-performance **Context-as-a-Service (CaaS)** platform that powers AI agents with zero-latency stateful context assembly, vector memory recall, and cold storage room lifecycle management.

### Key Architectural Model
- **User -> Has Projects -> Project Has Rooms**: Rooms and memories belong directly to a **Project**, not a raw user or global tenant.
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
- **Pre-seeded Demo Project**: `Demo AI Assistant Project` (`550e8400-e29b-41d4-a716-446655440000`)
- **Pre-seeded Test API Key**: `ck_live_dev_test_key_123`

---

## 3. Database Consolidation & Schema (`01_init.sql`)

All legacy incremental migrations (`000001` to `000007`) have been consolidated into a single authoritative migration script:
- **Location**: [01_init.sql](file:///Users/dika/Documents/github/contextkit/backend/db/migrations/01_init.sql)

### Core DDL Highlights
- **`users`**: User identity and bcrypt password hashes.
- **`projects`**: Project metadata with `owner_id` and `deleted_at TIMESTAMP WITH TIME ZONE` for **safe soft-deletion**.
- **`project_members` & `project_invitations`**: Multi-user team permissions and pending invite tracking.
- **`api_keys`**: Cryptographic SHA-256 key hash storage (`key_hash`), prefix display (`key_prefix`), optional `expires_at`, and `revoked_at`.
- **`rooms`**: Bound to `project_id`. Statuses: `active`, `archiving`, `archived`, `hydrating`.
- **`messages`**: Chronological transcript with `role VARCHAR(50)` supporting custom user roles.
- **`memories`**: PostgreSQL `pgvector(4096)` vector embeddings scoped by `project_id`.
- **`room_state`**, **`room_events`**, **`usage_records`**, **`context_evaluations`**: Real-time state tracking, SSE streaming, usage metering, and automated context quality evaluations.

---

## 4. Web Dashboard & Routing Architecture

Built with **React 19**, **React Router v7**, and **TailwindCSS** adhering to the Corporate Trust design system (Indigo `#4F46E5`, Violet `#7C3AED`, Slate `#F8FAFC`).

### Route Structure
1. `/login`: Modern sign-in form pre-filled with test credentials.
2. `/register`: New user registration (creates user identity; redirects to project creation).
3. `/projects`: Intermediary **Projects Hub** page where users pick an existing project workspace or create new projects.
4. `/dashboard`: **Overview Tab** (real-time usage metrics, active rooms summary, evaluations).
5. `/dashboard/rooms`: **Rooms Tab** (room list, search, status filter, pagination).
6. `/dashboard/rooms/:roomId`: **Room Detail Page** (live message stream, JSON state editor, presigned archive download).
7. `/dashboard/api-keys`: **API Keys Tab** (generate project secret keys, set expiration 7/30/90 days or lifetime, copy key modal, revoke key).
8. `/dashboard/settings`: **Settings Tab** (Project UUID copy, safe soft delete project button).

---

## 5. Security & BOLA Prevention

- **Stateless 0ms DB Check**: The JWT token carries the list of authorized `projects` in its signed payload. When a user sends `X-Project-ID`, the middleware validates it against the token's claims instantly.
- **Fallback Verification**: If a newly joined project is not yet in an older JWT token, the middleware seamlessly validates membership in PostgreSQL via `authRepo.IsUserProjectMemberOrOwner`.
- **Key Hashing**: Raw secret API keys (`ck_live_...`) are shown **ONLY ONCE** upon creation and are stored securely as SHA-256 hashes (`key_hash`).

---

## 6. How to Run & Verify locally

### 1. Backend Server (Go)
```bash
cd backend
go test ./...       # Run all unit tests
go run cmd/server/main.go
```

### 2. Frontend Web Dashboard (React + Vite)
```bash
cd frontend
npm run build       # Build production bundle
npm run dev         # Start local dev server at http://localhost:5173
```
