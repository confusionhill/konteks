# Context Poisoning Mitigation: Solution Approaches & Comparative Analysis

Comparative architectural analysis of **4 distinct mitigation paradigms** for preventing Context Poisoning, Prompt Injections, and Memory Pollution in ContextKit.

---

## 1. Comparative Solution Matrix

| Approach | Architectural Strategy | Ingress Latency | Retrieval Latency | LLM Token Cost Overhead | Schema Impact |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Approach A: Inline Active Ingress/Egress Guardrails** | Multi-layer classifier + contradiction resolver + XML sandbox | 0ms (Async background) | ~0ms | Low (~3-5%) | Moderate (`trust_score`, `status`) |
| **Approach B: Deterministic Trust-Level DB Sandboxing** | Physical separation of trusted vs untrusted data via HMAC signatures | 0ms | ~5ms (Dual-query join) | 0% (No LLM scanning) | High (Dual DB tables & signed origin fields) |
| **Approach C: Post-Retrieval Guard LLM Sanitization** | Late-binding inspection of retrieved items during `GetContextPackage` | 0ms | +200ms to 500ms (LLM Guard pass) | Moderate to High (Per context request) | 0% (Zero schema changes) |
| **Approach D: Immutable Event-Log Memory Graphs** | Append-only DAG revision graph tracking supersedes & overrides | 0ms | ~15-20ms (Graph traversal) | Low | High (Append-only DAG structure) |

---

## 2. Detailed Breakdown of Alternative Solutions

### Approach A: Inline Active Ingress/Egress Guardrails (Recommended Balanced Approach)
- **Mechanism**: Run pattern scanning and entropy checks in background workers during indexing. Wrap egress context in strict XML sandbox tags (`<user_context trust_level="...">`).
- **Side Effects on ContextKit**:
  - **Positive**: Low latency, high security against prompt injections, automatic contradiction resolution.
  - **Negative (Side Effects)**: Potential false positives on developer prompt examples (e.g., code snippets containing system prompt text). Requires threshold tuning.

---

### Approach B: Deterministic Trust-Level DB Sandboxing (Zero-LLM Cost Approach)
- **Mechanism**: No LLM classification. Partition memories into two strict PostgreSQL tables:
  1. `system_verified_memories` (Signed with HMAC-SHA256 by trusted API keys; weighted 1.0 in search).
  2. `untrusted_user_memories` (Unsigned user input facts; stripped of control tokens, weighted 0.3, strictly sandboxed).
- **Side Effects on ContextKit**:
  - **Positive**: **Zero LLM API cost overhead**. 100% deterministic isolation.
  - **Negative (Side Effects)**: API clients must explicitly tag memory provenance. Requires running dual-query joins in PostgreSQL/pgvector.

---

### Approach C: Post-Retrieval Guard LLM Sanitization (Zero DB Migration Approach)
- **Mechanism**: Store all user input as-is in PostgreSQL. When `GetContextPackage` is invoked, pass the candidate retrieved memories through a small, fast Guard LLM (e.g. Llama-3-Guard or Claude Haiku) to filter out injections or contradictory facts before returning to the caller.
- **Side Effects on ContextKit**:
  - **Positive**: **Zero database schema migrations required**. Easiest to implement initially.
  - **Negative (Side Effects)**: **Severe Latency Penalty on `GetContextPackage`** (+200ms-500ms per request). Increases per-request OpenRouter API costs.

---

### Approach D: Immutable Event-Log Memory Graphs (Audit & Time-Travel Approach)
- **Mechanism**: Memories are append-only events in a Directed Acyclic Graph (DAG). Outdated facts are never updated in-place; instead, new memory entries reference `overrides_memory_id`. Vector retrieval traverses graph nodes to resolve active state.
- **Side Effects on ContextKit**:
  - **Positive**: Complete audit trail and time-travel capability (inspect room state at turn 5 vs turn 25).
  - **Negative (Side Effects)**: Storage growth; graph traversal logic adds complexity to pgvector queries.

---

## 3. Why Multiple Approaches Exist & Which To Choose

Multiple approaches exist because context engineering applications must balance three competing trade-offs:
1. **Security & Safety** (Stopping prompt injections)
2. **Retrieval Latency** (Sub-100ms context packaging vs 500ms guard passes)
3. **Operational Cost** (LLM token overhead)

### Recommended Hybrid Strategy for ContextKit:
Combine **Approach B (Trust-Level Sandboxing)** for deterministic data isolation with **Approach A (Async Conflict Decay)** in background workers. This guarantees:
- **Zero latency penalty** on `GetContextPackage`.
- **Zero false positives** for trusted system artifacts.
- **Low LLM token cost**.
