import requests
from typing import Dict, Any, List, Optional
from .crypto import ContextCrypto
from .vector_rotation import VectorRotator

class ContextKitClient:
    """
    Official Python Client SDK for ContextKit Context-as-a-Service Platform.
    """

    def __init__(
        self,
        api_key: str,
        project_id: str,
        base_url: str = "http://localhost:8080",
        master_key: Optional[str] = None,
        rotator: Optional[VectorRotator] = None
    ):
        self.api_key = api_key
        self.project_id = project_id
        self.base_url = base_url.rstrip('/')
        self.master_key = master_key
        self.rotator = rotator

    def _headers(self) -> Dict[str, str]:
        return {
            "Content-Type": "application/json",
            "X-API-Key": self.api_key,
            "X-Project-ID": self.project_id,
        }

    def create_room(self, description: str = "", external_user_id: Optional[str] = None) -> Dict[str, Any]:
        """
        Creates a new conversation room.
        """
        url = f"{self.base_url}/v1/rooms"
        payload = {
            "description": description,
            "external_user_id": external_user_id
        }
        resp = requests.post(url, json=payload, headers=self._headers())
        resp.raise_for_status()
        return resp.json()

    def send_message(
        self,
        room_id: str,
        role: str,
        content: str,
        is_encrypted: bool = False
    ) -> Dict[str, Any]:
        """
        Sends a message to a room. If is_encrypted=True, encrypts payload with AES-256-GCM prior to sending.
        """
        url = f"{self.base_url}/v1/rooms/{room_id}/messages"
        
        final_content = content
        if is_encrypted and self.master_key:
            room_key = ContextCrypto.derive_room_key(self.master_key, room_id)
            final_content = ContextCrypto.encrypt_payload(room_key, content)

        payload = {
            "role": role,
            "content": final_content,
            "is_encrypted": is_encrypted
        }
        resp = requests.post(url, json=payload, headers=self._headers())
        resp.raise_for_status()
        return resp.json()

    def get_context(self, room_id: str, query: str) -> Dict[str, Any]:
        """
        Retrieves context package for a room. Decrypts encrypted memories transparently if master_key is provided.
        """
        url = f"{self.base_url}/v1/rooms/{room_id}/context"
        payload = {"query": query}
        resp = requests.post(url, json=payload, headers=self._headers())
        resp.raise_for_status()
        data = resp.json()

        if self.master_key and data.get("memories"):
            room_key = ContextCrypto.derive_room_key(self.master_key, room_id)
            for mem in data["memories"]:
                if mem.get("is_encrypted") or mem.get("content", "").startswith("enc_"):
                    mem["content"] = ContextCrypto.decrypt_payload(room_key, mem["content"])

        return data
