# ContextKit: Product Gaps & Roadmap Alignment Issues

This document maps out identified product gaps and developer experience (DX) friction points where the current implementation deviates from or falls short of the production-grade vision in [ai_agent_context_infrastructure_product(2).md](file:///Users/dika/Documents/github/contextkit/ai_agent_context_infrastructure_product(2).md).

---

## 1. Room Policy & Schema Mutation Gap (High Impact DX)
*   **Problem:** Currently, a developer can only configure a room's `state_schema` and `default_policy` parameters during the initial `POST /v1/rooms` call. There are no API endpoints or repository methods supporting `PUT` or `PATCH` on `/v1/rooms/:room_id`.
*   **Impact:** If a developer upgrades their downstream LLM (e.g., from GPT-3.5 to GPT-4o, expanding their `TokenBudget` from 4k to 128k tokens) or refactors their state variables extraction schema, they cannot modify the active room. They are forced to delete and recreate the room, which destroys the user's active conversation history.
*   **Action Plan:** Implement `/v1/rooms/:room_id` `PATCH` endpoint allowing dynamic mutation of `default_policy` and `state_schema` without touching the message/state logs.

---

## 2. Lack of Formal SDK Packages (Integration Friction)
*   **Problem:** The PRD roadmaps a "Multi-language SDK ecosystem" (TypeScript, Python, Go) in Phase 3. Currently, the project only provides raw REST endpoints and sample Python client code inside `example/app.py`.
*   **Impact:** Integrators must write custom boilerplate `fetch` / `requests` wrapper functions to handle authentication, parsing, and SSE streams, which increases developer integration friction.
*   **Action Plan:** Publish formal SDK packages:
    *   `@contextkit/node` (TypeScript/npm)
    *   `contextkit-python` (pip)
    *   `contextkit-go` (go get module)

---

## 3. No Context Quality Metrics (Observability Gap)
*   **Problem:** Section 19 of the PRD outlines the requirement for **Context Quality Metrics** (observability on context relevance, compression ratios, and retrieval hit-rates).
*   **Impact:** While the developer dashboard displays operations count and cost billing, it does not tell the developer if their context packing is *working*. They cannot determine if the `TokenBudget` is truncating critical facts, or if the agent is receiving low-relevance semantic chunks.
*   **Action Plan:** Build a telemetry service that calculates and logs:
    *   **Retrieval Precision:** Cosine distance of returned memories vs. query.
    *   **Truncation Ratio:** Percentage of relevant history discarded due to `TokenBudget` limits.
    *   **State Drift:** Frequency of state updates modifying existing variables.

---

## 4. Optional High-Level Run API (Convenience Gap)
*   **Problem:** Section 38 outlines the "Optional high-level Run API" (`POST /v1/rooms/:room_id/run`) to let ContextKit orchestrate LLM execution and tool-call loops (similar to OpenAI Threads/Runs). Currently, only the low-level Context API is built.
*   **Impact:** Developers must write custom double-message write loops and LLM invocations in their backend. While the PRD states this is optional, developers looking for an out-of-the-box solution may find the low-level API too verbose.
*   **Action Plan:** Design a managed `/run` handler that takes an agent's configuration, calls the downstream LLM, automatically fetches and saves context, and returns a stream of events/tokens.
