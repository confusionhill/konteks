# Context Infrastructure for AI Agents

> **You build the agent. We manage its context.**

A model-agnostic, agent-agnostic infrastructure layer that manages stateful AI conversations: rooms, messages, context retrieval, memory, state, compaction, and context delivery.

---

## 1. Executive Summary

The product is a **Context Management API for AI agents**.

Developers should not have to rebuild conversation infrastructure every time they build an AI agent.

The developer owns:

- LLM provider
- Agent reasoning
- System prompt
- Tool definitions
- Tool execution
- Business logic
- External APIs and databases

The Context Infrastructure owns:

- Conversation rooms
- Message persistence
- Conversation state
- Context retrieval
- Context selection
- Context compression
- Summarization
- Reference resolution
- Optional long-term memory
- Context lifecycle
- Streaming conversation events

### Core promise

> **Give your agent exactly the context it needs for every turn.**

The product is not another agent framework and not primarily a vector database. It is the **state/context layer underneath an agent**.

---

# 2. Problem

Building a conversational AI agent looks simple:

```text
User → LLM → Response
```

In production, it becomes:

```text
User
 ↓
Message history
 ↓
Conversation state
 ↓
Relevant historical messages
 ↓
Memory
 ↓
Summaries
 ↓
Entity/reference resolution
 ↓
Context-window management
 ↓
LLM
 ↓
Tool calls
 ↓
Tool results
 ↓
LLM
 ↓
Response
 ↓
Update state/memory
```

Every AI application ends up rebuilding some version of this.

### Common problems

#### Context-window limits

Sending the entire conversation to the LLM becomes expensive and eventually impossible.

#### Irrelevant history

The last 20 messages are not necessarily the 20 messages relevant to the current question.

#### Follow-up references

Users naturally say:

- "What about the cheaper one?"
- "Does it have a pool?"
- "Move it to Friday."
- "What did we decide earlier?"
- "Use the same address."

The agent needs to understand what those references refer to.

#### Conversation state

The agent needs more than raw messages:

```text
Topic: hotel booking

Location: Bali

Budget: <$150/night

Preference: swimming pool

Selected hotel: hotel_123

Booking date: Friday
```

#### Long conversations

A conversation needs to be summarized and compacted without losing important information.

#### Agent/tool separation

The context layer should not force developers to adopt a particular agent framework or tool ecosystem.

---

# 3. Product Definition

## The core abstraction: Room

A `Room` represents one persistent conversation.

```text
Tenant
 └── User
      └── Room
           ├── Messages
           ├── State
           ├── Summary
           ├── Context
           ├── Memories
           └── Events
```

Example:

```text
Room: room_123

User:
Find me a hotel in Bali.

Agent:
Here are five options...

User:
Which one is cheapest?

Agent:
...

User:
Does it have a pool?

Agent:
...
```

The service understands that:

```text
"Which one"
    ↓
previously discussed hotels

"it"
    ↓
the relevant hotel
```

The customer should not have to implement this manually.

---

# 4. What the Product Is / Is Not

## It is

- Conversation infrastructure
- Context infrastructure
- Stateful room management
- Context retrieval
- Memory infrastructure
- Context-window optimization
- Agent context delivery

## It is not

- An LLM provider
- An agent framework
- A tool marketplace
- A business database
- A CRM
- A general-purpose vector database
- A mandatory tool execution runtime

The customer remains in control of the agent.

---

# 5. Architecture

## High-level architecture

```text
┌─────────────────────────────────────────────────────┐
│                    END USERS                        │
│                                                     │
│       WhatsApp   Web   Mobile   API   Voice        │
└───────────────────────┬─────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│              CONTEXT INFRASTRUCTURE API             │
│                                                     │
│  Rooms                                              │
│  Messages                                           │
│  Context                                            │
│  State                                              │
│  Memory                                             │
│  Events                                             │
│  Streaming                                          │
└───────────────────────┬─────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│                CONTEXT ORCHESTRATOR                 │
│                                                     │
│  Context retrieval                                  │
│  Relevance ranking                                  │
│  Reference resolution                               │
│  Summarization                                      │
│  Compaction                                          │
│  State extraction                                   │
│  Memory extraction                                  │
│  Context assembly                                   │
└───────────────┬───────────────────────┬─────────────┘
                │                       │
                ▼                       ▼
┌─────────────────────────┐   ┌────────────────────────┐
│       STORAGE           │   │   MEMORY PROVIDERS     │
│                         │   │                        │
│ PostgreSQL              │   │ Honcho                 │
│ Messages                │   │ Redis Agent Memory     │
│ Room state              │   │ Future providers       │
│ Tenant data             │   │ Native implementation  │
│ pgvector                │   │                        │
└─────────────────────────┘   └────────────────────────┘
                │
                └───────────────┬────────────────────────┘
                                ▼
┌─────────────────────────────────────────────────────┐
│                CONTEXT PACKAGE                     │
│                                                     │
│  Summary                                            │
│  Relevant history                                  │
│  Current state                                      │
│  Entities                                           │
│  References                                         │
│  Memories                                           │
│  Constraints                                        │
└───────────────────────┬─────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│                CUSTOMER'S AGENT                     │
│                                                     │
│  Their LLM                                         │
│  Their reasoning                                   │
│  Their tools                                       │
│  Their business logic                              │
└─────────────────────────────────────────────────────┘
```

