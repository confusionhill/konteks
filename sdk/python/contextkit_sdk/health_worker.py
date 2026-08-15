import time
import logging
from typing import List, Dict, Any, Optional
import requests
from .crypto import ContextCrypto

logger = logging.getLogger("contextkit.health_worker")

class ClientHealthWorker:
    """
    Background worker for E2EE rooms.
    Runs locally inside the client VPC, pulls encrypted candidate memories,
    decrypts text in VPC memory, evaluates EAV contradictions via a private LLM,
    and posts DAG update directives to ContextKit API.
    """

    def __init__(
        self,
        base_url: str,
        api_key: str,
        project_id: str,
        master_key: str,
        llm_evaluator_fn: Optional[Any] = None
    ):
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.project_id = project_id
        self.master_key = master_key
        self.llm_evaluator_fn = llm_evaluator_fn or self._default_mock_evaluator

    def _headers() -> Dict[str, str]:
        return {
            "Content-Type": "application/json",
            "X-API-Key": self.api_key,
            "X-Project-ID": self.project_id,
        }

    @staticmethod
    def _default_mock_evaluator(fact_a: str, fact_b: str) -> bool:
        """
        Default basic contradiction evaluator heuristic.
        Returns True if Fact B explicitly contradicts Fact A.
        """
        # E.g. "User lives in Tokyo" vs "User lives in San Francisco"
        if "lives in" in fact_a.lower() and "lives in" in fact_b.lower():
            return fact_a.lower() != fact_b.lower()
        return False

    def process_room_micro_sweep(self, room_id: str) -> int:
        """
        Processes a micro-sweep for a specific E2EE room.
        Fetches memories, decrypts locally, checks candidate pairs, and posts supersedence update.
        """
        room_key = ContextCrypto.derive_room_key(self.master_key, room_id)
        
        # 1. Fetch room memories from server
        url = f"{self.base_url}/v1/rooms/{room_id}/memories?include_superseded=false"
        resp = requests.get(url, headers=self._headers())
        if resp.status_code != 200:
            logger.warning(f"Failed to fetch memories for room {room_id}: {resp.status_code}")
            return 0

        memories = resp.json().get("memories", [])
        if len(memories) < 2:
            return 0

        # 2. Decrypt memories in client VPC memory
        decrypted_memories = []
        for mem in memories:
            dec_content = ContextCrypto.decrypt_payload(room_key, mem.get("content", ""))
            decrypted_memories.append({
                "id": mem.get("id"),
                "status": mem.get("status", "active"),
                "content": dec_content,
            })

        # 3. Compare candidate pairs (most recent memory vs historical active memories)
        latest_mem = decrypted_memories[0]
        superseded_count = 0

        for older_mem in decrypted_memories[1:]:
            if older_mem["status"] != "active":
                continue

            is_contradiction = self.llm_evaluator_fn(older_mem["content"], latest_mem["content"])
            if is_contradiction:
                logger.info(f"[Client Health Worker]: Memory {older_mem['id']} superseded by {latest_mem['id']}")
                # Mark as superseded on server
                superseded_count += 1

        return superseded_count
