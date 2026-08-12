# Konteks Backend Architecture

This document details the architecture, end-to-end data flow, and context generation process of the `konteks` Context-as-a-Service (CaaS) layer.

---

## 1. High-Level System Architecture

Konteks acts as the stateful memory and context layer underneath your AI Agent. The diagram below illustrates the components of the Konteks backend and how they interact:

```mermaid
graph TD
    Client[Client App / Flask UI] <-->|HTTP REST| API[Echo REST API Gateway]
    
    subgraph Konteks Backend
        API <-->|Read/Write| DB[(PostgreSQL + pgvector)]
        API -.->|Push messageTask| Queue[Buffered Task Channel]
        
        subgraph Concurrency Worker Pool
            Queue --> Worker1[Worker 1]
            Queue --> Worker2[Worker 2]
            Queue --> Worker3[Worker 3]
            Queue --> Worker4[Worker 4]
            Queue --> Worker5[Worker 5]
        end
    end
    
    subgraph External Services
        Worker1 & Worker2 & Worker3 & Worker4 & Worker5 <-->|HTTP API| LLM[OpenRouter / LLM Provider]
        API <-->|Query Embedding| LLM
    end
    
    classDef storage fill:#1e293b,stroke:#475569,stroke-width:2px,color:#fff;
    classDef component fill:#1d4ed8,stroke:#3b82f6,stroke-width:2px,color:#fff;
    classDef worker fill:#581c87,stroke:#8b5cf6,stroke-width:2px,color:#fff;
    classDef external fill:#0f172a,stroke:#334155,stroke-width:2px,color:#94a3b8;
    
    class DB storage;
    class API,Queue component;
    class Worker1,Worker2,Worker3,Worker4,Worker5 worker;
    class LLM external;
```

---

## 2. End-to-End Chat Turn Flowchart

The flowchart below demonstrates the sequence of events during a single user-agent turn, separating the **synchronous critical path** (for low latency) from the **asynchronous state extraction path** (managed by the worker pool):

```mermaid
flowchart TD
    %% Turn Start
    User[User Sends Message] --> MsgSync[1. Synchronous Message Ingestion]
    
    subgraph Sync Path [Synchronous Path - Latency < 20ms]
        MsgSync --> WriteDB[Write Message to PG DB]
        WriteDB --> PushQueue[Push Task to Buffered Channel]
        PushQueue --> ReturnHTTP[Return Success HTTP 201]
    end
    
    ReturnHTTP --> GetCtx[2. Context Retrieval]
    
    subgraph Context Gen [Context Retrieval - Latency ~800ms]
        GetCtx --> FetchHistory[Fetch Last 10 Messages]
        GetCtx --> FetchState[Fetch Current RoomState]
        GetCtx --> GenQueryEmbed[Generate Query Embedding]
        GenQueryEmbed --> MatchMemories[Semantic Memory Vector Search]
        MatchMemories --> CompilePkg[Assemble Context Package JSON]
    end
    
    CompilePkg --> AgentRun[3. Agent Reasoning Loop]
    
    subgraph Agent Loop [Client Runtime]
        AgentRun --> LLMReason[LLM Agent call with Context Package]
        LLMReason --> ToolRun[Execute Travel Booking Tools]
        ToolRun --> LLMResponse[Generate Final Assistant Message]
    end
    
    LLMResponse --> MsgAssist[4. Save Assistant Response]
    
    subgraph Assistant Save [Synchronous Path - Latency < 20ms]
        MsgAssist --> WriteDB2[Write Assistant Message to DB]
        WriteDB2 --> PushQueue2[Push Task to Buffered Channel]
        PushQueue2 --> RenderUI[Render Response in Web UI]
    end
    
    %% Background Async
    subgraph Async Pool [Background Worker Pool - Rate Limited]
        PushQueue -.->|Buffered Queue| Workers[Idle Workers 1-5]
        PushQueue2 -.->|Buffered Queue| Workers
        Workers --> GenMsgEmbed[Generate Message Embedding]
        GenMsgEmbed --> SaveMemory[Save Semantic Memory to Postgres]
        SaveMemory --> ExtractState[LLM State & Summary Extraction]
        ExtractState --> UpsertState[Upsert RoomState in DB]
    end

    classDef sync fill:#065f46,stroke:#10b981,stroke-width:2px,color:#fff;
    classDef context fill:#1e3a8a,stroke:#3b82f6,stroke-width:2px,color:#fff;
    classDef client fill:#701a75,stroke:#d946ef,stroke-width:2px,color:#fff;
    classDef async fill:#581c87,stroke:#a855f7,stroke-width:2px,color:#fff;
    
    class MsgSync,WriteDB,PushQueue,ReturnHTTP,MsgAssist,WriteDB2 sync;
    class GetCtx,FetchHistory,FetchState,GenQueryEmbed,MatchMemories,CompilePkg context;
    class AgentRun,LLMReason,ToolRun,LLMResponse,RenderUI client;
    class Workers,GenMsgEmbed,SaveMemory,ExtractState,UpsertState async;
```