---

# 6. Provider Architecture

Memory systems such as **Honcho** and **Redis Agent Memory Server** can be reused as infrastructure components.

They should not define the external product API.

```text
                    Context API
                         │
                  Context Engine
                         │
        ┌────────────────┼────────────────┐
        │                │                │
        ▼                ▼                ▼
     Honcho          Redis AMS       Native Engine
        │                │                │
        └────────────────┼────────────────┘
                         ▼
                  Context Package
```

This creates a provider abstraction.

```go
type ContextEngine interface {
    AddMessage(ctx context.Context, message Message) error

    GetContext(
        ctx context.Context,
        roomID string,
        query string,
    ) (*ContextPackage, error)

    UpdateState(
        ctx context.Context,
        roomID string,
        state RoomState,
    ) error
}
```

The product can start with existing infrastructure and gradually replace individual components with proprietary implementations where differentiation matters.

---

# 7. Core API

## Rooms

```http
POST /v1/rooms
GET /v1/rooms/:room_id
DELETE /v1/rooms/:room_id
```

## Messages

```http
POST /v1/rooms/:room_id/messages
GET /v1/rooms/:room_id/messages
```

Example:

```json
{
  "role": "user",
  "content": "Which one is cheaper?"
}
```

## Context

```http
POST /v1/rooms/:room_id/context
```

Example response:

```json
{
  "room_id": "room_123",
  "context": {
    "summary": "User is comparing hotels in Bali.",
    "active_topic": "hotel comparison",
    "state": {
      "location": "Bali",
      "budget": 150
    },
    "relevant_messages": [
      "msg_123",
      "msg_128"
    ],
    "entities": [
      {
        "type": "hotel",
        "id": "hotel_456"
      }
    ]
  }
}
```

## Events

```http
POST /v1/rooms/:room_id/events
```

Possible events:

```text
user.message
context.updated
agent.started
agent.delta
tool.started
tool.completed
agent.completed
```

---

# 8. End-to-End Flow

```text
                    USER MESSAGE
                         │
                         ▼
                 ┌──────────────┐
                 │  API Gateway │
                 └──────┬───────┘
                        │
                        ▼
                 Persist Message
                        │
                        ▼
               Load Room State
                        │
                        ▼
             Retrieve Context
                        │
          ┌─────────────┼─────────────┐
          ▼             ▼             ▼
       Recent        Semantic       Memory
       Messages      Retrieval      Retrieval
          │             │             │
          └─────────────┼─────────────┘
                        ▼
                 Context Ranking
                        │
                        ▼
               Reference Resolution
                        │
                        ▼
                 Context Assembly
                        │
                        ▼
                Context Package
                        │
                        ▼
                CUSTOMER AGENT
                        │
             ┌──────────┼──────────┐
             ▼          ▼          ▼
            LLM        Tools      APIs
             │
             ▼
             Response
             │
             ▼
       Stream to User
             │
             ▼
       Persist Response
             │
             ▼
      Async State Update
             │
       ┌─────┼──────┐
       ▼     ▼      ▼
    Summary State  Memory
    Update  Update Extraction
```

---

# 9. Streaming Architecture

The context service should support streaming, but should not necessarily proxy the customer's LLM.

## Recommended architecture

```text
User
 │
 ▼
Context API
 │
 ├── persist user message
 │
 ├── build context
 │
 ▼
Customer Agent
 │
 ├── LLM
 ├── tools
 └── business logic
 │
 ▼
Customer's streaming layer
 │
 ▼
User
```

The customer can stream agent output directly.

The Context API receives the completed message or deltas as events.

```text
POST /rooms/123/events

{
  "type": "agent.delta",
  "content": "The cheapest"
}
```

Then:

```text
{
  "type": "agent.delta",
  "content": " option is..."
}
```

Finally:

```text
{
  "type": "agent.completed"
}
```

This avoids turning the Context API into an LLM gateway.

---

# 10. Optional Managed Run API

A higher-level API can eventually be offered:

```http
POST /v1/rooms/:room_id/run
```

The service:

```text
Message
   ↓
Context
   ↓
Customer Agent
   ↓
LLM
   ↓
Tools
   ↓
Streaming response
```

However, this should be an optional convenience layer.

The core product remains the context API.

---

# 11. Context Engine

The Context Engine is the primary technical differentiation.

A basic implementation:

```text
New Message
     │
     ▼
Recent Context
     │
     ├───────────────┐
     ▼               ▼
Semantic Search    Room State
     │               │
     └───────┬───────┘
             ▼
      Candidate Context
             │
             ▼
       Relevance Ranking
             │
             ▼
      Reference Resolution
             │
             ▼
      Context Assembly
             │
             ▼
       Final Context
```

The system should avoid sending the entire room history to the model.

Instead:

> **Find the minimum sufficient context required to answer the current turn.**

---

# 12. Context vs Memory

These should be separate concepts.

## Memory

Longer-lived information:

```text
User prefers hotels with pools.
User usually travels to Bali.
User's preferred budget is $100/night.
```

