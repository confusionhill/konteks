# Latency Reduction & API Improvement Research

This document outlines the findings from researching the customer-facing APIs (specifically `GetContextPackage` and `CreateMessage`) to identify rooms for improvement, focusing on latency reduction and system reliability.

## User Review Required

> [!IMPORTANT]  
> Please review the proposed solutions below. I have prioritized them based on their impact on latency and system reliability. Let me know if you agree with this prioritization and which ones you'd like to tackle first.

## Open Questions

> [!WARNING]  
> - For **FastMode=false** in `GetContextPackage`, is it strictly required to extract state and resolve references *synchronously* on every read? If we move this to a background process (or rely on the existing background worker), we can drastically cut latency.
> - For **Archived Rooms**, when a message is created on an archived room, it currently returns success immediately but processes the save asynchronously. If the worker fails, data is lost. Are you comfortable with moving this to a fast queue/Redis buffer before acknowledging success to the client?

## Identified Problems and Proposed Solutions (Prioritized)

### Priority 1: Synchronous Room Hydration in `GetContextPackage`
- **Problem**: When a customer requests context for an `Archived` room, `GetContextPackage` synchronously blocks the HTTP request to fetch the archive from S3, decompress it, parse JSON, and perform database writes (Stage 1 hydration). Furthermore, if another request triggers hydration, it falls into a synchronous 50ms polling loop (up to 5 seconds). This results in unpredictable and severe latency spikes.
- **Solution**: 
  - **Option A (Proactive)**: Introduce a lightweight `/rooms/:room_id/warmup` endpoint that clients call when a user session starts, so hydration happens before they actually need the context.
  - **Option B (Async & Retry)**: If a room is archived, immediately return a `202 Accepted` or specific status indicating "Hydrating," and let the client poll or use WebSockets to get notified when hydration is complete.
- **Expected Boost**: **Massive (1000ms - 5000ms reduction)** for archived rooms. Latency drops from seconds down to ~50-100ms.

### Priority 2: Synchronous LLM Calls in `GetContextPackage`
- **Problem**: When `FastMode` is false, `GetContextPackage` makes multiple synchronous LLM calls (`ExtractStateAndSummary`, `GenerateEmbedding`, `ResolveReferences`) via an `errgroup`, blocking the request until all complete. It also writes the updated state back to the database synchronously. LLM calls are notoriously slow and will violate typical API latency SLAs.
- **Solution**:
  - **Move Heavy Lifting to Background**: The `message_usecase` already has a background worker that updates state and summaries when new messages arrive. We should rely on this background state instead of re-calculating it synchronously on read.
  - **Streaming/WebSockets**: If on-the-fly calculation is absolutely necessary, use a streaming response (Server-Sent Events) so the client receives the base package immediately, and the LLM-enriched data as it becomes available.
- **Expected Boost**: **High (2000ms - 8000ms reduction)** for `FastMode=false` queries. Latency drops to ~100-300ms.

### Priority 3: Data Loss Risk in `CreateMessage` (Archived Rooms)
- **Problem**: In `message_usecase.go`, if `CreateMessage` is called on an archived room, it spawns a background goroutine to handle hydration and immediately returns a dummy message object to the client. It does *not* save the new message synchronously. If the server crashes or the goroutine fails, the message is permanently lost despite the client receiving a successful response.
- **Solution**: 
  - Introduce a fast append-only buffer (e.g., Redis List or an append-only WAL table in DB) to synchronously persist incoming messages for hydrating rooms. The background worker will drain this buffer and insert them into the main database once hydration completes.
- **Expected Boost**: **Critical Reliability Boost**. Prevents 100% data loss scenarios during hydration. Negligible latency change (~2-5ms added for Redis write).

### Priority 4: Sequential Database Queries
- **Problem**: `GetContextPackage` performs several sequential database queries: `roomRepo.GetByID`, `roomRepo.GetState`, `messageRepo.GetMaxSequence`, `messageRepo.ListByRoom`, and `roomRepo.ListChildRooms`. Each adds a network round-trip delay to the critical path.
- **Solution**: 
  - Group queries into a single database transaction or query where possible.
  - Execute independent queries concurrently using goroutines (e.g., once we have the `roomID` and verify permissions, we can fetch `State`, `MaxSequence`, `Messages`, and `ChildRooms` at the same time).
- **Expected Boost**: **Moderate (10ms - 30ms reduction)** per request by parallelizing database round trips.

### Priority 5: Synchronous Payload Decryption
- **Problem**: In both `GetContextPackage` and the background workers, iterating through historical messages to decrypt their contents is done sequentially.
- **Solution**: 
  - Implement a concurrent decryption worker pool or simply spawn goroutines for batch decryption, especially when `msgLimit` is high, to maximize CPU utilization and reduce wall-clock time.
- **Expected Boost**: **Minor (1ms - 5ms reduction)**. CPU-bound AES-GCM decryption is extremely fast, so this is a micro-optimization unless the history depth is massive.

---

Let me know your thoughts on this prioritization, and I will prepare the execution plan!