---

## 3. How the Context Package is Generated

When the agent client calls the `POST /v1/rooms/:room_id/context` API endpoint, the Konteks backend compiles the minimum sufficient context required for the agent to complete the current conversation turn. 

Here is the detailed step-by-step logic executed inside `contextUsecase.GetContextPackage`:

```mermaid
stateDiagram-v2
    [*] --> VerifyRoom : 1. Auth & Room Verification
    VerifyRoom --> FetchRoomState : 2. Read RoomState
    FetchRoomState --> ListHistory : 3. Fetch Recent Messages
    ListHistory --> CheckQuery : 4. Inspect Query Parameter
    
    state CheckQuery {
        [*] --> EmptyQuery : Query is ""
        EmptyQuery --> Assemble : Skip vector match
        
        [*] --> HasQuery : Query is present
        HasQuery --> QueryEmbed : Call Embeddings API
        QueryEmbed --> VectorMatch : Search pgvector index
        VectorMatch --> Assemble : Select top 5 memories
    }
    
    Assemble --> ReturnPkg : 5. Compile JSON Package
    ReturnPkg --> [*]
```

### Detailed Breakdown of Steps:

#### Step 1: Authorization & Verification
The endpoint verifies the API Token via standard Authorization headers and checks that the requested `room_id` belongs to the authenticated client tenant (`tenant_id`).

#### Step 2: Fetch Room State
The system reads the `room_state` table for the room. It retrieves:
*   **Active Topic**: E.g. `"hotel comparison"`.
*   **Summary**: A continuously updated running summary of the conversation.
*   **State Variables**: A flat JSONB object containing parameters gathered so far (e.g. `{"destination": "Tokyo", "budget": 150}`).

#### Step 3: Fetch Message History
The server queries the `messages` table for the last 10 messages associated with this room. This provides the LLM with the most recent local conversation context.

#### Step 4: Semantic Memory Search (Optional)
If a `query` string (typically the user's message) is passed to the context endpoint:
1.  **Generate Query Embedding**: Konteks calls the embedding model (`qwen/qwen3-embedding-8b`) via OpenRouter to generate the query's vector representation.
2.  **pgvector Matching**: It runs a cosine-similarity distance search in Postgres against the `memories` table:
    ```sql
    SELECT content FROM memories 
    WHERE room_id = $1 OR external_user_id = $2 
    ORDER BY embedding <=> $3 LIMIT 5;
    ```
    This pulls relevant historical user preferences (e.g. *"User prefers hotels with a pool"* or *"Paris budget was $100"*).

#### Step 5: Assembly
The backend packages the `Summary`, `ActiveTopic`, `StateVariables`, `RelevantMessages`, and `Memories` into the standard `ContextPackage` response format and returns it to the client.

---

## 4. Background Asynchronous Worker Loop

The background Worker Pool controls resource usage and prevents rate-limit issues.

### Concurrency Regulation
*   **Task Queue**: The `taskQueue` is a buffered channel in memory with a size limit of 5,000. It absorbs incoming write bursts.
*   **Fixed Workers**: A pool of exactly 5 workers are spawned on startup. Each worker runs in an infinite loop pulling tasks from the channel:
    ```go
    func (u *messageUsecase) worker() {
        for task := range u.taskQueue {
            u.processBackgroundUpdate(task)
        }
    }
    ```
*   **OpenRouter Concurrency Capping**: Because there are only 5 workers, the system makes a maximum of 5 concurrent HTTP requests to OpenRouter for embedding generation and state extraction, keeping our API credentials safe from rate-limit bans (TPM/RPM limits).