## Conversation history

What happened:

```text
Yesterday we discussed hotels in Bali.
```

## Active context

What matters now:

```text
Current topic: hotel booking
Location: Bali
Budget: <$150
Selected hotel: hotel_123
Booking date: Friday
```

The product primarily owns **active context**, while memory is one source used to construct it.

---

# 13. Data Model

Initial PostgreSQL schema:

```sql
rooms (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

```sql
messages (
    id UUID PRIMARY KEY,
    room_id UUID NOT NULL,
    role TEXT NOT NULL,
    content TEXT,
    sequence BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL
);
```

```sql
room_state (
    room_id UUID PRIMARY KEY,
    summary TEXT,
    active_topic TEXT,
    state JSONB,
    updated_at TIMESTAMP NOT NULL
);
```

```sql
memories (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID,
    room_id UUID,
    content TEXT NOT NULL,
    embedding VECTOR,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL
);
```

---

# 14. Recommended Tech Stack

## Backend

**Go**

Why:

- High concurrency
- Low memory overhead
- Excellent HTTP performance
- Good fit for streaming
- Strong PostgreSQL support
- Easy deployment as a single binary

Suggested stack:

```text
Go
Chi
pgx
sqlc (optional)
OpenTelemetry
```

## Primary database

**PostgreSQL**

Use it for:

- tenants
- users
- rooms
- messages
- state
- API keys
- billing metadata
- configuration

Use **pgvector** initially for embeddings.

Avoid adding a dedicated vector database until scale actually requires it.

## Redis

Use Redis for:

- rate limiting
- distributed locks
- short-lived state
- queues
- pub/sub
- stream coordination

PostgreSQL remains the source of truth.

## Memory providers

Initially support:

```text
Honcho
Redis Agent Memory Server
```

Later:

```text
Native Context Engine
Mem0
Zep
Custom customer provider
```

## API protocol

Start with:

```text
REST
SSE
Webhooks
```

Add WebSockets only if there is a concrete use case.

## SDKs

Start with:

```text
TypeScript
Python
Go
```

The SDKs should be thin wrappers around the API.

---

# 15. Infrastructure

Initial deployment:

```text
                    Cloudflare
                        │
                        ▼
                  Load Balancer
                        │
                        ▼
                Go API Servers
                  │         │
                  ▼         ▼
             PostgreSQL    Redis
                  │
                  ▼
               pgvector
```

Async workers:

```text
Go API
  │
  ▼
Queue
  │
  ├── Summary worker
  ├── Memory worker
  ├── Embedding worker
  ├── Context evaluation worker
  └── Analytics worker
```

The synchronous request should do only what is necessary to produce the current context.

Expensive memory maintenance should happen asynchronously.

---

# 16. Multi-Tenancy

Every resource belongs to a tenant:

```text
Tenant
 ├── API keys
 ├── Users
 ├── Rooms
 ├── Messages
 ├── Memories
 └── Usage
```

Every query must be tenant-scoped.

Example:

```text
tenant_id
    ↓
room_id
    ↓
messages
```

Do not allow a room ID alone to be sufficient authorization.

---

# 17. Security

Important because conversation data may contain sensitive information.

Requirements:

- TLS everywhere
- Encryption at rest
- Tenant isolation
- API key authentication
- Role-based access
- Audit logs
- Data deletion API
- Configurable retention
- Export API
- Optional customer-managed encryption
- No cross-tenant retrieval
- Strict authorization on every room

Potential enterprise features:

```text
SSO
SCIM
SOC 2
GDPR tooling
Data residency
Customer-managed keys
Private deployment
VPC deployment
```

---

# 18. Observability

The product should expose context observability.

For every agent turn:

```text
Turn
 ├── Input tokens
 ├── Context tokens
 ├── Context sources
 ├── Retrieved messages
 ├── Retrieved memories
 ├── Summary used
 ├── State used
 ├── Context generation latency
 ├── Context cost
 └── Context quality
```

This enables:

> "Why did my agent receive this context?"

A useful debugging view:

```text
User message
     ↓
Retrieved:
  msg_123  relevance: 0.94
  msg_127  relevance: 0.88
  memory_45 relevance: 0.82

Excluded:
  msg_001
  msg_002
  msg_003

Final context:
  3,842 tokens
```

This could become a major product feature.

---

# 19. Context Quality Metrics

The system should eventually measure:

### Relevance

Did we retrieve information relevant to the current turn?

### Completeness

Did we retrieve everything necessary?

### Token efficiency

How much context was provided versus how much was actually needed?

### Reference resolution

Did:

```text
"it"
```

resolve to the correct entity?

### State consistency

Does the current state agree with the conversation?

### Freshness

Did the context use the most recent relevant information?

These metrics can become the foundation of a context evaluation product.

---

# 20. Developer Experience

The ideal integration should take minutes.

### Step 1

Create API key.

### Step 2

Create room.

```bash
curl -X POST https://api.example.com/v1/rooms \
  -H "Authorization: Bearer $API_KEY"
```

### Step 3

Send message.

```bash
curl -X POST \
  https://api.example.com/v1/rooms/room_123/messages \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "role": "user",
    "content": "Find me a hotel in Bali"
  }'
