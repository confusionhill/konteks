# ContextKit: Current Features & Architecture Comparison

This document maps out the current features, system architecture, API schemas, and flow diagrams of ContextKit. It compares the actual codebase implementation against the original product requirements defined in [ai_agent_context_infrastructure_product(2).md](file:///Users/dika/Documents/github/contextkit/ai_agent_context_infrastructure_product(2).md).

---

## 1. System Architecture Diagram

ContextKit is designed as a **Room-First, Model-Agnostic Context Layer** that sits between client agents and LLM providers. 

```mermaid
graph TD
    Client[Client Agent Application] -->|1. Create Room / Message / Context| API[ContextKit API - Echo Go]
    
    subgraph ContextKit API Server
        API -->|Read/Write Hot State| DB[(PostgreSQL + pgvector)]
        API -->|Publish Events| PubSub[PubSub Event Broker]
        PubSub -->|Stream Log Events| SSE[Server-Sent Events Endpoint]
        
        API -->|Enqueue Task| TaskQueue[Bounded Task Channel - 5000]
        TaskQueue -->|Process Asynchronously| Workers[Compaction Workers Pool - 5]
        
        Workers -->|Generate Embeddings| LLM[OpenRouter Client]
        Workers -->|Extract Topic, Summary, State| LLM
        Workers -->|Update State & LastSummarizedSeq| DB
        
        API <-->|Lazy-Hydrate Rooms| MinIO[(MinIO S3 Cold Storage)]
    end
    
    subgraph Cron / CLI Archiver
        CLI[Standalone CLI Archiver] -->|1. List Inactive Rooms| DB
        CLI -->|2. Write compressed JSON.gz| MinIO
        CLI -->|3. Purge Messages & Memories| DB
    end
    
    API -->|Query/Log Usage| Metering[Usage & Billing Service]
    Metering -->|Update stats| DB
    
    Dashboard[React Developer Dashboard] -->|Query Stats / Rooms| API
```

---

## 2. Feature Mapping & PRD Alignment

Here is the breakdown of how the current codebase aligns with the original PRD roadmap (MVP / Must-Have vs. Nice-to-Have):

| Feature Area | PRD Must-Have Requirement | Actual Implementation | Alignment Status |
| :--- | :--- | :--- | :--- |
| **Rooms API** | Room creation, state retrieval, room deletion | Exposes `/v1/rooms` (Create, Retrieve State, Delete) with raw schema and default policy settings. | **Fully Aligned** |
| **Messages API** | Log messages (role/content), fetch message logs | Exposes `POST` and `GET` `/v1/rooms/:id/messages` with cursor pagination. | **Fully Aligned** |
| **Semantic Memory** | pgvector integration, cosine similarity search | Implemented Cosine similarity search in `SearchSemantic` using Postgres `pgvector` (`<=>`). | **Fully Aligned** |
| **Asynchronous Compaction** | Summarization, active topic, state extraction | Background worker pool (5 goroutines, 5k queue) extracts summary, topic, and strict JSON schemas. | **Fully Aligned** |
| **SSE / Event Streaming** | Streaming conversation events | `/v1/rooms/:id/events` streams log and custom developer events via Echo server-sent events. | **Fully Aligned** |
| **Multi-Tenancy** | API key authentication, tenancy boundary checks | Tenant and API Key tables in Postgres. Hashed token check in API header middleware. | **Fully Aligned** |
| **Usage Metering** | Track API calls, usage records, and tokens | `UsageTrackingMiddleware` tracks API calls, input/output tokens, and cost. | **Fully Aligned** |
| **Context Policies** | Token budgets, strategies, time-freshness decay | Implemented token-packing algorithm, `MaxHistoryDepth`, and freshness weight decay (`pgvector` scores). | **Fully Aligned (Phase 2)** |
| **Reference Resolution** | Solve ambiguous terms (*"it"*, *"this"*) | Implemented OpenRouter synchronous reference resolver on hot-path retrieves. | **Fully Aligned (Phase 2)** |
| **Developer Dashboard** | Stat metrics, rooms list, cost tracking | React frontend. Features cost totals, paginated/sortable rooms tables, search, and infinite message scroll. | **Fully Aligned (Phase 2)** |

---

## 3. Key Deviations & Product Enhancements

During implementation, a few design decisions were refined to optimize latency, cost controls, and developer ergonomics beyond the original PRD scope:

### A. Smart-Sync Reference Resolution (Heuristic Hot-Path)
* **Original PRD Plan:** Resolve references for every context call.
* **Our Deviation:** Resolving references on every call causes 1.5s–3s of API latency. We introduced a **Smart-Sync Heuristic** (`hasPronouns`) that scans user queries for terms like *"it"*, *"this"*, *"which"*, or *"them"*. 
* **Impact:** If no pronouns are detected, reference resolution is skipped on the hot path, maintaining **sub-100ms** context retrievals.

### B. Dual Budget Pricing Model Pivot
* **Original PRD Plan:** Basic token pricing / compute credit concepts.
* **Our Deviation:** Abstract compute credits are confusing to bill. We pivoted to a **Base Operations + Tokens Processed** billing model:
  * **Operations:** $0.01 per 1,000 requests.
  * **Tokens Processed:** $0.07 per 1,000,000 tokens (history scanned & ranked by our engine).
  * **Tokens Delivered:** **100% Free** (incentivizing developers to configure strict `TokenBudget` cost ceilings on downstream LLMs).
* **Impact:** Direct visual alignment in the dashboard stats, tracking "Avg Tokens Processed" vs "Avg Tokens Delivered".

### C. Dynamic Schema Extraction
* **Original PRD Plan:** Fixed schema extraction at the room level.
* **Our Deviation:** Adding a `DynamicSchema` override parameter in the `POST /context` payload.
* **Impact:** Allows agents to pass a temporary, ad-hoc JSON schema on a per-request basis, overriding the room's default schema for advanced workflow steps.

### D. Three-Tier Memory & Cold Storage Archival
* **Original PRD Plan:** Standard relational database storage for all messages and states indefinitely.
* **Our Deviation:** We implemented a sophisticated Three-Tier Memory architecture (Hot / Cold / Hydrating). Inactive rooms are automatically compressed into `.json.gz` files by our Archiver CLI, uploaded to S3-compatible cold storage (MinIO), and purged from PostgreSQL to minimize storage costs.
* **Impact:** Achieves up to a **98% storage cost reduction** for inactive data. Upon receiving a new context request, a dual-stage **Lazy Hydration** process dynamically streams the archive back into the hot database, restoring the room seamlessly without agent intervention. Additionally, the Dashboard allows direct `.json.gz` downloads via presigned S3 URLs to prevent unnecessary database load.

**Room State Lifecycle:**
```mermaid
stateDiagram-v2
    [*] --> active: Room Created
    
    active --> archiving: CLI Archiver Sweeps Inactive Room
    archiving --> archived: Zip to S3 & Purge from Postgres
    
    archived --> hydrating: New API Request (Lazy Hydrate)
    hydrating --> active: Stage 1 Hydration Complete (Fast)
    
    note right of hydrating
      Stage 2 (Background)
      backfills old messages
    end note
```

---

## 4. End-to-End Context Assembly Flow

This sequence diagram details the exact logic executed when a client requests context via the API:

```mermaid
sequenceDiagram
    autonumber
    actor Agent as Client Agent
    participant API as ContextKit API
    participant DB as PostgreSQL/pgvector
    participant S3 as MinIO Cold Storage
    participant LLM as OpenRouter Service
    
    Agent->>API: POST /v1/rooms/{id}/context {query, policy}
    API->>DB: Fetch Room Details, Status & Default Policy
    DB-->>API: Return Room Details
    
    alt Room Status is Archived (Lazy Hydration)
        API->>DB: Set status to hydrating
        API->>S3: GET room_archive.json.gz
        S3-->>API: Return archive bytes
        API->>API: Decompress & Parse state + last messages
        API->>DB: Hydrate RoomState & last 3 messages (Stage 1)
        API->>DB: Update room status to active
        Note over API, DB: Spawn background task to backfill remaining history (Stage 2)
    end
    
    API->>DB: Fetch RoomState (Summary, active_topic, last_summarized_seq)
    DB-->>API: Return RoomState
    
    %% Policy merging
    Note over API: Merge Policy parameters<br/>(TokenBudget, MaxHistoryDepth, FreshnessWeight)
    
    %% Heuristic Reference Resolution
    alt Query contains pronouns ("it", "this", "them")
        API->>DB: Fetch last 15 messages
        DB-->>API: Return messages
        API->>LLM: ResolveReferences(query, history, state)
        LLM-->>API: Return resolved reference key-value map & entities
    else No pronouns detected
        Note over API: Skip Reference Resolution<br/>(Sub-100ms Retrieval)
    end
    
    %% Vector Semantic Search
    API->>LLM: Generate query embedding
    LLM-->>API: Return embedding vector
    API->>DB: Semantic search (pgvector) with Freshness Decay Weight
    DB-->>API: Return ranked memory chunks
    
    %% Pending Delta Sliding Window Fetching
    API->>DB: Fetch messages WHERE sequence > last_summarized_seq (Pending Delta)
    DB-->>API: Return pending messages
    
    %% Greedy Token Packing
    Note over API: Greedy Token Packing:<br/>1. Pack Summary & State first<br/>2. Pack dynamic msg limit (max strategy or pending)<br/>3. Pack memories from highest to lowest score<br/>4. Ceil budget at TokenBudget limit
    
    %% Metering
    API->>DB: Log usage record (Operation + Scanned Tokens)
    API-->>Agent: Return Context Package
```

---

## 5. Future Roadmap & Gap Analysis

Based on the [future-impl.md](file:///Users/dika/Documents/github/contextkit/future-impl.md) notes, the following sections remain to transition ContextKit from MVP/Refinement to **Enterprise Platform Scale (Phase 3)**:

1. **Persistent Task Queuing:** Shift from in-memory Go channel worker pools (`chan messageTask`) to Redis-backed queues (`hibiken/asynq`) or Database Outbox Patterns to guarantee durability across restarts.
2. **Cold Storage Chunking & Streaming Optimization:** While our cold storage archiver is now active for MVP, future versions will process giant rooms (1M+ messages) using strict streaming and S3 Multipart Uploads to ensure fixed memory constraints during archival.
3. **Multi-Provider Memory Routing:** Add adapters to route vector memories to Honcho, Redis AMS, or Zep instead of storing all embeddings directly in PostgreSQL/pgvector.
4. **Context Quality Scoring:** Programmatic evaluation of context delivery usefulness (observability analytics).
