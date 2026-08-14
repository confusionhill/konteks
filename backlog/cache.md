# Redis Caching & Performance Backlog

This document tracks all caching requirements, data structures, and cache invalidation strategies to be implemented during the **Redis Migration Infrastructure Phase**.

---

## 1. Artifact & Core Memory Caching Architecture

```mermaid
flowchart TD
    Req["POST /v1/rooms/:room_id/context (include_artifacts = true)"] --> BlacklistCheck{"1. Check Room Blacklist"}
    
    BlacklistCheck -->|Redis Hit < 0.5ms| RedisBlacklist[("Redis Set: room:room_123:blacklist")]
    BlacklistCheck -->|Redis Miss| PGBlacklist[("PostgreSQL room_artifact_blacklists")]

    RedisBlacklist --> VectorSearch["2. pgvector Cosine Similarity Search (Filtered)"]
    PGBlacklist --> VectorSearch

    VectorSearch --> GraphCheck{"3. Expand 1-Hop References"}
    GraphCheck -->|Redis Hit < 0.5ms| RedisGraph[("Redis Set: artifact:graph:art_1")]
    GraphCheck -->|Redis Miss| PGGraph[("PostgreSQL artifact_references")]

    RedisGraph --> ContentCheck{"4. Fetch Chunk Content"}
    PGGraph --> ContentCheck

    ContentCheck -->|Redis Hit| RedisChunk[("Redis String: artifact:chunk:c_1")]
    ContentCheck -->|Redis Miss| PGChunk[("PostgreSQL artifact_chunks")]

    RedisChunk --> Assemble["Assemble Context Package (~10-15ms)"]
    PGChunk --> Assemble
```

---

## 2. Redis Key Patterns & Data Structures

### A. Room Blacklist Mask Cache
- **Key Pattern**: `room:{room_id}:blacklist:artifacts` (Redis Set `SADD`)
- **Key Pattern**: `room:{room_id}:blacklist:groups` (Redis Set `SADD`)
- **Purpose**: Microsecond room-level exclusion check before performing vector/graph search.
- **Operations**:
  - `SISMEMBER room:{room_id}:blacklist:artifacts {artifact_id}`
  - Fast Set Difference (`SDIFF`) during query assembly.
- **Invalidation**: On `POST/DELETE /v1/rooms/:room_id/blacklist`, update Redis Set and publish event.

### B. Knowledge Graph Adjacency List Cache
- **Key Pattern**: `artifact:graph:{project_id}:{source_artifact_id}` (Redis Set `SADD`)
- **Purpose**: Fast 1-hop reference edge traversal (resolving linked artifacts without recursive SQL queries).
- **Operations**:
  - `SMEMBERS artifact:graph:{project_id}:{source_artifact_id}`
  - Returns array of target artifact IDs in < 0.5 ms.
- **Invalidation**: When an artifact's Markdown links or YAML frontmatter references change, update Redis Set.

### C. Artifact Content & Chunk Cache
- **Key Pattern**: `artifact:chunk:{chunk_id}` (Redis Hash / String JSON)
- **TTL**: 24 hours (LRU eviction).
- **Purpose**: Avoid reading raw chunk text from PostgreSQL disk during final context package compilation.

---

## 3. General Room & State Caching (Pre-Existing Scope)

- **Active Room State Cache**: `room:{room_id}:state` (JSON string)
- **API Key Hashing & Auth Cache**: `apikey:{key_hash}` (String string)
- **Rate Limiting Counters**: `ratelimit:{project_id}:{endpoint}` (Redis Sliding Window)