```

### Step 4

Request context.

```bash
curl -X POST \
  https://api.example.com/v1/rooms/room_123/context \
  -H "Authorization: Bearer $API_KEY"
```

### Step 5

Pass the context into the developer's agent.

```text
Your Agent
    +
Our Context
    ↓
LLM
```

---

# 21. User Interaction

The end user should not know the Context API exists.

Example:

```text
USER

Find me a hotel in Bali.
```

Agent:

```text
I found five hotels in Bali...
```

User:

```text
Which one is cheapest?
```

The context infrastructure determines:

```text
"which one"
    ↓
hotel candidates from previous turn
```

User:

```text
Does it have a pool?
```

Context infrastructure determines:

```text
"it"
    ↓
selected hotel
```

User:

```text
Actually, make the budget $150.
```

Context infrastructure updates:

```text
budget = $150
```

The agent receives a clean context package rather than the entire conversation.

---

# 22. Pricing Model

Pricing should primarily follow a **Base Operations + Tokens Processed** model, rather than billing strictly on tokens delivered to the LLM. 

The customer already pays their LLM provider for the tokens output by our service. If we build a successful `TokenBudget` feature that minimizes the tokens sent to the LLM, but we bill based on output tokens, we would cannibalize our own revenue.

Therefore, the pricing structure is twofold:
1. **Base Operations:** A flat rate for API requests (e.g., $X per 10,000 Messages stored or Context Fetches).
2. **Tokens Processed:** A charge based on the total volume of history the Context Engine had to ingest, scan, and rank under the hood to create the optimized package.

### Control Mechanisms
- **Limit LLM Costs:** Customers set a `TokenBudget` to enforce a strict ceiling on the tokens returned by `konteks`.
- **Limit Konteks Costs:** Customers set a `MaxHistoryDepth` (e.g., last 100 messages) to limit the amount of "Tokens Processed" by the engine.

Possible structure:

## Free

```text
$0/month

10,000 Operations
1M Tokens Processed
1–2 rooms/project
Limited retention
Community support
```

## Developer

```text
$29/month

100,000 Operations
10M Tokens Processed
More rooms
Longer retention
Context analytics
SSE
Memory providers
```

## Pro

```text
$99/month

1M Operations
100M Tokens Processed
Large context workloads
Advanced analytics
Custom retention
Higher rate limits
Priority support
```

## Enterprise

```text
Custom

Dedicated infrastructure
SSO / SCIM
Private deployment
Data residency
Custom retention
SLA
Customer-managed encryption
```

Avoid charging directly on "number of LLM tokens" if the customer is using their own LLM. Your value is **context infrastructure**, not inference.

---

# 23. Unit Economics

Major cost drivers:

```text
                Cost
                 │
       ┌─────────┼─────────┐
       ▼         ▼         ▼
    Storage    Embeddings  LLM processing
       │         │         │
       ▼         ▼         ▼
   Postgres    Provider    Context engine
```

The biggest risk is using an LLM synchronously for every context operation.

Optimize by:

- deterministic state updates
- embeddings
- cached summaries
- incremental summarization
- small models for extraction
- asynchronous memory processing
- context caching
- avoiding repeated embeddings
- avoiding full-history LLM calls

---

# 24. MVP

The first version should be intentionally small.

## Must have

```text
✓ Multi-tenant API
✓ Rooms
✓ Messages
✓ Persistent history
✓ Context retrieval
✓ Summarization
✓ pgvector retrieval
✓ Context package API
✓ SSE/events
✓ API keys
✓ Usage metering
```

## Do later

```text
→ Advanced long-term memory
→ Sophisticated entity graphs
→ Automatic reference resolution
→ Multiple memory providers
→ Context quality scoring
→ Enterprise security
→ Managed agent runtime
→ LLM gateway
```

The first objective is to prove:

> Developers can connect an existing agent and immediately get better conversation context without building their own context infrastructure.

---

# 25. Differentiation

The market already has memory systems.

The product should not compete only on:

> "We store your AI memories."

Instead:

> **We manage the complete context lifecycle of an AI conversation.**

Potential differentiation:

### 1. Room-first architecture

Conversation rooms are the fundamental primitive.

### 2. Context-first retrieval

Retrieve the minimum sufficient context rather than generic semantic memories.

### 3. Agent agnostic

Works with any agent framework.

### 4. Model agnostic

Works with any LLM.

### 5. Tool agnostic

Customer owns all tools.

### 6. Context observability

Show exactly why each piece of context was selected.

### 7. Context policies

Allow developers to control:

```text
max tokens
retention
memory scope
retrieval strategy
summarization
freshness
```

---

# 26. Competitive Positioning

Avoid positioning as:

```text
"Another AI memory database"
```

Instead:

```text
                    AI AGENT STACK

┌──────────────────────────────────────┐
│              AI AGENT               │
│ Reasoning / Tools / Business Logic  │
└──────────────────────┬───────────────┘
                       │
             ┌─────────▼─────────┐
             │ CONTEXT LAYER     │
             │                   │
             │ Rooms             │
             │ State             │
             │ Memory            │
             │ Retrieval         │
             │ Compaction        │
             │ Context Assembly  │
             └─────────┬─────────┘
                       │
             ┌─────────▼─────────┐
             │ STORAGE / MEMORY  │
             │                   │
             │ Postgres          │
             │ Redis             │
             │ Honcho            │
             │ Redis AMS         │
             │ Vector DB         │
             └───────────────────┘
