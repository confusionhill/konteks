# E2E Testing Plan Specification: Context Health & Zero-Trust E2EE Sidecar

This document specifies the end-to-end (E2E) testing framework for validating ContextKit's **Context Health Engine**, **Zero-Trust E2EE Sidecar Proxy**, **Vector Rotation**, and **Fact Supersedence DAG**.

---

## 1. Test Architecture & Harness Flow

```mermaid
sequenceDiagram
    autonumber
    participant Harness as E2E Test Suite (example/e2e_test_suite.py)
    participant Sidecar as Python Sidecar Proxy (:8085)
    participant Server as ContextKit Go Backend (:8080)
    participant DB as PostgreSQL pgvector

    Note over Harness,Server: Scenario 1: Policy & Entropy Sanitization
    Harness->>Server: PUT /v1/projects/:id/context-health-policy
    Server-->>Harness: 200 OK (Policy Updated)
    Harness->>Server: POST /v1/rooms/:id/messages (High Entropy Gibberish)
    Server-->>Harness: 201 Created (Flagged in DB)

    Note over Harness,Sidecar: Scenario 2: Zero-Trust E2EE & Vector Rotation
    Harness->>Sidecar: POST /v1/rooms/:id/messages (Plaintext)
    Sidecar->>Server: POST /v1/rooms/:id/messages (Base64 AES-256-GCM Container)
    Server->>DB: Store enc_... payload + Rotated Vector (R * v)
    Harness->>Server: GET /v1/rooms/:id/messages (Direct Raw Inspection)
    Server-->>Harness: Verify zero plaintext stored on server!
    Harness->>Sidecar: POST /v1/rooms/:id/context (Via Proxy)
    Sidecar-->>Harness: Return decrypted context package

    Note over Harness,Server: Scenario 3: Fact Supersedence DAG & Sweep
    Harness->>Sidecar: Fact A: "User lives in Tokyo"
    Harness->>Sidecar: Fact B: "User moved to San Francisco"
    Harness->>Server: POST /v1/rooms/:id/context-health/clean (Micro-Sweep)
    Server-->>Harness: 202 Accepted (DAG Updated: Fact A -> Superseded by Fact B)
    Harness->>Sidecar: POST /v1/rooms/:id/context
    Sidecar-->>Harness: Verify context contains ONLY active Fact B (San Francisco)!

    Note over Harness,Server: Scenario 5: Fact Restoration & Audit Revisions
    Harness->>Server: POST /v1/memories/:mem_id/restore
    Server-->>Harness: 200 OK (Status restored to active, memory_revisions logged)
```

---

## 2. Test Scenarios Matrix

| Scenario | Objective | Target Endpoint | Success Criteria |
| :--- | :--- | :--- | :--- |
| **1. Entropy Sanitization** | Verify high-entropy noise flagging & technical whitelist bypass | `POST /v1/rooms/:id/messages` | High-entropy text flagged; code/API keys whitelisted |
| **2. Zero-Trust E2EE** | Verify zero plaintext on server DB & transparent proxy decryption | `http://localhost:8085` vs `http://localhost:8080` | `enc_...` payload in DB; decrypted text in proxy response |
| **3. Supersedence DAG** | Verify LLM EAV contradiction resolution supersedes outdated facts | `POST /v1/rooms/:id/context-health/clean` | Fact A status = `superseded`; active query returns only Fact B |
| **4. Temporal Recall** | Verify deterministic relative time query bounds (*"8 months ago"*) | `POST /v1/rooms/:id/context` | Temporal parser extracts bounds & returns historical facts |
| **5. Memory Restoration** | Verify manual memory restoration & audit log creation | `POST /v1/memories/:id/restore` | Status transitions to `active`; `memory_revisions` logged |

---

## 3. Running the E2E Test Suite

### Step 1: Start ContextKit Platform Services
```bash
docker compose up -d --build
```

### Step 2: Start Python Sidecar Proxy
```bash
./venv/bin/python sidecar/proxy.py
```

### Step 3: Run E2E Test Suite
```bash
./venv/bin/python example/e2e_test_suite.py
```
