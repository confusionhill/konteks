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

## 3. Database Consolidation & Schema

Consolidated schema migrations in `backend/db/migrations/`:
- **`01_init.sql`**: Authoritative core schema (`users`, `projects`, `project_members`, `api_keys`, `rooms`, `messages`, `memories` with `pgvector`, `room_state`, `room_events`, `usage_records`, `context_evaluations`).
- **`02_add_encryption_fields.sql`**: Adds optional `is_encrypted BOOLEAN DEFAULT FALSE` and `encryption_metadata JSONB DEFAULT NULL` fields to `messages`, `room_state`, and `memories`.

---

## 4. Web Dashboard & Routing Architecture

Built with **React 19**, **React Router v7**, and **TailwindCSS** adhering to the Corporate Trust design system (Indigo `#4F46E5`, Violet `#7C3AED`, Slate `#F8FAFC`).

### Route Structure
1. `/login`: Modern sign-in form pre-filled with test credentials.
2. `/register`: New user registration (creates user identity; redirects to project creation).
3. `/projects`: Intermediary **Projects Hub** page where users pick an existing project workspace or create new projects.
4. `/dashboard`: **Overview Tab** (real-time usage metrics, active rooms summary, evaluations).
5. `/dashboard/logs`: **Logs Tab** (Main Logs & Context Intelligence Dashboard featuring top 3 evaluation metric cards `AVG RELEVANCE`, `AVG COMPLETENESS`, `TOKEN EFFICIENCY`, dual-view switcher for project-wide message logs stream and context evaluations log stream).
6. `/dashboard/logs/:roomId`: **Dedicated Log Inspection Page** (live message stream, JSON state editor, presigned archive download).
7. `/dashboard/evaluations/:evalId`: **Dedicated Context Evaluation Inspection Page** (full query text, score progress bars, AI reasoning breakdown, measured context snapshot JSON tree).
8. `/dashboard/api-keys`: **API Keys Tab** (generate project secret keys, set expiration 7/30/90 days or lifetime, copy key modal, revoke key).
9. `/dashboard/settings`: **Settings Tab** (Project UUID copy, safe soft delete project button).

---

## 5. Security, End-to-End Encryption & Privacy Masking

- **Stateless 0ms DB Check**: The JWT token carries the list of authorized `projects` in its signed payload. When a user sends `X-Project-ID`, the middleware validates it against the token's claims instantly.
- **Optional Payload Encryption**:
  - API supports `"is_encrypted": true` and `"encryption_metadata": {"alg": "AES-256-GCM", "key_version": "v1", "nonce": "..."}`.
  - Plaintext is **never** saved to disk/DB for encrypted messages.
  - Ephemeral RAM processing runs state extraction and saves re-encrypted state blobs to PostgreSQL.
- **Dashboard Privacy Masking**:
  - Encrypted log items display a privacy badge: `🔒 Encrypted Message Payload (AES-256-GCM)`.
  - Interactive **"Reveal / Decrypt"** button decrypts ciphertexts in-memory using native Web Crypto API (`window.crypto.subtle.decrypt`).

---

## 6. How to Run & Verify Locally

### 1. Backend Server (Go)
```bash
cd backend
go test ./...       # Run unit tests
go run cmd/server/main.go
```

### 2. Python Sample App (`example/app.py`)
```bash
python example/app.py
```

### 3. Frontend Web Dashboard (React + Vite)
```bash
cd frontend
npm run build       # Build production bundle
npm run dev         # Start local dev server at http://localhost:5173
```