```

The context layer is the product.

The storage and memory systems are infrastructure components.

---

# 27. Product Roadmap

## Phase 0 — Prototype

```text
Postgres
Go API
pgvector
One memory provider
Basic context retrieval
```

Goal:

> Demonstrate follow-up understanding.

## Phase 1 — MVP

```text
Rooms
Messages
Context API
SSE
Summarization
Memory
Usage metering
Dashboard
```

Goal:

> First external developers integrate it.

## Phase 2 — Context Engine

```text
Reference resolution
Structured state
Context ranking
Automatic compaction
Context policies
```

Goal:

> Context quality becomes meaningfully better than generic memory retrieval.

## Phase 3 — Platform

```text
Multiple memory providers
Context evaluation
Observability
Enterprise security
Advanced analytics
SDK ecosystem
```

Goal:

> Become infrastructure for production AI agents.

---

# 28. Dashboard

The dashboard should show:

```text
Project
 ├── Rooms
 ├── Users
 ├── Messages
 ├── Context usage
 ├── Memory
 ├── Latency
 ├── Errors
 └── Cost
```

For each room:

```text
Room #123

Current topic:
Hotel booking

State:
Location: Bali
Budget: $150
Selected hotel: Hotel ABC
Date: Friday

Context:
3,842 tokens

Retrieved:
msg_123
msg_129
memory_45

Excluded:
msg_001
msg_002
...
```

This makes the system debuggable.

---

# 29. Core Product Principle

The central design principle should be:

> **The customer owns the intelligence. We own the conversational state.**

The customer decides:

```text
What to think
What tools to call
What action to take
Which model to use
```

The Context Infrastructure decides:

```text
What happened
What matters now
What should be remembered
What should be forgotten
What context should be provided
```

---

# 30. One-Sentence Pitch

> **A context infrastructure API that gives AI agents the right conversation state and history for every turn—without developers having to build memory, retrieval, summarization, or context-window management themselves.**

## Short headline

> **You build the agent. We manage its context.**

## Alternative headlines

> **The context layer for AI agents.**

> **Stateful conversations, as an API.**

> **Give your AI agent the context it actually needs.**

> **Stop rebuilding memory infrastructure for every AI agent.**


---

# 31. Critical Product Boundary: Context Infrastructure vs Agent Runtime

The product must preserve a strict separation between **context infrastructure** and **agent execution**.

The customer owns:

```text
LLM
Agent reasoning
System prompts
Business logic
Tools
Tool execution
External APIs
Databases
Policies
Agent framework
```

The Context Infrastructure owns:

```text
Rooms
Messages
Conversation state
Context retrieval
Memory
Summarization
Compaction
Reference resolution
Context assembly
Conversation events
```

### Core principle

> **The customer's agent decides what to do. Our infrastructure remembers what happened and tells the agent what matters.**

The product should never require customers to give up control of their AI layer.

---

# 32. `room.send()` Does Not Mean "Run My Agent"

A method such as:

```typescript
room.send("Which one is cheaper?")
```

must not imply:

```text
room.send()
    ↓
Our LLM
    ↓
Our agent
    ↓
Our tools
    ↓
Our business logic
```

That would turn the product into an agent runtime.

Instead:

```text
room.send()
    ↓
Store message
    ↓
Update conversation state
    ↓
Retrieve relevant context
    ↓
Return context package
```

The customer's agent remains outside the system.

### Better interpretation

> `room.send()` means **a message entered this conversation**, not **execute an agent**.

---

# 33. Recommended API Separation

The API should expose clear primitives.

## 33.1 Create a message

```http
POST /v1/rooms/:room_id/messages
```

```json
{
  "role": "user",
  "content": "Which one is cheaper?"
}
```

The Context Infrastructure:

```text
Persist message
     ↓
Update room
     ↓
Retrieve relevant context
     ↓
Prepare context
```

It does not invoke an LLM.

---

## 33.2 Get context

```http
POST /v1/rooms/:room_id/context
```

```json
{
  "query": "Which one is cheaper?"
}
```

Response:

```json
{
  "room_id": "room_123",
  "context": {
    "summary": "User is comparing hotels in Bali.",
    "state": {
      "location": "Bali"
    },
    "relevant_messages": [],
    "memories": [],
    "entities": []
  }
}
```

The customer then decides what to do:

```python
context = context_api.get_context(
    room_id,
    user_message
)

result = my_agent.run(
    user_message,
    context
)
```

The Context Infrastructure never needs to know how `my_agent.run()` works.

---

## 33.3 Conversation events

The customer's agent can send execution events back:

```http
POST /v1/rooms/:room_id/events
```

Example:

```json
{
  "type": "tool.started",
  "tool": "search_hotels",
  "arguments": {
    "location": "Bali"
  }
}
```

Then:

```json
{
  "type": "tool.completed",
  "tool": "search_hotels",
  "output": {
    "hotels": []
  }
}
```

Then:

```json
{
  "type": "assistant.message",
  "content": "I found five hotels..."
}
```

The Context Infrastructure treats these as conversation events and uses them to maintain context.

---

# 34. Tools Belong to the Customer

Tools should be opaque to the Context Infrastructure.

For example, the customer may have:

```text
search_hotels()
get_customer()
create_booking()
query_database()
send_whatsapp()
process_payment()
```

The Context Infrastructure does not execute these tools.

It only records their lifecycle:

```text
tool.started
      ↓
tool.completed
      ↓
tool output becomes conversation context
```

This allows the context engine to understand that a tool introduced information into the conversation.

For example:

```text
Agent:
search_hotels()

Tool:
Hotel A — $120
Hotel B — $90
Hotel C — $150

Agent:
Here are the options.

User:
Which one is cheapest?
```

The context engine can resolve:

```text
"which one"
     ↓
Hotel A / Hotel B / Hotel C
     ↓
previous tool result
```

The customer still owns the actual hotel API.

---

# 35. Agent-Agnostic Architecture

The customer can use any agent architecture:

```text
Your Context API
       │
       ├── OpenAI Agent
       ├── Claude Agent
       ├── Gemini Agent
       ├── LangGraph
       ├── Custom Go Agent
       ├── Custom Python Agent
       ├── Custom TypeScript Agent
       └── Local LLM Agent
```

The Context Infrastructure only exposes state and context.

This avoids vendor lock-in.

---

# 36. Model-Agnostic Architecture

The customer controls the model:

```text
Context API
     │
     ▼
Customer Agent
     │
     ├── GPT
     ├── Claude
     ├── Gemini
     ├── Qwen
     ├── Llama
     └── Local model
```

The Context API should not require a specific inference provider.

---

# 37. Streaming Without Owning the LLM

Streaming should be treated as **conversation events**, rather than requiring the Context Infrastructure to proxy every LLM token.

Recommended architecture:

```text
                         USER
                          │
                          ▼
                  Customer Application
                          │
             ┌────────────┴────────────┐
             │                         │
             ▼                         ▼
      Context Infrastructure     Customer Agent
             │                         │
             │                   ┌─────┼─────┐
             │                   ▼     ▼     ▼
             │                  LLM   Tools Logic
             │
             ◄────── events ───────────┘
```

The customer can stream the LLM response directly to the end user.

The Context Infrastructure receives events such as:

```text
agent.started
agent.delta
tool.started
tool.completed
agent.completed
```

For example:

```json
{
  "type": "agent.delta",
  "content": "The cheapest"
}
```

Then:

```json
{
  "type": "agent.delta",
  "content": " option is..."
}
```

Finally:

```json
{
  "type": "agent.completed"
}
```

This avoids turning the Context Infrastructure into an LLM gateway.

---

# 38. Two API Levels

The product can support two levels.

## Low-level Context API

For maximum control:

```text
messages.create()
context.get()
events.create()
```

Customer controls everything else.

```text
Customer Agent
      │
      ├── Context API
      ├── LLM
      ├── Tools
      └── Business logic
```

This should be the fundamental product.

## Optional high-level Run API

For customers who want convenience:

```http
POST /v1/rooms/:room_id/run
```

This could orchestrate:

```text
Message
  ↓
Context
  ↓
Customer-configured agent
  ↓
LLM
  ↓
Tools
  ↓
Streaming
```

But this should remain optional.

The Context API must never require customers to use the managed agent runtime.

---

# 39. Customer-Controlled Architecture

The ideal customer architecture is:

```text
                    CUSTOMER
┌──────────────────────────────────────────────┐
│                                              │
│  Application                                 │
│      │                                       │
│      ├───────────────┐                       │
│      │               │                       │
│      ▼               ▼                       │
│  Context API      Your Agent                 │
│                      │                       │
│                ┌─────┼─────┐                 │
│                ▼     ▼     ▼                 │
│               LLM  Tools  Logic              │
│                                              │
└──────────────────────────────────────────────┘
```

The two systems communicate through:

```text
Messages
Context
Events
```

This is the cleanest product boundary.

---

# 40. Why This Boundary Matters

If the Context Infrastructure owns the LLM, tools, or business logic, customers lose flexibility.

They may be unable to:

- use their preferred model
- use proprietary tools
- keep business logic private
- use an existing agent framework
- run their own inference
- change their LLM provider
- customize agent behavior

Instead, the product should provide:

> **Bring Your Own Agent (BYOA).**

The customer should be able to plug the Context Infrastructure into an existing production agent without rewriting it.

---

# 41. Core Product Abstraction

The most important abstraction is therefore not:

```text
memory.add()
memory.search()
```

It is:

```text
Room
```

A room represents the complete stateful conversation.

```text
Room
 ├── Messages
 ├── Assistant responses
 ├── Tool calls
 ├── Tool results
 ├── State
 ├── Summary
 ├── Entities
 ├── References
 ├── Memories
 └── Events
```

The Context Engine uses all of these sources to construct the next context package.

---

# 42. Revised Product Positioning

Do not position the product as:

> Another AI memory database.

Do not position it as:

> A hosted vector database for AI.

Do not position it as:

> An agent framework.

Instead:

> **The context infrastructure for stateful AI agents.**

### Primary headline

> **You build the agent. We manage its context.**

### Subheadline

> Store every conversation, maintain state, retrieve relevant history, and deliver the right context to your agent—without rebuilding memory and context infrastructure yourself.

### Alternative headline

> **Your agent decides what to do. We remember what happened.**

### Developer-oriented headline

> **Stateful conversations, as an API.**

---

# 43. Competitive Implication: Zep

Zep is a direct competitor for the **context and memory layer**.

Zep demonstrates that context engineering is already a recognized infrastructure category.

Therefore, the product should not attempt to differentiate solely through:

```text
Memory
Semantic search
Context retrieval
Temporal knowledge
Context assembly
```

Instead, differentiation should investigate the broader **conversation state/runtime layer**:

```text
Room lifecycle
+
Messages
+
Agent events
+
Tool events
+
State
+
Context
+
Streaming
+
Observability
+
Context policies
```

The product can still use Zep, Honcho, Redis Agent Memory Server, or other systems internally.

The customer sees one stable Context API.

---

# 44. Provider-Abstraction Strategy

The architecture should allow:

```text
                 Context API
                      │
                Context Engine
                      │
       ┌──────────────┼──────────────┐
       │              │              │
       ▼              ▼              ▼
     Zep            Honcho        Redis AMS
       │              │              │
       └──────────────┼──────────────┘
                      ▼
                Context Package
                      │
                      ▼
               Customer Agent
```

This allows the infrastructure to start by reusing mature systems instead of rebuilding every memory primitive.

Over time, proprietary components can replace providers where they create measurable differentiation.

---

# 45. Product Principle

The strongest product principle is:

> **Own the context lifecycle, not the customer's intelligence.**

The customer owns:

```text
What to think
What to do
Which model to use
Which tools to call
Which business rules apply
```

The Context Infrastructure owns:

```text
What happened
What matters now
What should be remembered
What should be summarized
What context should be delivered
How conversation state evolves
```

This creates a clear, composable infrastructure layer between messaging applications and AI agents.


---

# 47. Refined Product Definition: Context-as-a-Service (CaaS)

The product is best understood as:

> **Context-as-a-Service for AI agents.**

The core value is **not storing chat messages**.

The core value is:

> **Given an agent turn and the customer's available data, produce the context that the agent actually needs.**

This distinction is critical.

## Storage is optional

A customer may already have:

```text
PostgreSQL
MongoDB
Firestore
Supabase
CRM
Data warehouse
Chat history
Tool-result storage
Business databases
```

They should not need to migrate their data into the Context Infrastructure.

The service can operate as a **context layer on top of existing customer data**.

---

# 48. CaaS Architecture

```text
                    CUSTOMER SYSTEM
┌──────────────────────────────────────────────────────┐
│                                                      │
│  Chat DB          Business DB       Tool Results     │
│     │                  │                  │           │
│     └──────────────────┼──────────────────┘           │
│                        │                              │
└────────────────────────┼─────────────────────────────┘
                         │
                         ▼
              ┌───────────────────────┐
              │   Context-as-a-Service│
              │                       │
              │ Context Retrieval     │
              │ Memory                │
              │ State                 │
              │ Summarization         │
              │ Reference Resolution  │
              │ Context Ranking       │
              │ Compaction            │
              │ Context Assembly      │
              └───────────┬───────────┘
                          │
                          ▼
                  CONTEXT PACKAGE
                          │
                          ▼
                 CUSTOMER'S AGENT
                          │
                   ┌──────┼──────┐
                   ▼      ▼      ▼
                  LLM    Tools   Logic
```

The customer remains the source of truth.

CaaS creates a derived **context layer** optimized for AI agent consumption.

---

# 49. The Core API

A fundamental operation should be conceptually:

```typescript
const context = await caas.context({
    room: roomId,
    input: userMessage,

    sources: {
        conversation: customerConversation,
        user: customerUserData,
        tools: recentToolResults,
        business: relevantBusinessData
    },

    budget: {
        maxTokens: 8000
    }
});
```

The service returns:

```json
{
  "context": "...",
  "state": {},
  "references": {},
  "sources": [],
  "token_count": 4210
}
```

The customer then controls the agent:

```typescript
const response = await myAgent.run({
    input: userMessage,
    context
});
```

The complete lifecycle is:

```text
Customer Data
      │
      ▼
     CaaS
      │
      ▼
Optimal Context
      │
      ▼
Customer Agent
      │
 ┌────┼─────┐
 ▼    ▼     ▼
LLM  Tools Logic
```

---

# 50. Bring Your Own Data (BYOD)

A major product capability should be **Bring Your Own Data**.

Customers should be able to use their existing data infrastructure.

Possible sources:

```text
PostgreSQL
MySQL
MongoDB
Firestore
Supabase
S3
Kafka
Webhooks
REST APIs
CRM systems
Data warehouses
Existing conversation stores
```

There are two integration patterns.

## Pattern A — Push

The customer sends relevant events to CaaS:

```http
POST /v1/events
```

```json
{
  "room_id": "room_123",
  "type": "tool.completed",
  "tool": "search_hotels",
  "data": {
    "hotels": []
  }
}
```

CaaS builds and maintains an optimized context index.

## Pattern B — Pull

CaaS retrieves data from customer-defined sources when constructing context.

```text
Agent turn
    ↓
CaaS
    ├── conversation source
    ├── memory source
    ├── business source
    └── tool-result source
    ↓
Context assembly
```

Pattern A is simpler and generally preferable for latency and reliability.

Pattern B is useful when customers cannot duplicate or stream their data.

---

# 51. CaaS Is Not RAG

Traditional RAG:

```text
Question
   ↓
Vector search
   ↓
Documents
   ↓
LLM
```

CaaS:

```text
Agent turn
   ↓
┌───────────────────────────────┐
│ Conversation                  │
│ Memory                        │
│ User state                    │
│ Tool results                  │
│ Business state                │
│ Previous decisions            │
│ Current task                  │
│ Entity references             │
│ External data                 │
└───────────────┬───────────────┘
                ↓
         Context reasoning
                ↓
        Optimal context
                ↓
              Agent
```

RAG is one **input mechanism** inside CaaS.

It is not the entire product.

---

# 52. CaaS Is Not Memory

Memory answers:

> "What information about this user should I remember?"

CaaS answers:

> "What information does the agent need right now?"

For example:

```text
Memory:
User prefers hotels with pools.

Conversation:
User is currently choosing Hotel B.

Tool result:
Hotel B has a pool.

Business state:
Hotel B is currently available.

Current message:
"Can we book it?"
```

The CaaS context package should combine these into:

```text
The user is considering Hotel B.
Hotel B has a pool.
Hotel B is currently available.
The user wants to book Hotel B.
```

The value is the **combination and prioritization of context sources**.

---

# 53. CaaS Is Not Conversation Storage

Conversation storage is:

```text
messages
 ├── message 1
 ├── message 2
 ├── message 3
 └── message 4
```

CaaS transforms that raw history into:

```text
Current topic
Relevant history
Current state
Entities
References
Constraints
Decisions
Memories
Tool results
```

Therefore, a customer can keep their own chat database.

CaaS simply creates an AI-optimized representation of the information.

---

# 54. Context as a Derived Data Layer

A useful architectural model is:

```text
              CUSTOMER SOURCE OF TRUTH

     ┌────────────┬────────────┬────────────┐
     │            │            │            │
  Messages     Tool Data    User Data   Business Data
     │            │            │            │
     └────────────┴────────────┴────────────┘
                          │
                          ▼
                  Context-as-a-Service
                          │
                 Derived Context Layer
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
       State           Memory        Retrieval Index
          │               │               │
          └───────────────┼───────────────┘
                          ▼
                    Context Package
                          │
                          ▼
                     AI Agent
```

This makes CaaS analogous to an infrastructure layer that transforms raw application data into a representation optimized for agent reasoning.

---

# 55. The Real Product Promise

The strongest product promise becomes:

> **Your data goes in. The context your agent needs comes out.**

Or:

> **Turn your application's data into agent-ready context.**

Or:

> **The context layer between your data and your AI agent.**

Primary positioning:

> **Context-as-a-Service for AI agents.**

Primary headline:

> **You build the agent. We build its context.**

Alternative:

> **Your data. Our context engine. Your agent.**

---

# 56. Updated Product Boundary

The final architecture should therefore distinguish three layers:

```text
┌──────────────────────────────────────────────┐
│              APPLICATION                    │
│                                              │
│  Chat UI                                     │
│  Business logic                              │
│  Customer database                           │
│  Tools                                       │
└───────────────────────┬──────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────┐
│           CONTEXT-AS-A-SERVICE               │
│                                              │
│  Context ingestion                           │
│  Context retrieval                           │
│  Memory                                      │
│  State                                       │
│  Reference resolution                        │
│  Summarization                               │
│  Context ranking                             │
│  Compaction                                  │
│  Context assembly                            │
└───────────────────────┬──────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────┐
│                  AI AGENT                    │
│                                              │
│  LLM                                         │
│  Reasoning                                   │
│  Tool selection                              │
│  Business decisions                          │
└──────────────────────────────────────────────┘
```

The application and agent remain customer-controlled.

CaaS is the infrastructure layer connecting application data to agent context.

---

# 57. Revised Strategic Thesis

The startup thesis should no longer be:

> "Developers need somewhere to store their AI conversations."

It should be:

> **As AI agents become stateful and operate across long-running interactions, developers need infrastructure that continuously transforms application data into the context required for each agent turn.**

This is a much larger and more defensible problem.

The long-term opportunity is to become the **context plane for AI applications**:

```text
                 AI APPLICATION
                       │
                       ▼
              ┌─────────────────┐
              │   CONTEXT PLANE │
              │                 │
              │  What happened? │
              │  What matters?  │
              │  What changed?  │
              │  What's related?│
              │  What's needed? │
              └────────┬────────┘
                       │
                       ▼
                    AGENT
```

The Context Plane should be independent of:

```text
LLM provider
Agent framework
Application database
Tool provider
Messaging platform
```

That independence is the core infrastructure opportunity.
